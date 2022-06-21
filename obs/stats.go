package obs

import (
	"context"
	"strings"

	logging "github.com/ipfs/go-log/v2"
	rcmgr "github.com/libp2p/go-libp2p-resource-manager"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

var log = logging.Logger("rcmgrObs")

var (
	systemOutboundConns = stats.Int64("system/outbound/conn", "Number of outbound Connections", stats.UnitDimensionless)
	systemInboundConns  = stats.Int64("system/inbound/conn", "Number of inbound Connections", stats.UnitDimensionless)

	conns = stats.Int64("connections", "Number of Connections", stats.UnitDimensionless)

	peerConns         = stats.Int64("peer/connections", "Number of connections this peer has", stats.UnitDimensionless)
	peerConnsNegative = stats.Int64("peer/connections_negative", "Number of connections this peer had", stats.UnitDimensionless)

	streams = stats.Int64("streams", "Number of Streams", stats.UnitDimensionless)

	peerStreams         = stats.Int64("peer/streams", "Number of streams this peer has", stats.UnitDimensionless)
	peerStreamsNegative = stats.Int64("peer/streams_negative", "Number of streams this peer had", stats.UnitDimensionless)

	memory             = stats.Int64("memory", "Amount of memory reserved as reported to the Resource Manager", stats.UnitDimensionless)
	peerMemory         = stats.Int64("peer/memory", "Amount of memory currently reseved for peer", stats.UnitDimensionless)
	peerMemoryNegative = stats.Int64("peer/memory_negative", "Amount of memory previously reseved for peer", stats.UnitDimensionless)

	connMemory         = stats.Int64("conn/memory", "Amount of memory currently reseved for the connection", stats.UnitDimensionless)
	connMemoryNegative = stats.Int64("conn/memory_negative", "Amount of memory previously reseved for the connection", stats.UnitDimensionless)

	fds = stats.Int64("fds", "Number of fds as reported to the Resource Manager", stats.UnitDimensionless)

	blockedResources = stats.Int64("blocked_resources", "Number of resource requests blocked", stats.UnitDimensionless)
)

var (
	LessThanEq, _ = tag.NewKey("le")
	Direction, _  = tag.NewKey("dir")
	Scope, _      = tag.NewKey("scope")
	Service, _    = tag.NewKey("service")
	Protocol, _   = tag.NewKey("protocol")
	Resource, _   = tag.NewKey("resource")
)

var (
	SystemOutboundConnsView = &view.View{Measure: systemOutboundConns, Aggregation: view.Sum()}
	SystemInboundConnsView  = &view.View{Measure: systemInboundConns, Aggregation: view.Sum()}

	ConnView = &view.View{Measure: conns, Aggregation: view.Sum(), TagKeys: []tag.Key{Direction, Scope}}

	fibLikeDistribution = []float64{
		1.1, 2.1, 3.1, 5.1, 8.1, 13.1, 21.1, 34.1, 55.1, 100.1, 200.1,
	}

	PeerConnsView = &view.View{
		Measure:     peerConns,
		Aggregation: view.Distribution(fibLikeDistribution...),
		TagKeys:     []tag.Key{Direction},
	}
	PeerConnsNegativeView = &view.View{
		Measure:     peerConnsNegative,
		Aggregation: view.Distribution(fibLikeDistribution...),
		TagKeys:     []tag.Key{Direction},
	}

	StreamView             = &view.View{Measure: streams, Aggregation: view.Sum(), TagKeys: []tag.Key{Direction, Scope, Service, Protocol}}
	PeerStreamsView        = &view.View{Measure: peerStreams, Aggregation: view.Distribution(fibLikeDistribution...), TagKeys: []tag.Key{Direction}}
	PeerStreamNegativeView = &view.View{Measure: peerStreamsNegative, Aggregation: view.Distribution(fibLikeDistribution...), TagKeys: []tag.Key{Direction}}

	MemoryView = &view.View{Measure: memory, Aggregation: view.Sum(), TagKeys: []tag.Key{Scope, Service, Protocol}}

	memDistribution = []float64{
		1 << 10, // 1KB
		1 << 12, // 4KB
		1 << 15, // 32KB
		1 << 20, // 1MB
		1 << 25, // 32MB
		1 << 28, // 256MB
		1 << 29, // 512MB
		1 << 30, // 1GB
		1 << 31, // 2GB
		1 << 32, // 4GB
	}
	PeerMemoryView = &view.View{
		Measure:     peerMemory,
		Aggregation: view.Distribution(memDistribution...),
	}
	PeerMemoryNegativeView = &view.View{
		Measure:     peerMemoryNegative,
		Aggregation: view.Distribution(memDistribution...),
	}

	// Not setup yet. Memory isn't attached to a given connection.
	ConnMemoryView = &view.View{
		Measure:     connMemory,
		Aggregation: view.Distribution(memDistribution...),
	}
	ConnMemoryNegativeView = &view.View{
		Measure:     connMemoryNegative,
		Aggregation: view.Distribution(memDistribution...),
	}

	FDsView = &view.View{Measure: fds, Aggregation: view.Sum(), TagKeys: []tag.Key{Scope}}

	BlockedResourcesView = &view.View{
		Measure:     blockedResources,
		Aggregation: view.Sum(),
		TagKeys:     []tag.Key{Scope, Resource},
	}
)

var DefaultViews []*view.View = []*view.View{
	ConnView,
	PeerConnsView,
	PeerConnsNegativeView,
	FDsView,

	StreamView,
	PeerStreamsView,
	PeerStreamNegativeView,

	MemoryView,
	PeerMemoryView,
	PeerMemoryNegativeView,

	BlockedResourcesView,
}

// StatsTraceReporter reports stats on the resource manager using its traces.
type StatsTraceReporter struct{}

func NewStatsTraceReporter() (StatsTraceReporter, error) {
	return StatsTraceReporter{}, nil
}

func (r StatsTraceReporter) ConsumeEvent(evt rcmgr.TraceEvt) {
	ctx := context.Background()

	switch evt.Type {
	case rcmgr.TraceAddStreamEvt, rcmgr.TraceRemoveStreamEvt:
		if p := rcmgr.ParsePeerScopeName(evt.Name); p.Validate() == nil {
			// Aggregated peer stats. Counts how many peers have N number of streams open.
			// Uses two buckets aggregations. One to count how many streams the
			// peer has now. The other to count the negative value, or how many
			// streams did the peer use to have. When looking at the data you
			// take the difference from the two.

			oldStreamsOut := int64(evt.StreamsOut - evt.DeltaOut)
			peerStreamsOut := int64(evt.StreamsOut)
			if oldStreamsOut != peerStreamsOut {
				if oldStreamsOut != 0 {
					stats.RecordWithTags(ctx, []tag.Mutator{tag.Upsert(Direction, "outbound")}, peerStreamsNegative.M(oldStreamsOut))
				}
				if peerStreamsOut != 0 {
					stats.RecordWithTags(ctx, []tag.Mutator{tag.Upsert(Direction, "outbound")}, peerStreams.M(peerStreamsOut))
				}
			}

			oldStreamsIn := int64(evt.StreamsIn - evt.DeltaIn)
			peerStreamsIn := int64(evt.StreamsIn)
			if oldStreamsIn != peerStreamsIn {
				if oldStreamsIn != 0 {
					stats.RecordWithTags(ctx, []tag.Mutator{tag.Upsert(Direction, "inbound")}, peerStreamsNegative.M(oldStreamsIn))
				}
				if peerStreamsIn != 0 {
					stats.RecordWithTags(ctx, []tag.Mutator{tag.Upsert(Direction, "inbound")}, peerStreams.M(peerStreamsIn))
				}
			}
		} else {
			var tags []tag.Mutator
			if rcmgr.IsSystemScope(evt.Name) || rcmgr.IsTransientScope(evt.Name) {
				tags = append(tags, tag.Upsert(Scope, evt.Name))
			} else if svc := rcmgr.ParseServiceScopeName(evt.Name); svc != "" {
				tags = append(tags, tag.Upsert(Scope, "service"), tag.Upsert(Service, svc))
			} else if proto := rcmgr.ParseProtocolScopeName(evt.Name); proto != "" {
				tags = append(tags, tag.Upsert(Scope, "protocol"), tag.Upsert(Protocol, proto))
			} else {
				// Not measuring connscope, servicepeer and protocolpeer. Lots of data, and
				// you can use aggregated peer stats + service stats to infer
				// this.
				break
			}

			if evt.DeltaOut != 0 {
				stats.RecordWithTags(
					ctx,
					append([]tag.Mutator{tag.Upsert(Direction, "outbound")}, tags...),
					streams.M(int64(evt.DeltaOut)),
				)
			}

			if evt.DeltaIn != 0 {
				stats.RecordWithTags(
					ctx,
					append([]tag.Mutator{tag.Upsert(Direction, "inbound")}, tags...),
					streams.M(int64(evt.DeltaIn)),
				)
			}
		}

	case rcmgr.TraceAddConnEvt, rcmgr.TraceRemoveConnEvt:
		if p := rcmgr.ParsePeerScopeName(evt.Name); p.Validate() == nil {
			// Aggregated peer stats. Counts how many peers have N number of connections.
			// Uses two buckets aggregations. One to count how many streams the
			// peer has now. The other to count the negative value, or how many
			// conns did the peer use to have. When looking at the data you
			// take the difference from the two.

			oldConnsOut := int64(evt.ConnsOut - evt.DeltaOut)
			connsOut := int64(evt.ConnsOut)
			if oldConnsOut != connsOut {
				if oldConnsOut != 0 {
					stats.RecordWithTags(ctx, []tag.Mutator{tag.Upsert(Direction, "outbound")}, peerConnsNegative.M(oldConnsOut))
				}
				if connsOut != 0 {
					stats.RecordWithTags(ctx, []tag.Mutator{tag.Upsert(Direction, "outbound")}, peerConns.M(connsOut))
				}
			}

			oldConnsIn := int64(evt.ConnsIn - evt.DeltaIn)
			connsIn := int64(evt.ConnsIn)
			if oldConnsIn != connsIn {
				if oldConnsIn != 0 {
					stats.RecordWithTags(ctx, []tag.Mutator{tag.Upsert(Direction, "inbound")}, peerConnsNegative.M(oldConnsIn))
				}
				if connsIn != 0 {
					stats.RecordWithTags(ctx, []tag.Mutator{tag.Upsert(Direction, "inbound")}, peerConns.M(connsIn))
				}
			}
		} else {
			var tags []tag.Mutator
			if rcmgr.IsSystemScope(evt.Name) || rcmgr.IsTransientScope(evt.Name) {
				tags = append(tags, tag.Upsert(Scope, evt.Name))
			} else if rcmgr.IsConnScope(evt.Name) {
				// Not measuring this. I don't think it's useful.
				break
			} else {
				// There shouldn't be anything here. But we keep going so the metrics will tell us if we're wrong (scope="")
				log.Debugf("unexpected event in stats: %s", evt.Name)
			}

			if evt.DeltaOut != 0 {
				stats.RecordWithTags(
					ctx,
					append([]tag.Mutator{tag.Upsert(Direction, "outbound")}, tags...),
					conns.M(int64(evt.DeltaOut)),
				)
			}

			if evt.DeltaIn != 0 {
				stats.RecordWithTags(
					ctx,
					append([]tag.Mutator{tag.Upsert(Direction, "inbound")}, tags...),
					conns.M(int64(evt.DeltaIn)),
				)
			}

			// Represents the delta in fds
			if evt.Delta != 0 {
				stats.RecordWithTags(
					ctx,
					tags,
					fds.M(int64(evt.Delta)),
				)
			}
		}
	case rcmgr.TraceReserveMemoryEvt, rcmgr.TraceReleaseMemoryEvt:
		if p := rcmgr.ParsePeerScopeName(evt.Name); p.Validate() == nil {
			oldMem := evt.Memory - evt.Delta
			if oldMem != evt.Memory {
				if oldMem != 0 {
					stats.Record(ctx, peerMemoryNegative.M(oldMem))
				}
				if evt.Memory != 0 {
					stats.Record(ctx, peerMemory.M(evt.Memory))
				}
			}
		} else if rcmgr.IsConnScope(evt.Name) {
			oldMem := evt.Memory - evt.Delta
			if oldMem != evt.Memory {
				if oldMem != 0 {
					stats.Record(ctx, connMemoryNegative.M(oldMem))
				}
				if evt.Memory != 0 {
					stats.Record(ctx, connMemory.M(evt.Memory))
				}
			}
		} else {
			var tags []tag.Mutator
			if rcmgr.IsSystemScope(evt.Name) || rcmgr.IsTransientScope(evt.Name) {
				tags = append(tags, tag.Upsert(Scope, evt.Name))
			} else if svc := rcmgr.ParseServiceScopeName(evt.Name); svc != "" {
				tags = append(tags, tag.Upsert(Scope, "service"), tag.Upsert(Service, svc))
			} else if proto := rcmgr.ParseProtocolScopeName(evt.Name); proto != "" {
				tags = append(tags, tag.Upsert(Scope, "protocol"), tag.Upsert(Protocol, proto))
			} else {
				// Not measuring connscope, servicepeer and protocolpeer. Lots of data, and
				// you can use aggregated peer stats + service stats to infer
				// this.
				break
			}

			if evt.Delta != 0 {
				stats.RecordWithTags(ctx, tags, memory.M(int64(evt.Delta)))
			}
		}

	case rcmgr.TraceBlockAddConnEvt, rcmgr.TraceBlockAddStreamEvt, rcmgr.TraceBlockReserveMemoryEvt:
		var resource string
		if evt.Type == rcmgr.TraceBlockAddConnEvt {
			resource = "connection"
		} else if evt.Type == rcmgr.TraceBlockAddStreamEvt {
			resource = "stream"
		} else {
			resource = "memory"
		}

		// Only the top scope. We don't want to get the peerid here.
		scope := strings.SplitN(evt.Name, ":", 2)[0]
		// Drop the connection or stream id
		scope = strings.SplitN(scope, "-", 2)[0]

		tags := []tag.Mutator{tag.Upsert(Scope, scope), tag.Upsert(Resource, resource)}

		if evt.DeltaIn != 0 {
			stats.RecordWithTags(ctx, tags, blockedResources.M(int64(evt.DeltaIn)))
		}

		if evt.DeltaOut != 0 {
			stats.RecordWithTags(ctx, tags, blockedResources.M(int64(evt.DeltaOut)))
		}

		if evt.Delta != 0 {
			stats.RecordWithTags(ctx, tags, blockedResources.M(evt.Delta))
		}
	}
}
