package cmd

import (
	"errors"
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
	Use:   "rm <position>",
	Short: "remove a favourite by its list number",
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

	posStyle   := lipgloss.NewStyle().Foreground(ui.ColorGold).Width(4)
	cmdStyle   := lipgloss.NewStyle().Foreground(ui.ColorCyan)
	aliasStyle := lipgloss.NewStyle().Foreground(ui.ColorGray)
	ageStyle   := lipgloss.NewStyle().Foreground(ui.ColorGray)

	for i, f := range favs {
		alias := ""
		if f.Alias != "" {
			alias = "  " + aliasStyle.Render("["+f.Alias+"]")
		}
		age := formatAge(f.CreatedAt)
		fmt.Printf("  %s  %s%s  %s\n",
			posStyle.Render(fmt.Sprintf("#%d", i+1)),
			cmdStyle.Render(ui.Truncate(f.Command, 55)),
			alias,
			ageStyle.Render(age),
		)
	}

	fmt.Println()
	cyan := lipgloss.NewStyle().Foreground(ui.ColorCyan)
	fmt.Println("  " + ui.Muted.Render("tip: add a fav →  ") + cyan.Render("pulse f add \"<command>\""))
	fmt.Println("       " + ui.Muted.Render("remove one →  ") + cyan.Render("pulse f rm <number>"))
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

	_, err = database.AddFavorite(command, favAddAlias)
	if errors.Is(err, db.ErrAlreadySaved) {
		// find its position in the current list
		favs, _ := database.ListFavorites()
		pos := positionOf(favs, command)
		fmt.Println()
		cyan := lipgloss.NewStyle().Foreground(ui.ColorCyan)
		fmt.Println("  " + ui.Muted.Render(fmt.Sprintf("already saved as #%d  —  see it with ", pos)) + cyan.Render("pulse f"))
		fmt.Println()
		return nil
	}
	if err != nil {
		return fmt.Errorf("saving favourite: %w", err)
	}

	// find position of newly added item
	favs, _ := database.ListFavorites()
	pos := positionOf(favs, command)

	fmt.Println()
	cyan := lipgloss.NewStyle().Foreground(ui.ColorCyan)
	aliasNote := ""
	if favAddAlias != "" {
		aliasNote = "  " + ui.Muted.Render(fmt.Sprintf("alias: [%s]", favAddAlias))
	}
	fmt.Println("  " + ui.Success.Render(fmt.Sprintf("★  saved!  #%d", pos)) + "  " + cyan.Render(ui.Truncate(command, 55)) + aliasNote)
	fmt.Println("     " + ui.Muted.Render("list your favs with ") + cyan.Render("pulse f") +
		ui.Muted.Render("  ·  remove with ") + cyan.Render(fmt.Sprintf("pulse f rm %d", pos)))
	fmt.Println()
	return nil
}

func runFavRm(_ *cobra.Command, args []string) error {
	pos, err := strconv.Atoi(args[0])
	if err != nil || pos < 1 {
		return fmt.Errorf("invalid number %q — use the position shown in  pulse f", args[0])
	}

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
	if pos > len(favs) {
		fmt.Println("  " + ui.Err.Render(fmt.Sprintf("✗  no favourite at position #%d", pos)))
		fmt.Println("     " + ui.Muted.Render("run ") + lipgloss.NewStyle().Foreground(ui.ColorCyan).Render("pulse f") + ui.Muted.Render(" to see valid positions"))
	} else {
		_ , _ = database.RemoveFavorite(favs[pos-1].ID)
		fmt.Println("  " + ui.Success.Render(fmt.Sprintf("removed #%d  %s", pos, ui.Truncate(favs[pos-1].Command, 50))))
	}
	fmt.Println()
	return nil
}

// positionOf returns the 1-based position of command in the list, or 0 if not found.
func positionOf(favs []db.FavoriteRow, command string) int {
	for i, f := range favs {
		if f.Command == command {
			return i + 1
		}
	}
	return 0
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
