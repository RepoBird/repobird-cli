package views

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/repobird/repobird-cli/internal/api"
	"github.com/repobird/repobird-cli/internal/models"
	"github.com/repobird/repobird-cli/internal/tui/components"
	"github.com/repobird/repobird-cli/internal/tui/styles"
)

type RunDetailsView struct {
	client        *api.Client
	run           models.RunResponse
	keys          components.KeyMap
	help          help.Model
	viewport      viewport.Model
	width         int
	height        int
	loading       bool
	error         error
	spinner       spinner.Model
	pollTicker    *time.Ticker
	pollStop      chan bool
	showHelp      bool
	showLogs      bool
	logs          string
	statusHistory []string
}

func NewRunDetailsView(client *api.Client, run models.RunResponse) *RunDetailsView {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("63"))

	vp := viewport.New(80, 20)

	return &RunDetailsView{
		client:   client,
		run:      run,
		keys:     components.DefaultKeyMap,
		help:     help.New(),
		viewport: vp,
		spinner:  s,
		loading:  true,
		showLogs: false,
	}
}

func (v *RunDetailsView) Init() tea.Cmd {
	return tea.Batch(
		v.loadRunDetails(),
		v.spinner.Tick,
		v.startPolling(),
	)
}

func (v *RunDetailsView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
		v.viewport.Width = msg.Width
		v.viewport.Height = msg.Height - 8
		v.help.Width = msg.Width

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, v.keys.Quit):
			v.stopPolling()
			return v, tea.Quit
		case key.Matches(msg, v.keys.Back):
			v.stopPolling()
			return NewRunListView(v.client), nil
		case key.Matches(msg, v.keys.Help):
			v.showHelp = !v.showHelp
		case key.Matches(msg, v.keys.Refresh):
			cmds = append(cmds, v.loadRunDetails())
		case msg.String() == "l":
			v.showLogs = !v.showLogs
			v.updateContent()
		case key.Matches(msg, v.keys.Up):
			v.viewport.LineUp(1)
		case key.Matches(msg, v.keys.Down):
			v.viewport.LineDown(1)
		case key.Matches(msg, v.keys.PageUp):
			v.viewport.HalfViewUp()
		case key.Matches(msg, v.keys.PageDown):
			v.viewport.HalfViewDown()
		case key.Matches(msg, v.keys.Home):
			v.viewport.GotoTop()
		case key.Matches(msg, v.keys.End):
			v.viewport.GotoBottom()
		}

	case runDetailsLoadedMsg:
		v.loading = false
		v.run = msg.run
		v.error = msg.err
		if msg.err == nil {
			v.updateStatusHistory(string(msg.run.Status))
		}
		v.updateContent()

	case pollTickMsg:
		if isActiveStatus(string(v.run.Status)) {
			cmds = append(cmds, v.loadRunDetails())
		} else {
			v.stopPolling()
		}

	case spinner.TickMsg:
		if v.loading {
			var cmd tea.Cmd
			v.spinner, cmd = v.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	var vpCmd tea.Cmd
	v.viewport, vpCmd = v.viewport.Update(msg)
	cmds = append(cmds, vpCmd)

	return v, tea.Batch(cmds...)
}

func (v *RunDetailsView) View() string {
	var s strings.Builder

	header := v.renderHeader()
	s.WriteString(header)
	s.WriteString("\n")
	s.WriteString(strings.Repeat("─", v.width))
	s.WriteString("\n")

	if v.loading && len(v.statusHistory) == 0 {
		s.WriteString(v.spinner.View() + " Loading run details...\n")
	} else if v.error != nil {
		s.WriteString(styles.ErrorStyle.Render("Error: "+v.error.Error()) + "\n")
	} else {
		s.WriteString(v.viewport.View())
		s.WriteString("\n")
	}

	statusBar := v.renderStatusBar()
	s.WriteString(statusBar)

	if v.showHelp {
		helpView := v.help.View(v.keys)
		s.WriteString("\n" + helpView)
	}

	return s.String()
}

func (v *RunDetailsView) renderHeader() string {
	statusIcon := styles.GetStatusIcon(string(v.run.Status))
	statusStyle := styles.GetStatusStyle(string(v.run.Status))
	status := statusStyle.Render(fmt.Sprintf("%s %s", statusIcon, v.run.Status))

	title := fmt.Sprintf("Run #%s", v.run.ID[:8])
	if v.run.Title != "" {
		title += " - " + v.run.Title
	}

	header := styles.TitleStyle.Render(title)

	if isActiveStatus(string(v.run.Status)) {
		pollingIndicator := styles.ProcessingStyle.Render(" [Polling ⟳]")
		header += pollingIndicator
	}

	rightAlign := lipgloss.NewStyle().Align(lipgloss.Right).Width(v.width - lipgloss.Width(header))
	header += rightAlign.Render("Status: " + status)

	return header
}

func (v *RunDetailsView) renderStatusBar() string {
	options := "[b]ack [l]ogs [r]efresh [?]help [q]uit"

	if v.showLogs {
		options = "[b]ack [l]details [r]efresh [?]help [q]uit"
	}

	return styles.StatusBarStyle.Width(v.width).Render(options)
}

func (v *RunDetailsView) updateContent() {
	var content strings.Builder

	if v.showLogs {
		content.WriteString("═══ Logs ═══\n\n")
		if v.logs != "" {
			content.WriteString(v.logs)
		} else {
			content.WriteString("No logs available yet...\n")
		}
	} else {
		content.WriteString(fmt.Sprintf("Title: %s\n", v.run.Title))
		content.WriteString(fmt.Sprintf("Repository: %s\n", v.run.Repository))
		content.WriteString(fmt.Sprintf("Source Branch: %s\n", v.run.Source))
		if v.run.Target != "" && v.run.Target != v.run.Source {
			content.WriteString(fmt.Sprintf("Target Branch: %s\n", v.run.Target))
		}
		content.WriteString(fmt.Sprintf("Created: %s\n", v.run.CreatedAt.Format(time.RFC3339)))

		if v.run.UpdatedAt.After(v.run.CreatedAt) && (v.run.Status == models.StatusDone || v.run.Status == models.StatusFailed) {
			duration := v.run.UpdatedAt.Sub(v.run.CreatedAt)
			content.WriteString(fmt.Sprintf("Duration: %s\n", formatDuration(duration)))
		}

		content.WriteString("\n═══ Status History ═══\n")
		for _, status := range v.statusHistory {
			content.WriteString(status + "\n")
		}

		if v.run.Context != "" {
			content.WriteString("\n═══ Context ═══\n")
			content.WriteString(v.run.Context + "\n")
		}

		if v.run.Error != "" {
			content.WriteString("\n═══ Error ═══\n")
			content.WriteString(styles.ErrorStyle.Render(v.run.Error) + "\n")
		}
	}

	v.viewport.SetContent(content.String())
}

func (v *RunDetailsView) updateStatusHistory(status string) {
	if len(v.statusHistory) == 0 || v.statusHistory[len(v.statusHistory)-1] != status {
		timestamp := time.Now().Format("15:04:05")
		statusIcon := styles.GetStatusIcon(status)
		entry := fmt.Sprintf("[%s] %s %s", timestamp, statusIcon, status)
		v.statusHistory = append(v.statusHistory, entry)
	}
}

func (v *RunDetailsView) loadRunDetails() tea.Cmd {
	return func() tea.Msg {
		runPtr, err := v.client.GetRun(v.run.ID)
		if err != nil {
			return runDetailsLoadedMsg{run: v.run, err: err}
		}
		return runDetailsLoadedMsg{run: *runPtr, err: nil}
	}
}

func (v *RunDetailsView) startPolling() tea.Cmd {
	if !isActiveStatus(string(v.run.Status)) {
		return nil
	}

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

func (v *RunDetailsView) stopPolling() {
	if v.pollTicker != nil {
		v.pollTicker.Stop()
	}
	if v.pollStop != nil {
		select {
		case <-v.pollStop:
		default:
			close(v.pollStop)
		}
	}
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	} else if d < time.Hour {
		return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
	} else {
		return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
	}
}

type runDetailsLoadedMsg struct {
	run models.RunResponse
	err error
}
