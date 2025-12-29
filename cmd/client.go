package cmd

import (
	"github.com/spf13/cobra"
)

var clientCmd = &cobra.Command{
	Use:   "client",
	Short: "Run the API client",
	Run: func(cmd *cobra.Command, args []string) {
	},
}

var tokenCmd = &cobra.Command{
	Use:   "token",
	Short: "Acquire an access token",
	Run: func(cmd *cobra.Command, args []string) {
	},
}

func init() {
	clientCmd.AddCommand(tokenCmd)
}
