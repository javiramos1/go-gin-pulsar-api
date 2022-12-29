package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Error
var Error = promauto.NewCounter(prometheus.CounterOpts{
	Namespace: "go_gin_pulsar",
	Name:      "go_gin_pulsar_api_error",
	Help:      "Ingestion API Error",
})

// Fatal
var Fatal = promauto.NewCounter(prometheus.CounterOpts{
	Namespace: "go_gin_pulsar",
	Name:      "go_gin_pulsar_api_error_fatal",
	Help:      "Ingestion API Fatal Error",
})

// DataIngested
var DataIngested = promauto.NewCounter(prometheus.CounterOpts{
	Namespace: "go_gin_pulsar",
	Name:      "go_gin_pulsar_api_data_ingested",
	Help:      "Ingestion API Data Ingested",
})
