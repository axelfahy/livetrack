package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Metrics struct {
	// Fetcher
	msgFetchedCounter *prometheus.CounterVec
	// Database manager
	pilotRetrievedCounter prometheus.Counter
	trackRetrievedCounter prometheus.Counter
	trackWrittenCounter   prometheus.Counter
	// Bot
	msgBotSentCounter    prometheus.Counter
	msgBotRemovedCounter prometheus.Counter
}

func InitPrometheus(namespace, subsys string) *Metrics {
	metric := &Metrics{}

	metric.msgFetchedCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Name:      "msgs_fetched_count",
		Help:      "Number of messages fetched from trackers by source",
		Namespace: namespace,
		Subsystem: subsys,
	}, []string{"source"})

	metric.pilotRetrievedCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name:      "pilots_retrieved_count",
		Help:      "Number of calls to retrieve pilots from the database",
		Namespace: namespace,
		Subsystem: subsys,
	})

	metric.trackRetrievedCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name:      "tracks_retrieved_count",
		Help:      "Number of calls to retrieve tracks from the database",
		Namespace: namespace,
		Subsystem: subsys,
	})

	metric.trackWrittenCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name:      "tracks_written_count",
		Help:      "Number of tracks written to the database",
		Namespace: namespace,
		Subsystem: subsys,
	})

	metric.msgBotSentCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name:      "msgs_bot_sent_count",
		Help:      "Number of messages sent to telegram",
		Namespace: namespace,
		Subsystem: subsys,
	})

	metric.msgBotRemovedCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name:      "msgs_bot_removed_count",
		Help:      "Number of messages removed from telegram",
		Namespace: namespace,
		Subsystem: subsys,
	})

	return metric
}

// Fetcher metrics.
func (m *Metrics) MessageFetchedInc(source string) { m.msgFetchedCounter.WithLabelValues(source).Inc() }

// Database manager.
func (m *Metrics) PilotRetrievedInc() { m.pilotRetrievedCounter.Inc() }
func (m *Metrics) TrackRetrievedInc() { m.trackRetrievedCounter.Inc() }
func (m *Metrics) TrackWrittenInc()   { m.trackWrittenCounter.Inc() }

// Telegram bot metrics.
func (m *Metrics) MessageSentInc()    { m.msgBotSentCounter.Inc() }
func (m *Metrics) MessageRemovedInc() { m.msgBotRemovedCounter.Inc() }
