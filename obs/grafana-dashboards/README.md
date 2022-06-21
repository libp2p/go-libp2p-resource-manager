# Ready to go Grafana Dashboard

Here are some prebuilt dashboards that you can add to your Grafana instance. To
import follow the Grafana docs [here](https://grafana.com/docs/grafana/latest/dashboards/export-import/#import-dashboard)

## Setup

To make sure you're emitting the correct metrics you'll have to hook up the
Opencensus views that `stats.go` exports. For Prometheus this looks like:

``` go
import (
    // ...
	ocprom "contrib.go.opencensus.io/exporter/prometheus"

	rcmgr "github.com/libp2p/go-libp2p-resource-manager"
	rcmgrObs "github.com/libp2p/go-libp2p-resource-manager/obs"

	"github.com/prometheus/client_golang/prometheus"
	"go.opencensus.io/stats/view"
)

    func SetupResourceManager() (network.ResourceManager, error) {
        // Hook up the trace reporter metrics
        view.Register(rcmgrObs.DefaultViews...)
        ocprom.NewExporter(ocprom.Options{
            Registry:  prometheus.DefaultRegisterer.(*prometheus.Registry),
            Namespace: "rcmgr_trace_metrics",
        })

        str, err := rcmgrObs.NewStatsTraceReporter()
        if err != nil {
            return nil, err
        }

        return rcmgr.NewResourceManager(limiter, rcmgr.WithTraceReporter(str))
    }
```

It should be fairly similar for other exporters. See the [OpenCensus
docs](https://opencensus.io/exporters/supported-exporters/go/) to see how to
export to another exporter.