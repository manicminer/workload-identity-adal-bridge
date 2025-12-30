package cmd

import (
	"github.com/manicminer/workload-identity-adal-bridge/service"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Launch the instance metadata service",
	RunE: func(cmd *cobra.Command, args []string) error {
		httpPort := viper.GetInt("http-port")
		tlsEnabled := viper.GetBool("enable-tls")

		var httpsPort *int
		if tlsEnabled {
			httpsPortVal := viper.GetInt("https-port")
			httpsPort = &httpsPortVal
		}

		tlsCertPath := viper.GetString("tls-cert")
		tlsKeyPath := viper.GetString("tls-key")

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
	viper.BindPFlag("http-port", serveCmd.Flags().Lookup("http-port"))

	serveCmd.Flags().Int("https-port", defaultHttpsPort, "HTTPS port to listen on")
	viper.BindPFlag("https-port", serveCmd.Flags().Lookup("https-port"))

	serveCmd.Flags().Bool("enable-tls", false, "enable TLS")
	viper.BindPFlag("enable-tls", serveCmd.Flags().Lookup("enable-tls"))

	serveCmd.Flags().String("tls-cert", defaultTlsCertPath, "path to PEM-encoded TLS certificate")
	viper.BindPFlag("tls-cert", serveCmd.Flags().Lookup("tls-cert"))

	serveCmd.Flags().String("tls-key", defaultTlsKeyPath, "path to PEM-encoded TLS key")
	viper.BindPFlag("tls-key", serveCmd.Flags().Lookup("tls-key"))

	rootCmd.AddCommand(serveCmd)
}
