package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "development"

var rootCmd = &cobra.Command{
	Use:   serviceName,
	Short: fmt.Sprintf("%s is a wrapper for ADAL-based managed service identity in AKS clusters using workload identity", serviceFriendlyName),
	Long:  `A wrapper for ADAL-based managed service identity in Azure, for applications that do not support workload identity for Azure Kubernetes Service. It emulates the instance metadata service to vend access tokens using AKS Workload Identity.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func init() {
	rootCmd.AddCommand(clientCmd)
}

func Execute(ctx context.Context) {
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
