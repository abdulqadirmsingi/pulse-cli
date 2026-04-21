package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/abdulqadirmsingi/pulse-cli/internal/config"
	"github.com/abdulqadirmsingi/pulse-cli/internal/db"
	"github.com/abdulqadirmsingi/pulse-cli/internal/ui"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "verify your setup is working correctly 🩺",
	Long:  "Verifies your setup: data dir, database, and shell hook installation.",
	RunE:  runDoctor,
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}

func runDoctor(_ *cobra.Command, _ []string) error {
	fmt.Println()
	fmt.Println(ui.Title.Render("🩺  pulse doctor"))
	fmt.Println()

	allGood := true
	pass := func(msg string) { fmt.Printf("  %s  %s\n", ui.Success.Render("✓"), ui.Muted.Render(msg)) }
	fail := func(msg string) {
		fmt.Printf("  %s  %s\n", ui.Err.Render("✗"), ui.Err.Render(msg))
		allGood = false
	}

	cfg, err := config.Load()
	if err != nil {
		fail("config error: " + err.Error())
		return nil
	}
	pass("config OK")

	if _, err := os.Stat(cfg.DataDir); err != nil {
		fail("data dir missing — run: pulse init")
	} else {
		pass("data dir: " + cfg.DataDir)
	}

	database, dbErr := db.Open(cfg.DBPath)
	if dbErr != nil {
		fail("database error: " + dbErr.Error())
	} else {
		stats, _ := database.GetStats(36500) // effectively all-time
		database.Close()
		pass(fmt.Sprintf("database OK — %s commands recorded total", ui.FormatNumber(stats.TotalCommands)))
	}

	shell := detectShell()
	home, _ := os.UserHomeDir()
	var rcFile string
	switch shell {
	case "zsh":
		rcFile = filepath.Join(home, ".zshrc")
	case "fish":
		rcFile = filepath.Join(home, ".config", "fish", "config.fish")
	default:
		rcFile = filepath.Join(home, ".bashrc")
	}

	content, _ := os.ReadFile(rcFile)
	if strings.Contains(string(content), "Pulse shell hook") {
		// also verify the binary path in the hook matches this binary
		self, _ := os.Executable()
		if self != "" && !strings.Contains(string(content), self) {
			fail("shell hook references a stale binary path — run: pulse init --reinstall")
			allGood = false
		} else {
			pass("shell hook installed in " + rcFile)
		}
	} else {
		fail("shell hook missing — run: pulse init")
		allGood = false
	}

	fmt.Println()
	cyan := lipgloss.NewStyle().Foreground(ui.ColorCyan)

	if !allGood {
		fmt.Println(ui.Err.Render("  fix the issues above, then run: pulse init"))
	} else {
		fmt.Println(ui.Success.Render("  all good — pulse is tracking ur grind 🔥"))
		fmt.Println()
		fmt.Println(ui.Muted.Render("  ⚠️  commands only track in terminals opened AFTER pulse init"))
		fmt.Println(ui.Muted.Render("  if a terminal was open before, either open a new one or run:"))
		fmt.Println("  " + cyan.Render("source "+rcFile))
	}
	fmt.Println()
	return nil
}
