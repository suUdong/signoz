package main

import (
	"log/slog"

	"go.uber.org/zap" //nolint:depguard

	"github.com/SigNoz/signoz/cmd"
	"github.com/SigNoz/signoz/pkg/instrumentation"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

func main() {
	// initialize logger for logging in the cmd/ package. This logger is different from the logger used in the application.
	logger := instrumentation.NewLogger(instrumentation.Config{Logs: instrumentation.LogsConfig{Level: slog.LevelInfo}})

	// Register the JSONL audit sink before any server commands are wired up.
	// The operator may override the path by symlinking or redirecting the file.
	// Default rotation threshold: 50 MiB (ruletypes.DefaultPilotAuditJSONLMaxSizeBytes).
	if jsonlSink, err := ruletypes.NewPilotAuditEventJSONLSink(
		"var/audit/pilot-events.jsonl",
		ruletypes.DefaultPilotAuditJSONLMaxSizeBytes,
	); err != nil {
		zap.L().Warn("pilot audit JSONL sink init failed; falling back to nop", zap.Error(err)) //nolint:depguard
	} else {
		ruletypes.RegisterPilotAuditEventSink(jsonlSink)
	}

	// register a list of commands to the root command
	registerServer(cmd.RootCmd, logger)
	cmd.RegisterGenerate(cmd.RootCmd, logger)

	cmd.Execute(logger)
}
