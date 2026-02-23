package discovery

import "github.com/ppiankov/spectrehub/internal/models"

// ToolExecInfo describes how to invoke a spectre tool.
type ToolExecInfo struct {
	Binary      string   // executable name (looked up in PATH)
	Subcommand  string   // audit/scan subcommand
	JSONFlag    string   // flag to produce JSON output
	EnvVars     []string // env vars that signal the tool has a target
	ConfigFiles []string // config files that signal the tool is configured
}

// Registry is the single source of truth for how to invoke each spectre tool.
var Registry = map[models.ToolType]ToolExecInfo{
	models.ToolVault: {
		Binary:      "vaultspectre",
		Subcommand:  "scan",
		JSONFlag:    "--format json",
		EnvVars:     []string{"VAULT_ADDR", "VAULT_TOKEN"},
		ConfigFiles: nil,
	},
	models.ToolS3: {
		Binary:      "s3spectre",
		Subcommand:  "scan",
		JSONFlag:    "--format json",
		EnvVars:     []string{"AWS_PROFILE", "AWS_REGION", "AWS_ACCESS_KEY_ID"},
		ConfigFiles: nil,
	},
	models.ToolKafka: {
		Binary:      "kafkaspectre",
		Subcommand:  "audit",
		JSONFlag:    "--format json",
		EnvVars:     nil,
		ConfigFiles: []string{"~/.kafkaspectre.yaml", ".kafkaspectre.yaml"},
	},
	models.ToolClickHouse: {
		Binary:      "clickspectre",
		Subcommand:  "analyze",
		JSONFlag:    "--format json",
		EnvVars:     nil,
		ConfigFiles: []string{".clickspectre.yaml", "~/.clickspectre.yaml"},
	},
	models.ToolPg: {
		Binary:      "pgspectre",
		Subcommand:  "audit",
		JSONFlag:    "--format json",
		EnvVars:     []string{"PGSPECTRE_DB_URL", "DATABASE_URL"},
		ConfigFiles: nil,
	},
	models.ToolMongo: {
		Binary:      "mongospectre",
		Subcommand:  "audit",
		JSONFlag:    "--format json",
		EnvVars:     []string{"MONGODB_URI"},
		ConfigFiles: []string{".mongospectre.yml"},
	},
}
