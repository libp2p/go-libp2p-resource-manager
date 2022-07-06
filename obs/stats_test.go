package obs_test

import (
	"testing"

	rcmgr "github.com/libp2p/go-libp2p-resource-manager"
	"github.com/libp2p/go-libp2p-resource-manager/obs"
)

func TestTraceReporterStartAndClose(t *testing.T) {
	rcmgr, err := rcmgr.NewResourceManager(rcmgr.NewFixedLimiter(rcmgr.DefaultLimits.AutoScale()), rcmgr.WithTraceReporter(obs.StatsTraceReporter{}))
	if err != nil {
		t.Fatal(err)
	}
	defer rcmgr.Close()
}
