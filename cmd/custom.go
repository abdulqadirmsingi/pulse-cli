package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/abdulqadirmsingi/pulse-cli/internal/config"
	"github.com/abdulqadirmsingi/pulse-cli/internal/db"
	"github.com/abdulqadirmsingi/pulse-cli/internal/ui"
	"github.com/spf13/cobra"
)

var validCmdName = regexp.MustCompile(`^[a-z][a-z0-9-]{0,30}$`)

var builtinNames = map[string]bool{
	"c": true, "cmd": true, "dash": true, "doctor": true, "fav": true, "f": true,
	"history": true, "hooks": true, "init": true, "log": true,
	"projects": true, "reset": true, "search": true, "s": true,
	"stats": true, "today": true, "uninstall": true, "update": true,
	"version": true, "vibe": true, "git-check": true, "git-event": true,
	"git-guard": true, "help": true, "pulse": true,
}

var cmdCmd = &cobra.Command{
	Use:     "cmd",
	Aliases: []string{"c"},
	Short:   "create and run your own pulse shortcuts ⚡",
	Long: `Create your own pulse shortcuts for any shell command.

Examples:
  pulse cmd add simulator "open -a Simulator"
  pulse cmd add ios "cd ~/Projects/ios && xed ."
  pulse cmd add serve "python3 -m http.server 8080"

Both forms work when adding — quoted or unquoted:
  pulse cmd add simulator "open -a Simulator"
  pulse cmd add simulator open -a Simulator

Run your shortcuts:
  pulse simulator
  pulse c          list all custom commands
  pulse c rm <name>`,
	RunE: runCmdList,
}

var cmdAddCmd = &cobra.Command{
	Use:                "add <name> <command...>",
	Short:              "create a custom pulse command",
	DisableFlagParsing: true,
	RunE:               runCmdAdd,
}

var cmdRmCmd = &cobra.Command{
	Use:     "rm <name>",
	Aliases: []string{"remove"},
	Short:   "remove a custom pulse command",
	Args:    cobra.ExactArgs(1),
	RunE:    runCmdRm,
}

func init() {
	rootCmd.AddCommand(cmdCmd)
	cmdCmd.AddCommand(cmdAddCmd)
	cmdCmd.AddCommand(cmdRmCmd)
}

func RegisterCustomCommands() {
	cfg, err := config.Load()
	if err != nil {
		return
	}
	database, err := db.Open(cfg.DBPath)
	if err != nil {
		return
	}
	defer database.Close()

	commands, err := database.ListCustomCommands()
	if err != nil {
		return
	}

	for _, c := range commands {
		name := c.Name
		shell := c.Command
		rootCmd.AddCommand(&cobra.Command{
			Use:                name,
			Short:              shell,
			DisableFlagParsing: true,
			SilenceUsage:       true,
			RunE: func(_ *cobra.Command, args []string) error {
				return runCustomCommand(shell, args)
			},
		})
	}
}

func runCustomCommand(shellCmd string, args []string) error {
	full := shellCmd
	if len(args) > 0 {
		quoted := make([]string, len(args))
		for i, a := range args {
			quoted[i] = shellescape(a)
		}
		full += " " + strings.Join(quoted, " ")
	}
	c := exec.Command("sh", "-c", full)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		return err
	}
	return nil
}

func shellescape(s string) string {
	if s == "" {
		return "''"
	}
	for _, r := range s {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' ||
			r == '/' || r == ':' || r == '@' || r == ',') {
			return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
		}
	}
	return s
}

// buildCommandString reconstructs the shell command from cobra args.
// Handles both:
//   - pulse cmd add simulator "open -a Simulator"   → args = ["simulator", "open -a Simulator"]
//   - pulse cmd add simulator open -a Simulator      → args = ["simulator", "open", "-a", "Simulator"]
//   - pulse cmd add simulator open -a "My App"       → args = ["simulator", "open", "-a", "My App"]
func buildCommandString(args []string) string {
	if len(args) == 1 {
		// single token — the whole command was quoted by the shell, use as-is
		return args[0]
	}
	// multiple tokens — re-quote any that contain spaces so the stored
	// command string is safe to pass to sh -c later
	parts := make([]string, len(args))
	for i, a := range args {
		if strings.ContainsAny(a, " \t") {
			parts[i] = "'" + strings.ReplaceAll(a, "'", `'\''`) + "'"
		} else {
			parts[i] = a
		}
	}
	return strings.Join(parts, " ")
}

func openCmdDB() (*db.DB, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	return db.Open(cfg.DBPath)
}

func runCmdList(_ *cobra.Command, _ []string) error {
	database, err := openCmdDB()
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer database.Close()

	commands, err := database.ListCustomCommands()
	if err != nil {
		return fmt.Errorf("loading commands: %w", err)
	}

	fmt.Println()
	cyan := lipgloss.NewStyle().Foreground(ui.ColorCyan)

	if len(commands) == 0 {
		fmt.Println(ui.Title.Render("⚡  custom commands  ·  none yet"))
		fmt.Println()
		fmt.Println("  " + ui.Muted.Render("make ur own pulse shortcuts:"))
		fmt.Println()
		fmt.Println("  " + cyan.Render(`pulse cmd add <name> "<command>"`))
		fmt.Println()
		fmt.Println("  " + ui.Muted.Render("example:"))
		fmt.Println("  " + cyan.Render(`pulse cmd add simulator "open -a Simulator"`))
		fmt.Println("  " + ui.Muted.Render("then run it with: ") + cyan.Render("pulse simulator"))
		fmt.Println()
		return nil
	}

	fmt.Println(ui.Title.Render(fmt.Sprintf("⚡  custom commands  ·  %d", len(commands))))
	fmt.Println()

	nameStyle  := lipgloss.NewStyle().Foreground(ui.ColorCyan)
	arrowStyle := lipgloss.NewStyle().Foreground(ui.ColorGray)
	cmdStyle   := lipgloss.NewStyle().Foreground(ui.ColorGray)

	for _, c := range commands {
		fmt.Printf("  %s  %s  %s\n",
			nameStyle.Render("pulse "+c.Name),
			arrowStyle.Render("→"),
			cmdStyle.Render(c.Command),
		)
	}

	fmt.Println()
	fmt.Println("  " + ui.Muted.Render("add  →  ") + cyan.Render(`pulse cmd add <name> "<command>"`))
	fmt.Println("  " + ui.Muted.Render("rm   →  ") + cyan.Render("pulse cmd rm <name>"))
	fmt.Println()
	return nil
}

func runCmdAdd(_ *cobra.Command, args []string) error {
	// DisableFlagParsing means --help arrives as a normal arg
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		fmt.Println()
		fmt.Println(ui.Title.Render("⚡  pulse cmd add"))
		fmt.Println()
		cyan := lipgloss.NewStyle().Foreground(ui.ColorCyan)
		fmt.Println("  " + ui.Muted.Render("usage:"))
		fmt.Println("  " + cyan.Render(`pulse cmd add <name> "<command>"`))
		fmt.Println("  " + cyan.Render(`pulse c add <name> <command words...>`))
		fmt.Println()
		fmt.Println("  " + ui.Muted.Render("both of these do the same thing:"))
		fmt.Println(`  ` + cyan.Render(`pulse cmd add simulator "open -a Simulator"`))
		fmt.Println(`  ` + cyan.Render(`pulse cmd add simulator open -a Simulator`))
		fmt.Println()
		fmt.Println("  " + ui.Muted.Render("name rules: lowercase letters, digits, hyphens — no spaces"))
		fmt.Println("  " + ui.Muted.Render("name can't shadow a built-in pulse command"))
		fmt.Println()
		return nil
	}
	if len(args) < 2 {
		return fmt.Errorf("usage: pulse cmd add <name> \"<command>\"  (or: pulse cmd add <name> command words...)")
	}

	name := strings.ToLower(strings.TrimSpace(args[0]))
	command := strings.TrimSpace(buildCommandString(args[1:]))

	if !validCmdName.MatchString(name) {
		if strings.ContainsRune(name, ' ') {
			return fmt.Errorf("name can't have spaces — try %q", strings.ReplaceAll(name, " ", "-"))
		}
		return fmt.Errorf("name must be lowercase letters, digits, or hyphens and start with a letter (got %q)", name)
	}
	if builtinNames[name] {
		return fmt.Errorf("%q is a built-in pulse command — pick a different name", name)
	}
	if command == "" {
		return fmt.Errorf("command cannot be empty")
	}

	database, err := openCmdDB()
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer database.Close()

	err = database.AddCustomCommand(name, command)
	if errors.Is(err, db.ErrCommandExists) {
		cmds, _ := database.ListCustomCommands()
		for _, c := range cmds {
			if c.Name == name {
				fmt.Println()
				cyan := lipgloss.NewStyle().Foreground(ui.ColorCyan)
				fmt.Println("  " + ui.Muted.Render(`"pulse `+name+`" already exists →  `) + cyan.Render(c.Command))
				fmt.Println("  " + ui.Muted.Render("remove it first: ") + cyan.Render("pulse cmd rm "+name))
				fmt.Println()
				return nil
			}
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("saving command: %w", err)
	}

	fmt.Println()
	cyan := lipgloss.NewStyle().Foreground(ui.ColorCyan)
	fmt.Printf("  %s  %s  %s  %s\n",
		ui.Success.Render("✓"),
		cyan.Render("pulse "+name),
		ui.Muted.Render("→"),
		ui.Muted.Render(command),
	)
	fmt.Println("  " + ui.Muted.Render("run it: ") + cyan.Render("pulse "+name))
	fmt.Println()
	return nil
}

func runCmdRm(_ *cobra.Command, args []string) error {
	name := strings.ToLower(strings.TrimSpace(args[0]))

	database, err := openCmdDB()
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer database.Close()

	fmt.Println()
	if err := database.RemoveCustomCommand(name); errors.Is(err, db.ErrCommandNotFound) {
		cyan := lipgloss.NewStyle().Foreground(ui.ColorCyan)
		fmt.Println("  " + ui.Err.Render(`✗  no custom command named "`+name+`"`))
		fmt.Println("  " + ui.Muted.Render("see your commands: ") + cyan.Render("pulse cmd"))
		fmt.Println()
		return nil
	} else if err != nil {
		return fmt.Errorf("removing command: %w", err)
	}

	fmt.Println("  " + ui.Success.Render(`✓  removed "pulse `+name+`"`))
	fmt.Println()
	return nil
}
