package cmd

import (
	"fmt"
	"os"

	"github.com/manicminer/workload-identity-adal-bridge/identityclient"
	"github.com/spf13/cobra"
)

var clientCmd = &cobra.Command{
	Use:   "client",
	Short: "Run the API client",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var tokenCmd = &cobra.Command{
	Use:   "token",
	Short: "Acquire an access token",
	RunE: func(cmd *cobra.Command, args []string) error {
		resource, err := cmd.Flags().GetString("resource")
		if err != nil {
			return err
		}

		scope, err := cmd.Flags().GetString("scope")
		if err != nil {
			return err
		}

		clientId, err := cmd.Flags().GetString("client-id")
		if err != nil {
			return err
		}

		tokResp, err := identityclient.AccessToken(resource, scope, clientId)
		if err != nil {
			return err
		}

		fmt.Println(tokResp.AccessToken)
		return nil
	},
}

func init() {
	tokenCmd.Flags().String("client-id", "", "client ID")
	if err := tokenCmd.MarkFlagRequired("client-id"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	tokenCmd.Flags().String("resource", "", "resource URL")
	tokenCmd.Flags().String("scope", "", "scope URI")
	tokenCmd.MarkFlagsOneRequired("resource", "scope")
	tokenCmd.MarkFlagsMutuallyExclusive("resource", "scope")
	clientCmd.AddCommand(tokenCmd)
}
