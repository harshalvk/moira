package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const namespace = "moira"

var (
	SchedulingLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: namespace,
		Name:      "scheduling_attempt_duration_seconds",
		Help:      "Time from a pod being observed as unscheduled to a bind attempt completing.",
		Buckets:   []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
	})

	SchedulingAttempts = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "scheduling_attempts_total",
		Help:      "Total scheduling attempts, labeled by outcome.",
	}, []string{"result"})

	PluginDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: namespace,
		Name:      "plugin_duration_seconds",
		Help:      "Time spent in each plugin's filter or score call.",
		Buckets:   []float64{0.0001, 0.0005, 0.001, 0.005, 0.01, 0.05, 0.1, 0.5},
	}, []string{"plugin", "extension_point"})

	IsLeader = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "is_leader",
		Help:      "1 if this replica currently holds scheduler leadership, 0 otherwise",
	})
)
