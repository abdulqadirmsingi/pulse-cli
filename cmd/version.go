package cmd

import (
	"fmt"

	"github.com/devpulse-cli/devpulse/internal/config"
	"github.com/devpulse-cli/devpulse/internal/ui"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "show the version",
	Run: func(_ *cobra.Command, _ []string) {
		fmt.Printf("%s %s\n",
			ui.Title.Render("Pulse"),
			ui.Muted.Render("v"+config.AppVersion),
		)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
