package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/devpulse-cli/devpulse/internal/db"
)

// tab is a named type so we get compile-time safety: you can't accidentally
// pass a plain int where a tab is expected.
//
// 🧠 Go Lesson #32: `type tab int` creates a NEW distinct type.
// iota generates 0,1,2,3 automatically — like an enum in other languages.
type tab int

const (
	tabOverview tab = iota
	tabCommands
	tabProjects
	tabToday
)

var tabLabels = []string{"overview", "commands", "projects", "today"}

// Messages are plain Go values — any type can be a Bubble Tea message.
//
// 🧠 Go Lesson #33: Go interfaces are satisfied implicitly.
// tea.Msg is just `interface{}` — any value qualifies.
// We identify message types in Update with a type switch: msg.(type).
type (
	tickMsg time.Time // sent every 5 s to trigger a data refresh
	dataMsg struct {  // carries freshly loaded analytics data
		stats    *db.Stats
		cmds     []db.TopEntry
		projects []db.TopEntry
		hourly   []db.HourlyBucket
	}
)

// Model holds the complete state of the dashboard.
// Bubble Tea calls Init/Update/View on this struct.
type Model struct {
	activeTab tab
	width     int
	height    int
	data      dataMsg
	days      int
	ready     bool
	database  *db.DB
}

// New builds a fresh Model.  Pass in the open DB and the day window.
func New(database *db.DB, days int) Model {
	return Model{database: database, days: days}
}

// Init returns the first batch of Cmds to run immediately on startup.
//
// 🧠 Go Lesson #34: tea.Cmd is just `func() tea.Msg`.
// tea.Batch runs multiple Cmds concurrently (like Promise.all).
// The results arrive as messages in Update — no callbacks, no channels.
func (m Model) Init() tea.Cmd {
	return tea.Batch(fetchData(m.database, m.days), scheduleTick())
}

// Update is called for every incoming message. It returns a new Model and
// optionally the next Cmd to run.
//
// 🧠 Go Lesson #35: The type switch `switch msg := msg.(type)` unwraps
// the interface value. Each `case` binds msg to the concrete type,
// so inside `case tea.KeyMsg` msg is a tea.KeyMsg, not interface{}.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "tab", "right", "l":
			m.activeTab = (m.activeTab + 1) % tab(len(tabLabels))
		case "shift+tab", "left", "h":
			m.activeTab = (m.activeTab - 1 + tab(len(tabLabels))) % tab(len(tabLabels))
		case "1":
			m.activeTab = tabOverview
		case "2":
			m.activeTab = tabCommands
		case "3":
			m.activeTab = tabProjects
		case "4":
			m.activeTab = tabToday
		case "r":
			return m, fetchData(m.database, m.days)
		}

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height

	case tickMsg:
		// Auto-refresh: reload data then schedule the next tick.
		return m, tea.Batch(fetchData(m.database, m.days), scheduleTick())

	case dataMsg:
		m.data = msg
		m.ready = true
	}
	return m, nil
}

// fetchData loads all dashboard data from SQLite in the background.
//
// 🧠 Go Lesson #36: Returning a tea.Cmd (a function) instead of calling the
// DB directly means Bubble Tea runs it in a goroutine and sends the result
// back as a message.  The Update loop stays single-threaded and race-free.
func fetchData(database *db.DB, days int) tea.Cmd {
	return func() tea.Msg {
		stats, _ := database.GetStats(days)
		cmds, _ := database.GetTopCommands(days, 8)
		projects, _ := database.GetTopProjects(days, 8)
		hourly, _ := database.GetHourlyStats(time.Now().Format("2006-01-02"))
		return dataMsg{stats: stats, cmds: cmds, projects: projects, hourly: hourly}
	}
}

func scheduleTick() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}
