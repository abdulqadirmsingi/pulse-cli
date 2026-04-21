package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/abdulqadirmsingi/pulse-cli/internal/ui"
	"github.com/spf13/cobra"
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "remove Pulse from your machine 👋",
	Long:  "Removes the shell hook from your config and deletes the binary.",
	RunE:  runUninstall,
}

func init() {
	rootCmd.AddCommand(uninstallCmd)
}

func runUninstall(_ *cobra.Command, _ []string) error {
	fmt.Println()
	fmt.Println(ui.Title.Render("👋 uninstalling Pulse..."))
	fmt.Println()

	shell := detectShell()
	home, _ := os.UserHomeDir()
	var rcFile string
	switch shell {
	case "zsh":
		rcFile = filepath.Join(home, ".zshrc")
	default:
		rcFile = filepath.Join(home, ".bashrc")
	}

	content, err := os.ReadFile(rcFile)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("reading %s: %w", rcFile, err)
	}
	if strings.Contains(string(content), "Pulse shell hook") {
		cleaned := removeHookBlock(string(content))
		if err := os.WriteFile(rcFile, []byte(cleaned), 0644); err != nil {
			return fmt.Errorf("updating %s: %w", rcFile, err)
		}
		printInitStep("✓", "removed shell hook from "+rcFile)
	} else {
		printInitStep("~", "no hook found in "+rcFile)
	}

	binaryPath, _ := os.Executable()
	removedBinary := false
	for _, candidate := range binaryLocations(binaryPath) {
		if err := os.Remove(candidate); err == nil {
			printInitStep("✓", "removed binary at "+candidate)
			removedBinary = true
			break
		}
	}
	if !removedBinary {
		printInitStep("~", "binary not found in standard locations — remove manually if needed")
	}

	dataDir := filepath.Join(home, ".devpulse")
	fmt.Println()
	fmt.Println(ui.Muted.Render("  ur command history is still at: " + dataDir))
	fmt.Println(ui.Muted.Render("  delete it with:  rm -rf " + dataDir))

	fmt.Println()
	cyan := lipgloss.NewStyle().Foreground(lipgloss.Color("#00D4FF"))
	fmt.Println(ui.Box.Render(
		ui.Muted.Render("Pulse has been removed 👋")+"\n\n"+
			ui.Muted.Render("if u ever wanna come back:")+"\n"+
			cyan.Render("  go install github.com/abdulqadirmsingi/pulse-cli@latest")+"\n"+
			cyan.Render("  pulse init"),
	))
	fmt.Println()

	fmt.Println(ui.Muted.Render("  reload ur shell to finish:"))
	fmt.Println("  " + cyan.Render("source "+rcFile))
	fmt.Println()
	return nil
}

func binaryLocations(selfPath string) []string {
	home, _ := os.UserHomeDir()
	locs := []string{
		selfPath,
		filepath.Join(home, ".local", "bin", "pulse"),
		"/usr/local/bin/pulse",
		filepath.Join(home, "bin", "pulse"),
	}
	// deduplicate
	seen := map[string]bool{}
	var out []string
	for _, l := range locs {
		if l != "" && !seen[l] {
			seen[l] = true
			out = append(out, l)
		}
	}
	return out
}
