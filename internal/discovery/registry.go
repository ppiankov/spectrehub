package discovery

import "github.com/ppiankov/spectrehub/internal/models"

// ToolExecInfo describes how to invoke a spectre tool.
type ToolExecInfo struct {
	Binary      string   // executable name (looked up in PATH)
	Subcommand  string   // audit/scan subcommand
	JSONFlag    string   // flag to produce JSON output
	EnvVars     []string // env vars that signal the tool has a target
	ConfigFiles []string // config files that signal the tool is configured
	InstallHint string   // install command shown when tool is not found
}

// Registry is the single source of truth for how to invoke each spectre tool.
var Registry = map[models.ToolType]ToolExecInfo{
	models.ToolVault: {
		Binary:      "vaultspectre",
		Subcommand:  "scan",
		JSONFlag:    "--format json",
		EnvVars:     []string{"VAULT_ADDR", "VAULT_TOKEN"},
		ConfigFiles: nil,
		InstallHint: "brew install ppiankov/tap/vaultspectre",
	},
	models.ToolS3: {
		Binary:      "s3spectre",
		Subcommand:  "scan",
		JSONFlag:    "--format json",
		EnvVars:     []string{"AWS_PROFILE", "AWS_REGION", "AWS_ACCESS_KEY_ID"},
		ConfigFiles: nil,
		InstallHint: "brew install ppiankov/tap/s3spectre",
	},
	models.ToolKafka: {
		Binary:      "kafkaspectre",
		Subcommand:  "audit",
		JSONFlag:    "--format json",
		EnvVars:     nil,
		ConfigFiles: []string{"~/.kafkaspectre.yaml", ".kafkaspectre.yaml"},
		InstallHint: "brew install ppiankov/tap/kafkaspectre",
	},
	models.ToolClickHouse: {
		Binary:      "clickspectre",
		Subcommand:  "analyze",
		JSONFlag:    "--format json",
		EnvVars:     nil,
		ConfigFiles: []string{".clickspectre.yaml", "~/.clickspectre.yaml"},
		InstallHint: "brew install ppiankov/tap/clickspectre",
	},
	models.ToolPg: {
		Binary:      "pgspectre",
		Subcommand:  "audit",
		JSONFlag:    "--format json",
		EnvVars:     []string{"PGSPECTRE_DB_URL", "DATABASE_URL"},
		ConfigFiles: nil,
		InstallHint: "brew install ppiankov/tap/pgspectre",
	},
	models.ToolMongo: {
		Binary:      "mongospectre",
		Subcommand:  "audit",
		JSONFlag:    "--format json",
		EnvVars:     []string{"MONGODB_URI"},
		ConfigFiles: []string{".mongospectre.yml"},
		InstallHint: "brew install ppiankov/tap/mongospectre",
	},
	models.ToolAWS: {
		Binary:      "awsspectre",
		Subcommand:  "scan",
		JSONFlag:    "--format json",
		EnvVars:     []string{"AWS_PROFILE", "AWS_REGION", "AWS_ACCESS_KEY_ID"},
		ConfigFiles: []string{".awsspectre.yaml"},
		InstallHint: "brew install ppiankov/tap/awsspectre",
	},
	models.ToolIAM: {
		Binary:      "iamspectre",
		Subcommand:  "aws",
		JSONFlag:    "--format json",
		EnvVars:     []string{"AWS_PROFILE", "AWS_REGION", "AWS_ACCESS_KEY_ID", "GOOGLE_APPLICATION_CREDENTIALS"},
		ConfigFiles: []string{".iamspectre.yaml"},
		InstallHint: "brew install ppiankov/tap/iamspectre",
	},
	models.ToolGCS: {
		Binary:      "gcsspectre",
		Subcommand:  "discover",
		JSONFlag:    "--format json",
		EnvVars:     []string{"GOOGLE_APPLICATION_CREDENTIALS", "GOOGLE_CLOUD_PROJECT", "CLOUDSDK_CORE_PROJECT"},
		ConfigFiles: []string{".gcsspectre.yaml"},
		InstallHint: "brew install ppiankov/tap/gcsspectre",
	},
	models.ToolGCPCompute: {
		Binary:      "gcpspectre",
		Subcommand:  "scan",
		JSONFlag:    "--format json",
		EnvVars:     []string{"GOOGLE_APPLICATION_CREDENTIALS", "CLOUDSDK_CORE_PROJECT"},
		ConfigFiles: []string{".gcpspectre.yaml"},
		InstallHint: "brew install ppiankov/tap/gcpspectre",
	},
}
