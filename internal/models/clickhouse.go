package models

import "time"

// ClickHouseReport is the complete output structure from ClickSpectre
type ClickHouseReport struct {
	Metadata               ClickMetadata               `json:"metadata"`
	Tables                 []ClickTable                `json:"tables"`
	Services               []ClickService              `json:"services"`
	Edges                  []ClickEdge                 `json:"edges"`
	Anomalies              []ClickAnomaly              `json:"anomalies"`
	CleanupRecommendations ClickCleanupRecommendations `json:"cleanup_recommendations"`
}

// ClickMetadata contains report generation info
type ClickMetadata struct {
	GeneratedAt          time.Time `json:"generated_at"`
	LookbackDays         int       `json:"lookback_days"`
	ClickHouseHost       string    `json:"clickhouse_host"`
	TotalQueriesAnalyzed uint64    `json:"total_queries_analyzed"`
	AnalysisDuration     string    `json:"analysis_duration"`
	Version              string    `json:"version"`
	K8sResolutionEnabled bool      `json:"k8s_resolution_enabled"`
}

// ClickTable represents a ClickHouse table with usage stats
type ClickTable struct {
	Name         string                 `json:"name"`
	Database     string                 `json:"database"`
	FullName     string                 `json:"full_name"`
	Reads        uint64                 `json:"reads"`
	Writes       uint64                 `json:"writes"`
	LastAccess   time.Time              `json:"last_access"`
	FirstSeen    time.Time              `json:"first_seen"`
	Sparkline    []ClickTimeSeriesPoint `json:"sparkline"`
	Score        float64                `json:"score"`
	Category     string                 `json:"category"` // "active", "unused", "suspect"
	IsMV         bool                   `json:"is_materialized_view"`
	MVDependency []string               `json:"mv_dependencies,omitempty"`
	Engine       string                 `json:"engine,omitempty"`
	IsReplicated bool                   `json:"is_replicated"`
	TotalBytes   uint64                 `json:"total_bytes,omitempty"`
	TotalRows    uint64                 `json:"total_rows,omitempty"`
	CreateTime   time.Time              `json:"create_time,omitempty"`
	ZeroUsage    bool                   `json:"zero_usage"`
}

// ClickService represents a Kubernetes service or raw IP
type ClickService struct {
	IP           string    `json:"ip"`
	K8sService   string    `json:"k8s_service,omitempty"`
	K8sNamespace string    `json:"k8s_namespace,omitempty"`
	K8sPod       string    `json:"k8s_pod,omitempty"`
	TablesUsed   []string  `json:"tables_used"`
	QueryCount   uint64    `json:"query_count"`
	LastSeen     time.Time `json:"last_seen"`
}

// ClickEdge represents a Service→Table relationship
type ClickEdge struct {
	ServiceIP    string    `json:"service"`
	ServiceName  string    `json:"service_name,omitempty"`
	TableName    string    `json:"table"`
	Reads        uint64    `json:"reads"`
	Writes       uint64    `json:"writes"`
	LastActivity time.Time `json:"last_activity"`
}

// ClickAnomaly represents unusual access patterns
type ClickAnomaly struct {
	Type            string    `json:"type"`
	Description     string    `json:"description"`
	Severity        string    `json:"severity"` // "low", "medium", "high"
	AffectedTable   string    `json:"affected_table,omitempty"`
	AffectedService string    `json:"affected_service,omitempty"`
	DetectedAt      time.Time `json:"detected_at"`
}

// ClickTimeSeriesPoint for sparkline visualization
type ClickTimeSeriesPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     uint64    `json:"value"`
}

// ClickCleanupRecommendations groups tables by safety category
type ClickCleanupRecommendations struct {
	ZeroUsageNonReplicated []ClickTableRecommendation `json:"zero_usage_non_replicated"`
	ZeroUsageReplicated    []ClickTableRecommendation `json:"zero_usage_replicated"`
	SafeToDrop             []string                   `json:"safe_to_drop"`
	LikelySafe             []string                   `json:"likely_safe"`
	Keep                   []string                   `json:"keep"`
}

// ClickTableRecommendation contains detailed information about a table for cleanup
type ClickTableRecommendation struct {
	Name         string  `json:"name"`
	Database     string  `json:"database"`
	Engine       string  `json:"engine"`
	IsReplicated bool    `json:"is_replicated"`
	SizeMB       float64 `json:"size_mb"`
	Rows         uint64  `json:"rows"`
}
