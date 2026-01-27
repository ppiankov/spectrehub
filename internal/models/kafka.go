package models

// KafkaReport represents the complete output from KafkaSpectre
type KafkaReport struct {
	Summary         *KafkaSummary      `json:"summary"`
	UnusedTopics    []*UnusedTopic     `json:"unused_topics"`
	ActiveTopics    []*ActiveTopic     `json:"active_topics,omitempty"`
	ClusterMetadata *ClusterMetadata   `json:"cluster_metadata"`
}

// KafkaSummary provides high-level audit insights
type KafkaSummary struct {
	// Cluster Overview
	ClusterName         string  `json:"cluster_name"`
	TotalBrokers        int     `json:"total_brokers"`

	// Topic Statistics
	TotalTopicsIncludingInternal int     `json:"total_topics_including_internal"`
	TotalTopics                  int     `json:"total_topics_analyzed"`
	UnusedTopics                 int     `json:"unused_topics"`
	ActiveTopics                 int     `json:"active_topics"`
	InternalTopics               int     `json:"internal_topics_excluded"`
	UnusedPercentage             float64 `json:"unused_percentage"`

	// Partition Statistics
	TotalPartitions         int     `json:"total_partitions"`
	UnusedPartitions        int     `json:"unused_partitions"`
	ActivePartitions        int     `json:"active_partitions"`
	UnusedPartitionsPercent float64 `json:"unused_partitions_percentage"`

	// Consumer Group Statistics
	TotalConsumerGroups int `json:"total_consumer_groups"`

	// Risk Breakdown
	HighRiskCount   int `json:"high_risk_count"`
	MediumRiskCount int `json:"medium_risk_count"`
	LowRiskCount    int `json:"low_risk_count"`

	// Recommendations
	RecommendedCleanup   []string `json:"recommended_cleanup_topics"`
	ClusterHealthScore   string   `json:"cluster_health_score"`
	PotentialSavingsInfo string   `json:"potential_savings_info"`
}

// UnusedTopic represents a topic that has no active consumers
type UnusedTopic struct {
	Name              string            `json:"name"`
	Partitions        int               `json:"partitions"`
	ReplicationFactor int               `json:"replication_factor"`
	RetentionMs       string            `json:"retention_ms"`
	RetentionHuman    string            `json:"retention_human"`
	CleanupPolicy     string            `json:"cleanup_policy"`
	MinInsyncReplicas string            `json:"min_insync_replicas"`
	InterestingConfig map[string]string `json:"interesting_config"`
	Reason            string            `json:"reason"`
	Recommendation    string            `json:"recommendation"`
	Risk              string            `json:"risk"`
	CleanupPriority   int               `json:"cleanup_priority"`
}

// ActiveTopic represents a topic with active consumers
type ActiveTopic struct {
	Name              string   `json:"name"`
	Partitions        int      `json:"partitions"`
	ReplicationFactor int      `json:"replication_factor"`
	ConsumerGroups    []string `json:"consumer_groups"`
	ConsumerCount     int      `json:"consumer_count"`
}

// ClusterMetadata simplified for JSON output
type ClusterMetadata struct {
	Brokers       []BrokerInfo `json:"brokers"`
	ConsumerCount int          `json:"consumer_groups_count"`
	FetchedAt     string       `json:"fetched_at"`
}

// BrokerInfo simplified for JSON output
type BrokerInfo struct {
	ID   int32  `json:"id"`
	Host string `json:"host"`
	Port int32  `json:"port"`
}
