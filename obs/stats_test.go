package obs_test

import (
	"testing"
	"time"

	rcmgr "github.com/libp2p/go-libp2p-resource-manager"
	"github.com/libp2p/go-libp2p-resource-manager/obs"
	"go.opencensus.io/stats/view"
)

func TestTraceReporterStartAndClose(t *testing.T) {
	rcmgr, err := rcmgr.NewResourceManager(rcmgr.NewFixedLimiter(rcmgr.DefaultLimits.AutoScale()), rcmgr.WithTraceReporter(obs.StatsTraceReporter{}))
	if err != nil {
		t.Fatal(err)
	}
	defer rcmgr.Close()
}

func TestConsumeEvent(t *testing.T) {
	evt := rcmgr.TraceEvt{
		Type:     rcmgr.TraceBlockAddStreamEvt,
		Name:     "conn-1",
		DeltaOut: 1,
		Time:     time.Now().Format(time.RFC3339Nano),
	}

	err := view.Register(obs.DefaultViews...)
	if err != nil {
		t.Fatal(err)
	}

	str, err := obs.NewStatsTraceReporter()
	if err != nil {
		t.Fatal(err)
	}

	str.ConsumeEvent(evt)
}
