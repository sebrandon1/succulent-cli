package cmd

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/sebrandon1/succulent-cli/lib"
)

const serverVersionCheckTimeout = 2 * time.Second

func warnOnVersionMismatch(ctx context.Context, client *lib.Client, cliVersion string, warnings io.Writer) {
	if cliVersion == "" || cliVersion == "dev" || cliVersion == "devel" {
		return
	}

	checkCtx, cancel := context.WithTimeout(ctx, serverVersionCheckTimeout)
	defer cancel()

	serverVersion, err := client.GetServerVersion(checkCtx)
	if err != nil {
		return
	}

	_, warning := lib.CheckVersionCompatibility(cliVersion, *serverVersion)
	if warning != "" {
		_, _ = fmt.Fprintf(warnings, "⚠️  %s\n", warning)
	}
}
