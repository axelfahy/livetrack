package metrics

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/common/version"
)

const (
	Namespace = "livetrack"
	Path      = "/metrics"
)

type Prometheus struct {
	// Fetcher
	msgsFetchedTotal *prometheus.CounterVec
	// Database manager
	pilotsRetrievedTotal prometheus.Counter
	tracksRetrievedTotal prometheus.Counter
	tracksWrittenTotal   prometheus.Counter
	// Bot
	msgsBotSentTotal    prometheus.Counter
	msgsBotRemovedTotal prometheus.Counter
	// Web
	requests *prometheus.SummaryVec
	errors   *prometheus.CounterVec
}

func NewPrometheusMetrics(subsys string) (*Prometheus, *prometheus.Registry, error) {
	prom := &Prometheus{}
	promReg := prometheus.NewPedanticRegistry()
	errs := make([]error, 0)

	programName := os.Args[0]

	buildInfo := prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Namespace: Namespace,
			Name:      "build_info",
			Help:      "Pseudo-metric containing build and version info for the program",
			ConstLabels: prometheus.Labels{
				"version":   version.Version,
				"revision":  version.GetRevision(),
				"branch":    version.Branch,
				"goversion": version.GoVersion,
				"goos":      version.GoOS,
				"goarch":    version.GoArch,
				"tags":      version.GetTags(),
				"program":   programName,
			},
		},
		func() float64 { return 1 },
	)

	prom.msgsFetchedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name:      "msgs_fetched_total",
		Help:      "Number of messages fetched from trackers by source",
		Namespace: Namespace,
		Subsystem: subsys,
	}, []string{"source"})

	prom.pilotsRetrievedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name:      "pilots_retrieved_total",
		Help:      "Number of calls to retrieve pilots from the database",
		Namespace: Namespace,
		Subsystem: subsys,
	})

	prom.tracksRetrievedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name:      "tracks_retrieved_total",
		Help:      "Number of calls to retrieve tracks from the database",
		Namespace: Namespace,
		Subsystem: subsys,
	})

	prom.tracksWrittenTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name:      "tracks_written_total",
		Help:      "Number of tracks written to the database",
		Namespace: Namespace,
		Subsystem: subsys,
	})

	prom.msgsBotSentTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name:      "msgs_bot_sent_total",
		Help:      "Number of messages sent to telegram",
		Namespace: Namespace,
		Subsystem: subsys,
	})

	prom.msgsBotRemovedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name:      "msgs_bot_removed_total",
		Help:      "Number of messages removed from telegram",
		Namespace: Namespace,
		Subsystem: subsys,
	})

	prom.requests = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Name:      "requests_duration_seconds",
		Help:      "Timing of calls by endpoint.",
		Namespace: "http",
	}, []string{"method", "path"})

	prom.errors = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name:      "errors_total",
		Help:      "Number of errors by endpoint.",
		Namespace: "http",
	}, []string{"code", "path"})

	errs = append(errs,
		promReg.Register(collectors.NewBuildInfoCollector()),
		promReg.Register(collectors.NewGoCollector()),
		promReg.Register(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{})),
		promReg.Register(buildInfo),
		promReg.Register(prom.msgsFetchedTotal),
		promReg.Register(prom.pilotsRetrievedTotal),
		promReg.Register(prom.tracksRetrievedTotal),
		promReg.Register(prom.tracksWrittenTotal),
		promReg.Register(prom.msgsBotSentTotal),
		promReg.Register(prom.msgsBotRemovedTotal),
		promReg.Register(prom.requests),
		promReg.Register(prom.errors),
	)

	if err := errors.Join(errs...); err != nil {
		return nil, nil, fmt.Errorf("registering metrics collectors: %w", err)
	}

	return prom, promReg, nil
}

// Fetcher metrics.
func (p *Prometheus) MessageFetched(source string) {
	p.msgsFetchedTotal.WithLabelValues(source).Inc()
}

// Database manager.
func (p *Prometheus) PilotRetrieved() { p.pilotsRetrievedTotal.Inc() }
func (p *Prometheus) TrackRetrieved() { p.tracksRetrievedTotal.Inc() }
func (p *Prometheus) TrackWritten()   { p.tracksWrittenTotal.Inc() }

// Telegram bot metrics.
func (p *Prometheus) MessageSent()    { p.msgsBotSentTotal.Inc() }
func (p *Prometheus) MessageRemoved() { p.msgsBotRemovedTotal.Inc() }

// Web metrics.
func (p *Prometheus) Request(method, endpoint string, duration time.Duration) {
	p.requests.WithLabelValues(method, endpoint).Observe(duration.Seconds())
}

func (p *Prometheus) Error(code int, endpoint string) {
	p.errors.WithLabelValues(strconv.Itoa(code), endpoint).Inc()
}
