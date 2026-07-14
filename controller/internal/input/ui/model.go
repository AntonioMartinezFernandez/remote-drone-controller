package ui

import (
	"fmt"

	"charm.land/bubbles/v2/progress"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/AntonioMartinezFernandez/remote-drone-controller/internal/input"
)

var _ tea.Model = (*Model)(nil)

type StateMsg struct {
	State *input.InputState
}

var (
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#0EA5E9")).
			PaddingBottom(1).
			PaddingTop(1)

	axisLabelStyle = lipgloss.NewStyle().
			Width(12).
			Align(lipgloss.Right).
			Foreground(lipgloss.Color("#94A3B8"))

	footerStyle = lipgloss.NewStyle().
			Italic(true).
			Foreground(lipgloss.Color("#475569")).
			PaddingTop(1)

	buttonOnStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#22C55E"))
	buttonOffStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#475569"))
)

type Model struct {
	progressBars []progress.Model
	axes         []string
	buttonNames  []string
	state        *input.InputState
	quitting     bool
	DeviceInfo   input.DeviceInfo
}

func NewModel(config input.Config, deviceInfo input.DeviceInfo) Model {
	m := Model{
		progressBars: make([]progress.Model, len(config.Axes)),
		axes:         make([]string, len(config.Axes)),
		buttonNames:  make([]string, len(config.Buttons)),
		DeviceInfo:   deviceInfo,
	}

	for i, ax := range config.Axes {
		m.progressBars[i] = progress.New(
			progress.WithDefaultBlend(),
			progress.WithWidth(30),
			progress.WithSpringOptions(20, 1.0),
			progress.WithoutPercentage(),
		)
		m.axes[i] = string(ax.Name)
	}
	for i, btn := range config.Buttons {
		m.buttonNames[i] = btn.Name
	}

	return m
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		}

	case StateMsg:
		if msg.State == nil {
			return m, nil
		}
		m.state = msg.State
		var cmds []tea.Cmd
		for i, ax := range msg.State.Axes {
			if i < len(m.progressBars) {
				cmd := m.progressBars[i].SetPercent(ax.Value / 100.0)
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			}
		}
		return m, tea.Batch(cmds...)

	case progress.FrameMsg:
		var cmd tea.Cmd
		for i := range m.progressBars {
			m.progressBars[i], cmd = m.progressBars[i].Update(msg)
		}
		return m, cmd
	}
	return m, nil
}

func (m Model) View() tea.View {
	if m.state == nil {
		return tea.NewView(headerStyle.Render("Waiting for input..."))
	}

	var s string
	s += renderHeader(m.DeviceInfo)
	s += m.renderAxes()
	s += m.renderButtons()
	s += renderFooter()

	return tea.NewView(s)
}

func renderHeader(info input.DeviceInfo) string {
	title := "USB HID Drone Controller"
	device := fmt.Sprintf("%s (VID:0x%04X PID:0x%04X)", info.Product, info.Vendor, info.PID)

	return headerStyle.Render(title) + "\n" +
		lipgloss.NewStyle().Foreground(lipgloss.Color("#64748B")).Render(device) + "\n"
}

func (m Model) renderAxes() string {
	var s string

	for i, ax := range m.axes {
		val := m.state.Axes[i].Value
		pct := fmt.Sprintf("%5.1f%%", val)
		label := axisLabelStyle.Render(ax + ":")
		bar := m.progressBars[i].View()

		s += "\n" + lipgloss.NewStyle().PaddingLeft(2).Render(label+" "+bar+" "+pct)
	}

	s += "\n"

	return s
}

func (m Model) renderButtons() string {
	var s string

	for i, btn := range m.buttonNames {
		var state string
		if i < len(m.state.Buttons) && m.state.Buttons[i].Pressed {
			state = buttonOnStyle.Render("ON")
		} else {
			state = buttonOffStyle.Render("OFF")
		}

		s += "\n" + lipgloss.NewStyle().PaddingLeft(2).Render(fmt.Sprintf("  %s: [%s]", btn, state))
	}

	s += "\n"

	return s
}

func renderFooter() string {
	return footerStyle.Render("(*) Press q or ctrl+c to exit")
}
