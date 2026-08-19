package cmd

import (
	"crypto/tls"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/sebrandon1/succulent-cli/lib"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	healthCheckTimeout             = 10 * time.Second
	certExpirationWarningDays      = 30
	certExpirationWarningThreshold = certExpirationWarningDays * 24 * time.Hour
)

var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check connectivity to succulent server",
	Long: `Verify connectivity to the succulent server, test TLS configuration,
and check basic API availability without side effects.`,
	Example: `  succulent-cli health
  succulent-cli health --url https://succulent.example.com
  succulent-cli health --verify-ssl`,
	RunE: func(_ *cobra.Command, _ []string) error {
		baseURL := viper.GetString("url")
		verifySSL := viper.GetBool("verify_ssl")
		caCert := viper.GetString("ca_cert")

		fmt.Printf("Checking health of: %s\n", baseURL)

		if verifySSL {
			fmt.Println("TLS verification: enabled")
		} else {
			fmt.Println("TLS verification: disabled (insecure)")
		}

		if caCert != "" {
			fmt.Printf("CA certificate: %s\n", caCert)
		}

		fmt.Println()

		start := time.Now()

		client, err := lib.NewClientWithTimeout(baseURL, !verifySSL, caCert, healthCheckTimeout)
		if err != nil {
			fmt.Printf("❌ Failed to create client: %v\n", err)
			os.Exit(1)
		}

		client.Logger = slog.Default()

		resp, err := client.HTTPClient.Get(baseURL)
		if err != nil {
			elapsed := time.Since(start)
			fmt.Printf("❌ Health check failed after %v\n", elapsed.Round(time.Millisecond))
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		defer func() {
			_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, lib.MaxResponseSize))
			_ = resp.Body.Close()
		}()

		elapsed := time.Since(start)

		fmt.Printf("✅ Server is reachable\n")
		fmt.Printf("Response time: %v\n", elapsed.Round(time.Millisecond))
		fmt.Printf("Status code: %d %s\n", resp.StatusCode, http.StatusText(resp.StatusCode))

		if resp.TLS != nil {
			fmt.Println("\nTLS Information:")
			fmt.Printf("  Version: %s\n", tlsVersionString(resp.TLS.Version))
			fmt.Printf("  Cipher suite: %s\n", tls.CipherSuiteName(resp.TLS.CipherSuite))

			if len(resp.TLS.PeerCertificates) > 0 {
				cert := resp.TLS.PeerCertificates[0]
				fmt.Printf("  Certificate subject: %s\n", cert.Subject)
				fmt.Printf("  Certificate issuer: %s\n", cert.Issuer)
				fmt.Printf("  Valid from: %s\n", cert.NotBefore.Format(time.RFC3339))
				fmt.Printf("  Valid until: %s\n", cert.NotAfter.Format(time.RFC3339))

				now := time.Now()
				if now.After(cert.NotAfter) {
					fmt.Printf("  ⚠️  Certificate expired!\n")
				} else if now.Add(certExpirationWarningThreshold).After(cert.NotAfter) {
					daysLeft := int(cert.NotAfter.Sub(now).Hours() / 24)
					fmt.Printf("  ⚠️  Certificate expires in %d days\n", daysLeft)
				}
			}
		}

		if resp.StatusCode >= 400 {
			fmt.Printf("\n⚠️  Server returned error status code: %d\n", resp.StatusCode)
			os.Exit(1)
		}

		fmt.Println("\n✅ Health check passed")

		return nil
	},
}

func tlsVersionString(version uint16) string {
	switch version {
	case tls.VersionTLS10:
		return "TLS 1.0"
	case tls.VersionTLS11:
		return "TLS 1.1"
	case tls.VersionTLS12:
		return "TLS 1.2"
	case tls.VersionTLS13:
		return "TLS 1.3"
	default:
		return fmt.Sprintf("Unknown (0x%04x)", version)
	}
}

func init() {
	rootCmd.AddCommand(healthCmd)
}
