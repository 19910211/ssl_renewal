package main

import (
	"os"
	"ssl_reload/acme"
	"ssl_reload/reload_cmd"

	"github.com/spf13/cobra"
)

// TIP <p>To run your code, right-click the code and select <b>Run</b>.</p> <p>Alternatively, click
// the <icon src="AllIcons.Actions.Execute"/> icon in the gutter and select the <b>Run</b> menu item from here.</p>

var rootCmd = &cobra.Command{
	Use:          "ssl_renewal",
	SilenceUsage: true,
	Run:          func(cmd *cobra.Command, args []string) {},
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(-1)
	}
}

func init() {
	rootCmd.AddCommand(acme.StartCmd)
	rootCmd.AddCommand(reload_cmd.StartCmd)
}
