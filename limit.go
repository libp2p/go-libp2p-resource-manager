/*
Package rcmgr is the resource manager for go-libp2p. This allows you to track
resources being used throughout your go-libp2p process. As well as making sure
that the process doesn't use more resources than what you define as your
limits. The resource manager only knows about things it is told about, so it's
the responsibility of the user of this library (either go-libp2p or a go-libp2p
user) to make sure they check with the resource manager before actually
allocating the resource.

Resource Management basics – Scopes

The Resource Manager is an object that keeps track of how many resources have
been allocated and what they have been allocated for. A resource is a stream,
connection, or memory reservation. The resources can be allocated for the
system, for a peer, for a protocol, or some combination.

The things that are allocating resources are called "Scopes". A scope can have
a parent scope that limits its resources. A scope can also have child scopes
and it can limit the resources of the child scopes. Scopes form a directed
acyclic graph (DAG) representing resource limits. For example, if scope A is
the parent of scope B, and scope A has a connection limit of 10, then whatever
limit B sets for connections it can never be greater than 10.

The common scopes are:

System scope: This is the root scope and represents all the resources that the
resource manager knows about. It can define the absolute limit of the process.

Transient scope: This is a scope for resources that have yet to be assigned a
peer as an owner. When we first start a connection we are unsure who we're
connecting to, so these connections are limited by the transient (and system)
scope.

Peer scope: This is a scope defined for a specific peer id.

Connection scope: This is a scope for a specific connection.

Allowlist system scope: This is a separate root scope for allowlisted peers. It
lets you define limits for a set of trusted multiaddrs and peers. See
`WithAllowlistedMultiaddrs` and ./docs/allowlist.md for more information on the
allowlist.

Allowlist transient scope: Similar to the above and the normal transient scope
but for allowlisted peers.

Protocol scope: This is a scope that defines limits for a specific protocol id.

There are a couple other scopes that are combination of the above. For example
there is a ProtocolPeer scope that represents the limits for a specific
protocol id for a specific peer.

Resource Management basics – Limits

Limits are what define how much of a resource we are willing to allocate. See
`BaseLimit` for what the limit looks like. These are attached to a scope so
that the scope + limit define the resource constraints of the go-libp2p
process.

Limit scaling

If the same go-libp2p application is run on various different machines, it's
helpful to have limits that scale relative to the specs of the machine. This
is where `ScalingLimitConfig` helps. With `ScalingLimitConfig` and it's
`ScalingLimitConfig.Scale` method you can define what the minimum resources
should be and how they scale up with machine size. Consult `limit_test.go` for
usage examples.

Default limits

By default the resource manager ships with some reasonable scaling limits and
makes a reasonable guess at how much system memory you want to dedicate to the
go-libp2p process. For the default definitions see `DefaultLimits` and
`ScalingLimitConfig.AutoScale()`.

Tweaking Defaults

If the defaults seem mostly okay, but you want to adjust one facet you can do
simply copy the defaults and update the field you want to change. You can
apply changes to a `BaseLimit`, `BaseLimitIncrease`, and `LimitConfig` with
`.Apply`.

Monitoring

Once you have limits set, you'll want to monitor to see if you're running into
your limits often. This could be a sign that you need to raise your limits
(your process is more intensive than you originally thought) or that you need
fix something in your application (surely you don't need over 1000 streams?).

There are OpenCensus metrics that can be hooked up to the resource manager. See
`obs/stats_test.go` for an example on how to enable this, and `DefaultViews` in
`stats.go` for recommended views. These metrics can be hooked up to Prometheus
or any other OpenCensus supported platform.

There is also an included Grafana dashboard to help kickstart your
observability into the resource manager. Find more information about it at
`./obs/grafana-dashboards/README.md`.

How to tune your limits

Once you've set your limits and monitoring you can now tune your limits better.
The `blocked_resources` metric will tell you what was blocked and for what
scope. If you see a steady stream of these blocked requests it means your
resource limits are too low for your usage. If you see a rare sudden spike,
this is okay and it means the resource manager protected you from some anamoly.

How to disable limits

Sometimes disabling all limits is useful when you want to see how much
resources you use during normal operation. You can then use this information to
define your initial limits.

How to debug "resource limit exceeded" errors

If you're seeing a lot of "resource limit exceeded" errors take a look at the
`blocked_resources` metric for some information on what was blocked. Also take
a look at the resources used per stream, and per protocol (the Grafana
Dashboard is ideal for this) and check if you're routinely hitting limits or if
these are rare (but noisy) spikes.

*/
package rcmgr

import (
	"encoding/json"
	"io"

	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
)

// Limit is an object that specifies basic resource limits.
type Limit interface {
	// GetMemoryLimit returns the (current) memory limit.
	GetMemoryLimit() int64
	// GetStreamLimit returns the stream limit, for inbound or outbound streams.
	GetStreamLimit(network.Direction) int
	// GetStreamTotalLimit returns the total stream limit
	GetStreamTotalLimit() int
	// GetConnLimit returns the connection limit, for inbound or outbound connections.
	GetConnLimit(network.Direction) int
	// GetConnTotalLimit returns the total connection limit
	GetConnTotalLimit() int
	// GetFDLimit returns the file descriptor limit.
	GetFDLimit() int
}

// Limiter is the interface for providing limits to the resource manager.
type Limiter interface {
	GetSystemLimits() Limit
	GetTransientLimits() Limit
	GetAllowlistedSystemLimits() Limit
	GetAllowlistedTransientLimits() Limit
	GetServiceLimits(svc string) Limit
	GetServicePeerLimits(svc string) Limit
	GetProtocolLimits(proto protocol.ID) Limit
	GetProtocolPeerLimits(proto protocol.ID) Limit
	GetPeerLimits(p peer.ID) Limit
	GetStreamLimits(p peer.ID) Limit
	GetConnLimits() Limit
}

// NewDefaultLimiterFromJSON creates a new limiter by parsing a json configuration,
// using the default limits for fallback.
func NewDefaultLimiterFromJSON(in io.Reader) (Limiter, error) {
	return NewLimiterFromJSON(in, DefaultLimits.AutoScale())
}

// NewLimiterFromJSON creates a new limiter by parsing a json configuration.
func NewLimiterFromJSON(in io.Reader, defaults LimitConfig) (Limiter, error) {
	cfg, err := readLimiterConfigFromJSON(in, defaults)
	if err != nil {
		return nil, err
	}
	return &fixedLimiter{cfg}, nil
}

func readLimiterConfigFromJSON(in io.Reader, defaults LimitConfig) (LimitConfig, error) {
	var cfg LimitConfig
	if err := json.NewDecoder(in).Decode(&cfg); err != nil {
		return LimitConfig{}, err
	}
	cfg.Apply(defaults)
	return cfg, nil
}

// fixedLimiter is a limiter with fixed limits.
type fixedLimiter struct {
	LimitConfig
}

var _ Limiter = (*fixedLimiter)(nil)

func NewFixedLimiter(conf LimitConfig) Limiter {
	log.Debugw("initializing new limiter with config", "limits", conf)
	return &fixedLimiter{LimitConfig: conf}
}

// BaseLimit is a mixin type for basic resource limits.
type BaseLimit struct {
	Streams         int
	StreamsInbound  int
	StreamsOutbound int
	Conns           int
	ConnsInbound    int
	ConnsOutbound   int
	FD              int
	Memory          int64
}

// Apply overwrites all zero-valued limits with the values of l2
// Must not use a pointer receiver.
func (l *BaseLimit) Apply(l2 BaseLimit) {
	if l.Streams == 0 {
		l.Streams = l2.Streams
	}
	if l.StreamsInbound == 0 {
		l.StreamsInbound = l2.StreamsInbound
	}
	if l.StreamsOutbound == 0 {
		l.StreamsOutbound = l2.StreamsOutbound
	}
	if l.Conns == 0 {
		l.Conns = l2.Conns
	}
	if l.ConnsInbound == 0 {
		l.ConnsInbound = l2.ConnsInbound
	}
	if l.ConnsOutbound == 0 {
		l.ConnsOutbound = l2.ConnsOutbound
	}
	if l.Memory == 0 {
		l.Memory = l2.Memory
	}
	if l.FD == 0 {
		l.FD = l2.FD
	}
}

// BaseLimitIncrease is the increase per GB of system memory.
type BaseLimitIncrease struct {
	Streams         int
	StreamsInbound  int
	StreamsOutbound int
	Conns           int
	ConnsInbound    int
	ConnsOutbound   int
	Memory          int64
	FDFraction      float64
}

// Apply overwrites all zero-valued limits with the values of l2
// Must not use a pointer receiver.
func (l *BaseLimitIncrease) Apply(l2 BaseLimitIncrease) {
	if l.Streams == 0 {
		l.Streams = l2.Streams
	}
	if l.StreamsInbound == 0 {
		l.StreamsInbound = l2.StreamsInbound
	}
	if l.StreamsOutbound == 0 {
		l.StreamsOutbound = l2.StreamsOutbound
	}
	if l.Conns == 0 {
		l.Conns = l2.Conns
	}
	if l.ConnsInbound == 0 {
		l.ConnsInbound = l2.ConnsInbound
	}
	if l.ConnsOutbound == 0 {
		l.ConnsOutbound = l2.ConnsOutbound
	}
	if l.Memory == 0 {
		l.Memory = l2.Memory
	}
	if l.FDFraction == 0 {
		l.FDFraction = l2.FDFraction
	}
}

func (l *BaseLimit) GetStreamLimit(dir network.Direction) int {
	if dir == network.DirInbound {
		return l.StreamsInbound
	} else {
		return l.StreamsOutbound
	}
}

func (l *BaseLimit) GetStreamTotalLimit() int {
	return l.Streams
}

func (l *BaseLimit) GetConnLimit(dir network.Direction) int {
	if dir == network.DirInbound {
		return l.ConnsInbound
	} else {
		return l.ConnsOutbound
	}
}

func (l *BaseLimit) GetConnTotalLimit() int {
	return l.Conns
}

func (l *BaseLimit) GetFDLimit() int {
	return l.FD
}

func (l *BaseLimit) GetMemoryLimit() int64 {
	return l.Memory
}

func (l *fixedLimiter) GetSystemLimits() Limit {
	return &l.System
}

func (l *fixedLimiter) GetTransientLimits() Limit {
	return &l.Transient
}

func (l *fixedLimiter) GetAllowlistedSystemLimits() Limit {
	return &l.AllowlistedSystem
}

func (l *fixedLimiter) GetAllowlistedTransientLimits() Limit {
	return &l.AllowlistedTransient
}

func (l *fixedLimiter) GetServiceLimits(svc string) Limit {
	sl, ok := l.Service[svc]
	if !ok {
		return &l.ServiceDefault
	}
	return &sl
}

func (l *fixedLimiter) GetServicePeerLimits(svc string) Limit {
	pl, ok := l.ServicePeer[svc]
	if !ok {
		return &l.ServicePeerDefault
	}
	return &pl
}

func (l *fixedLimiter) GetProtocolLimits(proto protocol.ID) Limit {
	pl, ok := l.Protocol[proto]
	if !ok {
		return &l.ProtocolDefault
	}
	return &pl
}

func (l *fixedLimiter) GetProtocolPeerLimits(proto protocol.ID) Limit {
	pl, ok := l.ProtocolPeer[proto]
	if !ok {
		return &l.ProtocolPeerDefault
	}
	return &pl
}

func (l *fixedLimiter) GetPeerLimits(p peer.ID) Limit {
	pl, ok := l.Peer[p]
	if !ok {
		return &l.PeerDefault
	}
	return &pl
}

func (l *fixedLimiter) GetStreamLimits(_ peer.ID) Limit {
	return &l.Stream
}

func (l *fixedLimiter) GetConnLimits() Limit {
	return &l.Conn
}
