package cmd

import (
	"fmt"
	"strconv"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/devpulse-cli/devpulse/internal/config"
	"github.com/devpulse-cli/devpulse/internal/db"
	"github.com/devpulse-cli/devpulse/internal/ui"
	"github.com/spf13/cobra"
)

var favAddAlias string

var favCmd = &cobra.Command{
	Use:     "fav",
	Aliases: []string{"f"},
	Short:   "manage your favourite commands ★",
	Long:    "List, save, and remove favourite commands for quick reference.",
	RunE:    runFavList,
}

var favAddCmd = &cobra.Command{
	Use:   "add <command>",
	Short: "save a command as a favourite",
	Args:  cobra.ExactArgs(1),
	RunE:  runFavAdd,
}

var favRmCmd = &cobra.Command{
	Use:   "rm <id>",
	Short: "remove a favourite by ID",
	Args:  cobra.ExactArgs(1),
	RunE:  runFavRm,
}

func init() {
	rootCmd.AddCommand(favCmd)
	favCmd.AddCommand(favAddCmd)
	favCmd.AddCommand(favRmCmd)
	favAddCmd.Flags().StringVar(&favAddAlias, "as", "", "short alias for this command")
}

func openFavDB() (*db.DB, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	return db.Open(cfg.DBPath)
}

func runFavList(_ *cobra.Command, _ []string) error {
	database, err := openFavDB()
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer database.Close()

	favs, err := database.ListFavorites()
	if err != nil {
		return fmt.Errorf("loading favourites: %w", err)
	}

	fmt.Println()
	if len(favs) == 0 {
		fmt.Println(ui.Title.Render("★  favorites  ·  none saved yet"))
		fmt.Println()
		cyan := lipgloss.NewStyle().Foreground(ui.ColorCyan)
		fmt.Println("  " + ui.Muted.Render("save your first one:"))
		fmt.Println("  " + cyan.Render("pulse f add \"<command>\""))
		fmt.Println("  " + cyan.Render("pulse f add \"<command>\" --as <alias>"))
		fmt.Println()
		return nil
	}

	fmt.Println(ui.Title.Render(fmt.Sprintf("★  favorites  ·  %d saved", len(favs))))
	fmt.Println()

	idStyle    := lipgloss.NewStyle().Foreground(ui.ColorGold).Width(4)
	cmdStyle   := lipgloss.NewStyle().Foreground(ui.ColorCyan)
	aliasStyle := lipgloss.NewStyle().Foreground(ui.ColorGray)
	ageStyle   := lipgloss.NewStyle().Foreground(ui.ColorGray)

	for _, f := range favs {
		alias := ""
		if f.Alias != "" {
			alias = "  " + aliasStyle.Render("["+f.Alias+"]")
		}
		age := formatAge(f.CreatedAt)
		fmt.Printf("  %s  %s%s  %s\n",
			idStyle.Render(fmt.Sprintf("#%d", f.ID)),
			cmdStyle.Render(ui.Truncate(f.Command, 55)),
			alias,
			ageStyle.Render(age),
		)
	}

	fmt.Println()
	cyan := lipgloss.NewStyle().Foreground(ui.ColorCyan)
	fmt.Println("  " + ui.Muted.Render("tip: add a fav →  ") + cyan.Render("pulse f add \"<command>\""))
	fmt.Println("       " + ui.Muted.Render("remove one →  ") + cyan.Render("pulse f rm <id>"))
	fmt.Println("       " + ui.Muted.Render("save with alias →  ") + cyan.Render("pulse f add \"<cmd>\" --as <alias>"))
	fmt.Println()
	return nil
}

func runFavAdd(_ *cobra.Command, args []string) error {
	command := args[0]

	database, err := openFavDB()
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer database.Close()

	id, err := database.AddFavorite(command, favAddAlias)
	if err != nil {
		return fmt.Errorf("saving favourite: %w", err)
	}

	fmt.Println()
	cyan := lipgloss.NewStyle().Foreground(ui.ColorCyan)
	aliasNote := ""
	if favAddAlias != "" {
		aliasNote = "  " + ui.Muted.Render(fmt.Sprintf("alias: [%s]", favAddAlias))
	}
	fmt.Println("  " + ui.Success.Render(fmt.Sprintf("★  saved!  #%d", id)) + "  " + cyan.Render(ui.Truncate(command, 55)) + aliasNote)
	fmt.Println("     " + ui.Muted.Render("list your favs with ") + cyan.Render("pulse f") +
		ui.Muted.Render("  ·  remove with ") + cyan.Render(fmt.Sprintf("pulse f rm %d", id)))
	fmt.Println()
	return nil
}

func runFavRm(_ *cobra.Command, args []string) error {
	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid id %q — use the number shown in  pulse f", args[0])
	}

	database, err := openFavDB()
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer database.Close()

	found, err := database.RemoveFavorite(id)
	if err != nil {
		return fmt.Errorf("removing favourite: %w", err)
	}

	fmt.Println()
	if !found {
		fmt.Println("  " + ui.Err.Render(fmt.Sprintf("✗  no favourite with id #%d", id)))
		fmt.Println("     " + ui.Muted.Render("run ") + lipgloss.NewStyle().Foreground(ui.ColorCyan).Render("pulse f") + ui.Muted.Render(" to see valid IDs"))
	} else {
		fmt.Println("  " + ui.Success.Render(fmt.Sprintf("removed #%d", id)))
	}
	fmt.Println()
	return nil
}

// formatAge returns a human-readable age string relative to now.
func formatAge(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < 2*time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 48*time.Hour:
		return "yesterday"
	default:
		return fmt.Sprintf("%d days ago", int(d.Hours()/24))
	}
}
