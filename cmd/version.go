package cmd

import (
	"fmt"

	"github.com/abdulqadirmsingi/pulse-cli/internal/config"
	"github.com/abdulqadirmsingi/pulse-cli/internal/ui"
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
