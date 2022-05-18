package main

import (
	"net/http"
	"os"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/promlog"
	"github.com/prometheus/common/promlog/flag"
	"github.com/prometheus/common/version"
	"github.com/prometheus/exporter-toolkit/web"
	webflag "github.com/prometheus/exporter-toolkit/web/kingpinflag"
	kingpin "gopkg.in/alecthomas/kingpin.v2"

	"github.com/stakefish/peggo-exporter/collector"
)

var (
	peggoRestRpc           = kingpin.Flag("peggo.peggo-rest-rpc", "Peggo REST API URL.").Default("http://localhost:1317").Envar("PEGGO_REST_RPC").String()
	cosmosOrchestratorAddr = kingpin.Flag("peggo.cosmos-orchestrator-address", "Cosmos orchestrator address.").Default("").Envar("COSMOS_ORCHESTRATOR_ADDRESS").String()
	timeout                = kingpin.Flag("peggo.timeout", "Peggo connect timeout.").Default("5s").Envar("PEGGO_TIMEOUT").Duration()
	webConfig              = webflag.AddFlags(kingpin.CommandLine)
	listenAddress          = kingpin.Flag("web.listen-address", "Address to listen on for web interface and telemetry.").Default(":5566").Envar("EXPORTER_WEB_LISTEN_ADDRESS").String()
	metricPath             = kingpin.Flag("web.telemetry-path", "Path under which to expose metrics.").Default("/metrics").Envar("EXPORTER_WEB_TELEMETRY_PATH").String()
	logger                 = log.NewNopLogger()
)

// Metric name parts.
const (
	// The name of the exporter.
	exporterName = "peggo-exporter"
)

func main() {
	kingpin.Version(version.Print(exporterName))
	promlogConfig := &promlog.Config{}
	flag.AddFlags(kingpin.CommandLine, promlogConfig)
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()
	logger = promlog.New(promlogConfig)

	level.Info(logger).Log("msg", "Starting peggo-exporter", "version", version.Info())
	level.Info(logger).Log("msg", "Build context", "context", version.BuildContext())

	prometheus.MustRegister(collector.New(*peggoRestRpc, *cosmosOrchestratorAddr, *timeout, logger))

	var landingPage = []byte(`<html>
	<head><title>Peggo exporter</title></head>
	<body>
	<h1>Peggo exporter</h1>
	<p><a href='` + *metricPath + `'>Metrics</a></p>
	</body>
	</html>
	`)

	http.Handle(*metricPath, promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=UTF-8")
		w.Write(landingPage)
	})

	level.Info(logger).Log("msg", "Listening on address", "address", *listenAddress)
	srv := &http.Server{Addr: *listenAddress}
	if err := web.ListenAndServe(srv, *webConfig, logger); err != nil {
		level.Error(logger).Log("msg", "Error running HTTP server", "err", err)
		os.Exit(1)
	}
}
