// Package qa answers natural-ish questions with local Pulse data.
// It deliberately avoids AI: similar questions map to known intents, then run
// SQLite-backed analytics. This keeps `pulse ask` free and private.
package qa

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/abdulqadirmsingi/pulse-cli/internal/db"
)

type Store interface {
	GetStats(days int) (*db.Stats, error)
	GetTodayStats() (*db.Stats, error)
	GetTodayTopProjects(limit int) ([]db.TopEntry, error)
	GetTopCommands(days, limit int) ([]db.TopEntry, error)
	GetTopProjects(days, limit int) ([]db.TopEntry, error)
	GetHourlyStats(date string) ([]db.HourlyBucket, error)
}

type Answer struct {
	Title string
	Lines []string
	Tips  []string
}

func Suggestions() []string {
	return []string{
		"what did I work on today?",
		"which project took most of my time?",
		"why is my success rate low?",
		"what commands do I repeat a lot?",
		"how was my week?",
	}
}

func AnswerQuestion(store Store, question string) (Answer, error) {
	q := normalize(question)
	switch {
	case q == "" || hasAny(q, "help", "examples", "what can i ask"):
		return helpAnswer(), nil
	case hasAny(q, "today", "worked on", "work on today", "did i do", "doing today"):
		return todayAnswer(store)
	case hasAny(q, "success", "fail", "failed", "error", "exit rate", "broken"):
		return successAnswer(store)
	case hasAny(q, "repeat", "often", "most used", "common command", "commands do i use", "top command"):
		return repeatedCommandsAnswer(store)
	case hasAny(q, "week", "weekly", "recap", "summary", "lately", "last 7"):
		return weekAnswer(store)
	case hasAny(q, "project", "repo", "repository", "most time", "time spent", "focus"):
		return projectAnswer(store)
	case hasAny(q, "streak", "consistent", "consistency"):
		return streakAnswer(store)
	default:
		return fallbackAnswer(), nil
	}
}

func todayAnswer(store Store) (Answer, error) {
	stats, err := store.GetTodayStats()
	if err != nil {
		return Answer{}, err
	}
	projects, err := store.GetTodayTopProjects(3)
	if err != nil {
		return Answer{}, err
	}
	hourly, err := store.GetHourlyStats(time.Now().Format("2006-01-02"))
	if err != nil {
		return Answer{}, err
	}

	lines := []string{
		fmt.Sprintf("Today you logged %s of focused terminal time across %d dev commands.", formatDuration(stats.TotalTimeMS), stats.TotalCommands),
	}
	if stats.NoiseCommands > 0 {
		lines = append(lines, fmt.Sprintf("I ignored %d housekeeping commands so the numbers stay honest.", stats.NoiseCommands))
	}
	if len(projects) > 0 {
		lines = append(lines, "Top projects today:")
		for i, p := range projects {
			lines = append(lines, fmt.Sprintf("%d. %s — %s", i+1, p.Name, formatDuration(p.MS)))
		}
	}
	if hour, count := busiestHour(hourly); count > 0 {
		lines = append(lines, fmt.Sprintf("Your busiest hour was %02d:00 with %d commands.", hour, count))
	}
	return Answer{Title: "today's pulse", Lines: lines, Tips: []string{"nice work — clean data, clean decisions"}}, nil
}

func projectAnswer(store Store) (Answer, error) {
	projects, err := store.GetTopProjects(7, 5)
	if err != nil {
		return Answer{}, err
	}
	if len(projects) == 0 {
		return Answer{Title: "project focus", Lines: []string{"No project time yet. Run commands inside a git repo and Pulse will start ranking them."}}, nil
	}
	lines := []string{"Your most active projects in the last 7 days:"}
	for i, p := range projects {
		lines = append(lines, fmt.Sprintf("%d. %s — %s across %d commands", i+1, p.Name, formatDuration(p.MS), p.Count))
	}
	tips := []string{"if one project dominates, that is focus; if five do, that is context switching tax"}
	return Answer{Title: "project focus", Lines: lines, Tips: tips}, nil
}

func successAnswer(store Store) (Answer, error) {
	stats, err := store.GetStats(7)
	if err != nil {
		return Answer{}, err
	}
	lines := []string{
		fmt.Sprintf("Your 7-day success rate is %.1f%% across %d dev commands.", stats.SuccessRate, stats.TotalCommands),
	}
	var tips []string
	switch {
	case stats.TotalCommands == 0:
		lines = append(lines, "No dev commands yet, so there is nothing to judge.")
	case stats.SuccessRate >= 95:
		tips = append(tips, "clean execution — you are moving with intent")
	case stats.SuccessRate >= 85:
		tips = append(tips, "solid overall; failed commands are normal while debugging")
	default:
		tips = append(tips, "slow down on repeated failures; check the first error before rerunning")
	}
	return Answer{Title: "success rate", Lines: lines, Tips: tips}, nil
}

func repeatedCommandsAnswer(store Store) (Answer, error) {
	cmds, err := store.GetTopCommands(30, 8)
	if err != nil {
		return Answer{}, err
	}
	if len(cmds) == 0 {
		return Answer{Title: "repeated commands", Lines: []string{"No repeated dev commands yet. Give Pulse a little more history."}}, nil
	}
	lines := []string{"Your most repeated commands in the last 30 days:"}
	for i, c := range cmds {
		lines = append(lines, fmt.Sprintf("%d. %s — %d runs", i+1, c.Name, c.Count))
	}
	return Answer{Title: "repeated commands", Lines: lines, Tips: commandTips(cmds)}, nil
}

func weekAnswer(store Store) (Answer, error) {
	stats, err := store.GetStats(7)
	if err != nil {
		return Answer{}, err
	}
	projects, err := store.GetTopProjects(7, 3)
	if err != nil {
		return Answer{}, err
	}
	cmds, err := store.GetTopCommands(7, 3)
	if err != nil {
		return Answer{}, err
	}

	lines := []string{
		fmt.Sprintf("This week: %d dev commands, %s focused time, %.1f%% success.", stats.TotalCommands, formatDuration(stats.TotalTimeMS), stats.SuccessRate),
		fmt.Sprintf("Current streak: %d days.", stats.StreakDays),
	}
	if len(projects) > 0 {
		lines = append(lines, "Main projects: "+joinNames(projects))
	}
	if len(cmds) > 0 {
		lines = append(lines, "Main tools: "+joinNames(cmds))
	}
	return Answer{Title: "weekly recap", Lines: lines, Tips: []string{"you have enough signal here to spot patterns, not just count commands"}}, nil
}

func streakAnswer(store Store) (Answer, error) {
	stats, err := store.GetStats(30)
	if err != nil {
		return Answer{}, err
	}
	line := fmt.Sprintf("Your current streak is %d days.", stats.StreakDays)
	tip := "showing up consistently beats one giant grind day"
	if stats.StreakDays >= 7 {
		tip = "streak looking strong — keep the chain alive"
	}
	return Answer{Title: "streak", Lines: []string{line}, Tips: []string{tip}}, nil
}

func helpAnswer() Answer {
	lines := []string{"Ask Pulse about your local activity. Try:"}
	for _, s := range Suggestions() {
		lines = append(lines, "pulse ask \""+s+"\"")
	}
	return Answer{Title: "ask pulse", Lines: lines}
}

func fallbackAnswer() Answer {
	return Answer{
		Title: "ask pulse",
		Lines: []string{
			"I can answer local activity questions, but I could not match that wording yet.",
			"Try asking about today, projects, success rate, repeated commands, your week, or your streak.",
		},
		Tips: Suggestions(),
	}
}

func normalize(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	replacer := strings.NewReplacer("?", " ", "!", " ", ".", " ", ",", " ", "-", " ")
	return strings.Join(strings.Fields(replacer.Replace(s)), " ")
}

func hasAny(s string, needles ...string) bool {
	for _, n := range needles {
		if strings.Contains(s, n) {
			return true
		}
	}
	return false
}

func busiestHour(buckets []db.HourlyBucket) (int, int64) {
	sort.SliceStable(buckets, func(i, j int) bool { return buckets[i].Count > buckets[j].Count })
	if len(buckets) == 0 {
		return 0, 0
	}
	return buckets[0].Hour, buckets[0].Count
}

func joinNames(entries []db.TopEntry) string {
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name)
	}
	return strings.Join(names, ", ")
}

func commandTips(cmds []db.TopEntry) []string {
	tipsByCommand := map[string]string{
		"git":    "git is doing heavy lifting; `git status -s` keeps checks fast",
		"go":     "Go in the top list — compiled and composed, love to see it",
		"npm":    "lots of npm? a package script shortcut might save keystrokes",
		"docker": "docker showing up often? `lazydocker` may be worth trying",
		"rg":     "ripgrep in the stack — elite search behavior",
	}
	var tips []string
	for _, c := range cmds {
		if tip, ok := tipsByCommand[strings.ToLower(c.Name)]; ok {
			tips = append(tips, tip)
		}
		if len(tips) == 2 {
			break
		}
	}
	if len(tips) == 0 {
		tips = append(tips, "repeated commands are candidates for aliases, scripts, or Pulse custom commands")
	}
	return tips
}

func formatDuration(ms int64) string {
	if ms <= 0 {
		return "0s"
	}
	secs := ms / 1000
	mins := secs / 60
	hours := mins / 60
	switch {
	case hours > 0:
		return fmt.Sprintf("%dh %dm", hours, mins%60)
	case mins > 0:
		return fmt.Sprintf("%dm %ds", mins, secs%60)
	default:
		return fmt.Sprintf("%ds", secs)
	}
}
