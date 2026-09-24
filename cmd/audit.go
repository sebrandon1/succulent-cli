package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/sebrandon1/succulent-cli/lib"
	"github.com/spf13/cobra"
)

var sharedAuditLog *lib.AuditLog

func auditLogPath() string {
	if path := os.Getenv("SUCCULENT_AUDIT_LOG"); path != "" {
		return path
	}

	return filepath.Join(configDir(), "audit.log")
}

func userID() string {
	if id := os.Getenv("USER"); id != "" {
		return id
	}
	if id := os.Getenv("USERNAME"); id != "" {
		return id
	}
	if current, err := user.Current(); err == nil && current.Username != "" {
		return current.Username
	}

	return "unknown"
}

func auditParameters(values ...string) map[string]string {
	parameters := make(map[string]string, len(values)/2)
	for i := 0; i+1 < len(values); i += 2 {
		if values[i+1] != "" {
			parameters[values[i]] = values[i+1]
		}
	}
	if len(parameters) == 0 {
		return nil
	}

	return parameters
}

func currentAuditLog() (*lib.AuditLog, error) {
	if sharedAuditLog != nil {
		return sharedAuditLog, nil
	}

	return lib.NewAuditLog(auditLogPath())
}

func runAuditedOperation(operation string, parameters map[string]string, action func() error) error {
	return runAuditedOperationForEnvironment(envName, operation, parameters, action)
}

func runAuditedOperationForEnvironment(environment, operation string, parameters map[string]string, action func() error) error {
	auditLog, err := currentAuditLog()
	if err != nil {
		return fmt.Errorf("initializing audit log: %w", err)
	}

	operationErr := action()
	entry := lib.AuditEntry{
		Timestamp:   time.Now().UTC(),
		UserID:      userID(),
		Operation:   operation,
		Environment: environment,
		Parameters:  parameters,
		Result:      "success",
	}
	if operationErr != nil {
		entry.Result = "failure"
		entry.Error = operationErr.Error()
	}

	if err := auditLog.Record(entry); err != nil {
		if operationErr == nil {
			return fmt.Errorf("%s succeeded, but recording audit log: %w", operation, err)
		}
		return errors.Join(operationErr, fmt.Errorf("recording failed-operation audit entry: %w", err))
	}

	return operationErr
}

var historyLimit int

var configHistoryCmd = &cobra.Command{
	Use:   "history",
	Short: "Show recent operation audit history",
	Example: `  succulent-cli config history
  succulent-cli config history --limit 25`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if historyLimit <= 0 {
			return fmt.Errorf("--limit must be greater than zero")
		}

		auditLog, err := currentAuditLog()
		if err != nil {
			return fmt.Errorf("initializing audit log: %w", err)
		}

		entries, err := auditLog.ReadLast(historyLimit)
		if err != nil {
			return fmt.Errorf("reading audit history: %w", err)
		}
		if len(entries) == 0 {
			fmt.Println("No audit history found")
			return nil
		}

		writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(writer, "TIMESTAMP\tUSER\tOPERATION\tENVIRONMENT\tRESULT\tDETAILS")
		for _, entry := range entries {
			fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\t%s\n",
				entry.Timestamp.Format(time.RFC3339), entry.UserID, entry.Operation,
				entry.Environment, entry.Result, formatAuditDetails(entry))
		}

		return writer.Flush()
	},
}

func formatAuditDetails(entry lib.AuditEntry) string {
	keys := make([]string, 0, len(entry.Parameters))
	for key := range entry.Parameters {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	details := make([]string, 0, len(keys)+1)
	for _, key := range keys {
		details = append(details, key+"="+strings.Join(strings.Fields(entry.Parameters[key]), " "))
	}
	if entry.Error != "" {
		details = append(details, "error="+strings.Join(strings.Fields(entry.Error), " "))
	}

	return strings.Join(details, "; ")
}

func init() {
	configHistoryCmd.Flags().IntVar(&historyLimit, "limit", 10, "Number of recent audit entries to show")
}
