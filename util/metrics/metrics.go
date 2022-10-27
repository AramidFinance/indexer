package metrics

import "github.com/prometheus/client_golang/prometheus"

// RegisterPrometheusMetrics register all prometheus metrics with the global
// metrics handler.
func RegisterPrometheusMetrics() {
	prometheus.Register(BlockImportTimeSeconds)
	prometheus.Register(ImportedTxnsPerBlock)
	prometheus.Register(ImportedRoundGauge)
	prometheus.Register(BlockUploadTimeSeconds)
	prometheus.Register(PostgresEvalTimeSeconds)
	prometheus.Register(GetAlgodRawBlockTimeSeconds)
	prometheus.Register(ImportedTxns)
	prometheus.Register(ImporterTimeSeconds)
	prometheus.Register(ProcessorTimeSeconds)
	prometheus.Register(ExporterTimeSeconds)
	prometheus.Register(PipelineRetryCount)
}

// Prometheus metric names broken out for reuse.
const (
	BlockImportTimeName      = "import_time_sec"
	BlockUploadTimeName      = "block_upload_time_sec"
	ImportedTxnsPerBlockName = "imported_tx_per_block"
	ImportedRoundGaugeName   = "imported_round"
	PostgresEvalName         = "postgres_eval_time_sec"
	GetAlgodRawBlockTimeName = "get_algod_raw_block_time_sec"
	ImportedTxnsName         = "imported_txns"
	ImporterTimeName         = "importer_time_sec"
	ProcessorTimeName        = "processor_time_sec"
	ExporterTimeName         = "exporter_time_sec"
	PipelineRetryCountName   = "pipeline_retry_count"
)

// AllMetricNames is a reference for all the custom metric names.
var AllMetricNames = []string{
	BlockImportTimeName,
	BlockUploadTimeName,
	ImportedTxnsPerBlockName,
	ImportedRoundGaugeName,
	PostgresEvalName,
	GetAlgodRawBlockTimeName,
	ImporterTimeName,
	ProcessorTimeName,
	ExporterTimeName,
	PipelineRetryCountName,
}

// Initialize the prometheus objects.
var (
	BlockImportTimeSeconds = prometheus.NewSummary(
		prometheus.SummaryOpts{
			Subsystem: "indexer_daemon",
			Name:      BlockImportTimeName,
			Help:      "Total block upload and processing time in seconds.",
		})

	BlockUploadTimeSeconds = prometheus.NewSummary(
		prometheus.SummaryOpts{
			Subsystem: "indexer_daemon",
			Name:      BlockUploadTimeName,
			Help:      "Block upload time in seconds.",
		})

	ImportedTxnsPerBlock = prometheus.NewSummary(
		prometheus.SummaryOpts{
			Subsystem: "indexer_daemon",
			Name:      ImportedTxnsPerBlockName,
			Help:      "Transactions per block.",
		},
	)

	ImportedTxns = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: "indexer_daemon",
			Name:      ImportedTxnsName,
			Help:      "Imported transactions grouped by type",
		},
		[]string{"txn_type"},
	)

	ImportedRoundGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Subsystem: "indexer_daemon",
			Name:      ImportedRoundGaugeName,
			Help:      "The most recent round indexer has imported.",
		})

	PostgresEvalTimeSeconds = prometheus.NewSummary(
		prometheus.SummaryOpts{
			Subsystem: "indexer_daemon",
			Name:      PostgresEvalName,
			Help:      "Time spent calling Eval function in seconds.",
		})

	GetAlgodRawBlockTimeSeconds = prometheus.NewSummary(
		prometheus.SummaryOpts{
			Subsystem: "indexer_daemon",
			Name:      GetAlgodRawBlockTimeName,
			Help:      "Total response time from Algod's raw block endpoint in seconds.",
		})

	ImporterTimeSeconds = prometheus.NewSummary(
		prometheus.SummaryOpts{
			Subsystem: "indexer_daemon",
			Name:      ImporterTimeName,
			Help:      "Time spent at importer step",
		})

	ProcessorTimeSeconds = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Subsystem: "indexer_daemon",
			Name:      ProcessorTimeName,
			Help:      "Time spent running a processor",
		},
		[]string{"processor_name"},
	)

	ExporterTimeSeconds = prometheus.NewSummary(
		prometheus.SummaryOpts{
			Subsystem: "indexer_daemon",
			Name:      ExporterTimeName,
			Help:      "Time spent at exporter step",
		})

	PipelineRetryCount = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Subsystem: "indexer_daemon",
			Name:      PipelineRetryCountName,
			Help:      "Total pipeline retries since last successful run",
		})
)
