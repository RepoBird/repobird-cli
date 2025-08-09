package views

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/repobird/repobird-cli/internal/api"
	"github.com/repobird/repobird-cli/internal/models"
	"github.com/repobird/repobird-cli/internal/tui/components"
	"github.com/repobird/repobird-cli/internal/tui/styles"
)

type RunListView struct {
	client       *api.Client
	runs         []models.RunResponse
	table        *components.Table
	keys         components.KeyMap
	help         help.Model
	width        int
	height       int
	loading      bool
	error        error
	spinner      spinner.Model
	pollTicker   *time.Ticker
	pollStop     chan bool
	showHelp     bool
	searchMode   bool
	searchQuery  string
	filteredRuns []models.RunResponse
}

func NewRunListView(client *api.Client) *RunListView {
	columns := []components.Column{
		{Title: "ID", Width: 8},
		{Title: "Status", Width: 15},
		{Title: "Repository", Width: 25},
		{Title: "Time", Width: 12},
		{Title: "Branch", Width: 15},
	}

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("63"))

	return &RunListView{
		client:  client,
		table:   components.NewTable(columns),
		keys:    components.DefaultKeyMap,
		help:    help.New(),
		spinner: s,
		loading: true,
	}
}

func (v *RunListView) Init() tea.Cmd {
	return tea.Batch(
		v.loadRuns(),
		v.spinner.Tick,
		v.startPolling(),
	)
}

func (v *RunListView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
		v.table.SetDimensions(msg.Width, msg.Height-4)
		v.help.Width = msg.Width

	case tea.KeyMsg:
		if v.searchMode {
			switch msg.String() {
			case "enter":
				v.searchMode = false
				v.filterRuns()
			case "esc":
				v.searchMode = false
				v.searchQuery = ""
				v.filterRuns()
			case "backspace":
				if len(v.searchQuery) > 0 {
					v.searchQuery = v.searchQuery[:len(v.searchQuery)-1]
				}
			default:
				if len(msg.String()) == 1 {
					v.searchQuery += msg.String()
				}
			}
			return v, nil
		}

		switch {
		case key.Matches(msg, v.keys.Quit):
			v.stopPolling()
			return v, tea.Quit
		case key.Matches(msg, v.keys.Help):
			v.showHelp = !v.showHelp
		case key.Matches(msg, v.keys.Refresh):
			cmds = append(cmds, v.loadRuns())
		case key.Matches(msg, v.keys.Search):
			v.searchMode = true
			v.searchQuery = ""
		case key.Matches(msg, v.keys.Enter):
			if idx := v.table.GetSelectedIndex(); idx >= 0 && idx < len(v.filteredRuns) {
				run := v.filteredRuns[idx]
				return NewRunDetailsView(v.client, run), nil
			}
		case key.Matches(msg, v.keys.New):
			return NewCreateRunView(v.client), nil
		case key.Matches(msg, v.keys.Up):
			v.table.MoveUp()
		case key.Matches(msg, v.keys.Down):
			v.table.MoveDown()
		case key.Matches(msg, v.keys.PageUp):
			v.table.PageUp()
		case key.Matches(msg, v.keys.PageDown):
			v.table.PageDown()
		case key.Matches(msg, v.keys.Home):
			v.table.GoToTop()
		case key.Matches(msg, v.keys.End):
			v.table.GoToBottom()
		}

	case runsLoadedMsg:
		v.loading = false
		v.runs = msg.runs
		v.error = msg.err
		v.filterRuns()

	case pollTickMsg:
		if v.hasActiveRuns() {
			cmds = append(cmds, v.loadRuns())
		}

	case spinner.TickMsg:
		if v.loading {
			var cmd tea.Cmd
			v.spinner, cmd = v.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return v, tea.Batch(cmds...)
}

func (v *RunListView) View() string {
	var s strings.Builder

	title := styles.TitleStyle.Render("RepoBird CLI - Runs")
	s.WriteString(title)
	s.WriteString("\n\n")

	if v.loading {
		s.WriteString(v.spinner.View() + " Loading runs...\n")
	} else if v.error != nil {
		s.WriteString(styles.ErrorStyle.Render("Error: " + v.error.Error()) + "\n")
	} else {
		if v.searchMode {
			s.WriteString("Search: " + v.searchQuery + "_\n")
		} else if v.searchQuery != "" {
			s.WriteString("Filter: " + v.searchQuery + "\n")
		}

		s.WriteString(v.table.View())
		s.WriteString("\n")

		statusBar := v.renderStatusBar()
		s.WriteString(statusBar)
	}

	if v.showHelp {
		helpView := v.help.View(v.keys)
		s.WriteString("\n" + helpView)
	}

	return s.String()
}

func (v *RunListView) filterRuns() {
	if v.searchQuery == "" {
		v.filteredRuns = v.runs
	} else {
		v.filteredRuns = []models.RunResponse{}
		query := strings.ToLower(v.searchQuery)
		for _, run := range v.runs {
			if strings.Contains(strings.ToLower(run.ID), query) ||
				strings.Contains(strings.ToLower(run.Repository), query) ||
				strings.Contains(strings.ToLower(string(run.Status)), query) ||
				strings.Contains(strings.ToLower(run.Source), query) ||
				strings.Contains(strings.ToLower(run.Target), query) {
				v.filteredRuns = append(v.filteredRuns, run)
			}
		}
	}
	v.updateTable()
}

func (v *RunListView) updateTable() {
	rows := make([]components.Row, len(v.filteredRuns))
	for i, run := range v.filteredRuns {
		statusIcon := styles.GetStatusIcon(string(run.Status))
		statusText := fmt.Sprintf("%s %s", statusIcon, run.Status)
		timeAgo := formatTimeAgo(run.CreatedAt)
		branch := run.Source
		if run.Target != "" && run.Target != run.Source {
			branch = fmt.Sprintf("%s→%s", run.Source, run.Target)
		}

		rows[i] = components.Row{
			run.ID[:8],
			statusText,
			run.Repository,
			timeAgo,
			branch,
		}
	}
	v.table.SetRows(rows)
}

func (v *RunListView) renderStatusBar() string {
	left := fmt.Sprintf(" %d runs | %s", len(v.filteredRuns), v.table.StatusLine())
	
	activeCount := 0
	for _, run := range v.runs {
		if isActiveStatus(string(run.Status)) {
			activeCount++
		}
	}
	
	right := ""
	if activeCount > 0 {
		right = fmt.Sprintf("⟳ %d active ", activeCount)
	}
	
	right += "[n]ew [r]efresh [/]search [?]help [q]uit "

	padding := v.width - len(left) - len(right)
	if padding < 0 {
		padding = 0
	}

	return styles.StatusBarStyle.Width(v.width).Render(
		left + strings.Repeat(" ", padding) + right,
	)
}

func (v *RunListView) loadRuns() tea.Cmd {
	return func() tea.Msg {
		runs, err := v.client.ListRuns()
		return runsLoadedMsg{runs: runs, err: err}
	}
}

func (v *RunListView) startPolling() tea.Cmd {
	v.pollTicker = time.NewTicker(5 * time.Second)
	v.pollStop = make(chan bool)

	return func() tea.Msg {
		for {
			select {
			case <-v.pollTicker.C:
				return pollTickMsg{}
			case <-v.pollStop:
				return nil
			}
		}
	}
}

func (v *RunListView) stopPolling() {
	if v.pollTicker != nil {
		v.pollTicker.Stop()
	}
	if v.pollStop != nil {
		close(v.pollStop)
	}
}

func (v *RunListView) hasActiveRuns() bool {
	for _, run := range v.runs {
		if isActiveStatus(string(run.Status)) {
			return true
		}
	}
	return false
}

func isActiveStatus(status string) bool {
	activeStatuses := []string{"QUEUED", "INITIALIZING", "PROCESSING", "POST_PROCESS"}
	for _, s := range activeStatuses {
		if status == s {
			return true
		}
	}
	return false
}

func formatTimeAgo(t time.Time) string {
	duration := time.Since(t)
	
	if duration < time.Minute {
		return fmt.Sprintf("%ds ago", int(duration.Seconds()))
	} else if duration < time.Hour {
		return fmt.Sprintf("%dm ago", int(duration.Minutes()))
	} else if duration < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(duration.Hours()))
	} else {
		return fmt.Sprintf("%dd ago", int(duration.Hours()/24))
	}
}

type runsLoadedMsg struct {
	runs []models.RunResponse
	err  error
}

type pollTickMsg struct{}