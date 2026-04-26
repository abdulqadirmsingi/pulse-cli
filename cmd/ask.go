package cmd

import (
	"fmt"
	"strings"

	"github.com/abdulqadirmsingi/pulse-cli/internal/config"
	"github.com/abdulqadirmsingi/pulse-cli/internal/db"
	"github.com/abdulqadirmsingi/pulse-cli/internal/qa"
	"github.com/abdulqadirmsingi/pulse-cli/internal/ui"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var askCmd = &cobra.Command{
	Use:   "ask [question...]",
	Short: "ask Pulse about your activity 💬",
	Long:  "Answers common questions about your local Pulse data. This is private and free: no AI, no cloud, no API key.",
	Args:  cobra.ArbitraryArgs,
	RunE:  runAsk,
}

func init() {
	rootCmd.AddCommand(askCmd)
}

func runAsk(_ *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	database, err := db.Open(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer database.Close()

	answer, err := qa.AnswerQuestion(database, strings.Join(args, " "))
	if err != nil {
		return fmt.Errorf("answering question: %w", err)
	}
	printAnswer(answer)
	return nil
}

func printAnswer(answer qa.Answer) {
	cyan := lipgloss.NewStyle().Foreground(ui.ColorCyan)
	fmt.Println()
	fmt.Println(ui.Title.Render("💬  " + answer.Title))
	fmt.Println()
	for _, line := range answer.Lines {
		fmt.Println("  " + line)
	}
	if len(answer.Tips) > 0 {
		fmt.Println()
		for _, tip := range answer.Tips {
			fmt.Println("  " + cyan.Render("→") + " " + ui.Muted.Render(tip))
		}
	}
	fmt.Println()
}
