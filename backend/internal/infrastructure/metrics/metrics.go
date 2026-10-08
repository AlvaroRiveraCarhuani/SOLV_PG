package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// SubmissionsTotal counts submissions per course, language, and verdict.
	SubmissionsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "solv_submissions_total",
			Help: "Total number of code submissions processed by SOLV.",
		},
		[]string{"course_id", "language", "verdict"},
	)

	// SubmissionDurationSeconds measures total evaluation duration (compile, run, compare).
	SubmissionDurationSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "solv_submission_duration_seconds",
			Help:    "Histogram of submission evaluation duration in seconds.",
			Buckets: []float64{0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30},
		},
		[]string{"language"},
	)

	// DockerContainersActive tracks the number of currently active Docker containers.
	DockerContainersActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "solv_docker_containers_active",
			Help: "Real-time count of active Docker containers managed by SOLV.",
		},
	)

	// DockerContainerFailuresTotal counts container failures by reason (e.g. oom, timeout, fork_bomb).
	DockerContainerFailuresTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "solv_docker_container_failures_total",
			Help: "Total number of Docker container failures grouped by reason.",
		},
		[]string{"reason"},
	)

	// EvaluatorQueueDepth tracks the depth of the pending evaluation queue.
	EvaluatorQueueDepth = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "solv_evaluator_queue_depth",
			Help: "Current depth of pending evaluation queue in SOLV judge.",
		},
	)

	// HTTPRequestsTotal counts standard HTTP requests.
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "solv_http_requests_total",
			Help: "Total count of HTTP requests processed by SOLV API.",
		},
		[]string{"method", "path", "status"},
	)

	// HTTPRequestDurationSeconds measures latency of HTTP requests.
	HTTPRequestDurationSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "solv_http_request_duration_seconds",
			Help:    "Histogram of HTTP request latency in seconds.",
			Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
		[]string{"method", "path"},
	)
)
