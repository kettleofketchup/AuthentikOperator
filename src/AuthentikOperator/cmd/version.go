package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
"github.com/kettleofketchup/AuthentikOperator/src/AuthentikOperator/version"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long:  `Print the version, commit hash, and build date of AuthentikOperator.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("AuthentikOperator %s\n", version.Version)
		fmt.Printf("  Commit:     %s\n", version.Commit)
		fmt.Printf("  Built:      %s\n", version.BuildDate)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
