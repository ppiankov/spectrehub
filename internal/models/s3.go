package models

import "time"

// S3Report represents the complete output from S3Spectre
type S3Report struct {
	Tool       string              `json:"tool"`
	Version    string              `json:"version"`
	Timestamp  time.Time           `json:"timestamp"`
	Config     S3Config            `json:"config"`
	Summary    S3Summary           `json:"summary"`
	Buckets    map[string]*BucketAnalysis `json:"buckets"`
	References []S3Reference       `json:"references,omitempty"`
}

// S3Config contains scan configuration for S3Spectre
type S3Config struct {
	RepoPath           string `json:"repo_path"`
	AWSProfile         string `json:"aws_profile,omitempty"`
	AWSRegion          string `json:"aws_region,omitempty"`
	StaleThresholdDays int    `json:"stale_threshold_days"`
}

// S3Summary contains high-level analysis summary
type S3Summary struct {
	TotalBuckets       int      `json:"total_buckets"`
	OKBuckets          int      `json:"ok_buckets"`
	MissingBuckets     []string `json:"missing_buckets,omitempty"`
	UnusedBuckets      []string `json:"unused_buckets,omitempty"`
	MissingPrefixes    []string `json:"missing_prefixes,omitempty"`
	StalePrefixes      []string `json:"stale_prefixes,omitempty"`
	VersionSprawl      []string `json:"version_sprawl,omitempty"`
	LifecycleMisconfig []string `json:"lifecycle_misconfig,omitempty"`
}

// BucketAnalysis contains analysis results for a bucket
type BucketAnalysis struct {
	Name              string           `json:"name"`
	Status            string           `json:"status"`
	Message           string           `json:"message,omitempty"`
	ReferencedInCode  bool             `json:"referenced_in_code"`
	ExistsInAWS       bool             `json:"exists_in_aws"`
	VersioningEnabled bool             `json:"versioning_enabled"`
	LifecycleRules    int              `json:"lifecycle_rules"`
	Prefixes          []PrefixAnalysis `json:"prefixes,omitempty"`
	UnusedScore       *UnusedScore     `json:"unused_score,omitempty"`
}

// PrefixAnalysis contains analysis results for a prefix
type PrefixAnalysis struct {
	Prefix            string `json:"prefix"`
	Status            string `json:"status"`
	Message           string `json:"message,omitempty"`
	ObjectCount       int    `json:"object_count"`
	DaysSinceModified int    `json:"days_since_modified,omitempty"`
}

// UnusedScore contains scoring details for unused bucket detection
type UnusedScore struct {
	Total         int      `json:"total"`
	Reasons       []string `json:"reasons"`
	IsUnused      bool     `json:"is_unused"`
	NotInCode     int      `json:"not_in_code"`
	Empty         int      `json:"empty"`
	OldBucket     int      `json:"old_bucket"`
	DeprecatedTag int      `json:"deprecated_tag"`
}

// S3Reference represents a reference to an S3 resource in code
type S3Reference struct {
	File   string `json:"file"`
	Line   int    `json:"line"`
	Bucket string `json:"bucket"`
	Prefix string `json:"prefix,omitempty"`
}
