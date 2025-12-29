package cmd

import (
	"github.com/manicminer/workload-identity-adal-bridge/service"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Launch the instance metadata service",
	RunE: func(cmd *cobra.Command, args []string) error {
		httpPort, err := cmd.Flags().GetInt("http-port")
		if err != nil {
			return err
		}

		tlsEnabled, err := cmd.Flags().GetBool("enable-tls")
		if err != nil {
			return err
		}

		var httpsPort *int
		if tlsEnabled {
			httpsPortVal, err := cmd.Flags().GetInt("https-port")
			if err != nil {
				return err
			}
			httpsPort = &httpsPortVal
		}

		tlsCertPath, err := cmd.Flags().GetString("tls-cert")
		if err != nil {
			return err
		}
		tlsKeyPath, err := cmd.Flags().GetString("tls-key")
		if err != nil {
			return err
		}

		config := service.Server{
			TLSCertPath:         tlsCertPath,
			TLSKeyPath:          tlsKeyPath,
			ServiceName:         serviceName,
			ServiceFriendlyName: serviceFriendlyName,
			HTTPPort:            &httpPort,
			HTTPSPort:           httpsPort,
		}

		return service.StartServer(cmd.Context(), config)
	},
}

func init() {
	serveCmd.Flags().Int("http-port", defaultHttpPort, "HTTP port to listen on")
	serveCmd.Flags().Int("https-port", defaultHttpsPort, "HTTPS port to listen on")
	serveCmd.Flags().Bool("enable-tls", false, "enable TLS")
	serveCmd.Flags().String("tls-cert", defaultTlsCertPath, "path to PEM-encoded TLS certificate")
	serveCmd.Flags().String("tls-key", defaultTlsKeyPath, "path to PEM-encoded TLS key")
	rootCmd.AddCommand(serveCmd)
}
