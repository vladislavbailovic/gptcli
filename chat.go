package main

import (
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/glamour"
)

func chat(opts options, convo conversation) {
	width := 30

	tx := textarea.New()

	tx.FocusedStyle.CursorLine = lipgloss.NewStyle()
	tx.ShowLineNumbers = false
	tx.CharLimit = 280

	tx.SetWidth(width)
	tx.SetHeight(1)

	tx.Focus()
	tx.KeyMap.InsertNewline.SetEnabled(false)

	vp := viewport.New(width, 5)

	m := model{
		opts:     opts,
		convo:    convo,
		prompt:   tx,
		viewport: vp,
		width:    width,
	}
	m.setStatus(statusAwaitingInput)

	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}

type systemStatus uint8

const (
	statusAwaitingInput systemStatus = iota
	statusAwaitingResponse
	statusAwaitingCommand
)

type model struct {
	opts  options
	convo conversation

	status     systemStatus
	statusLine string

	prompt   textarea.Model
	viewport viewport.Model

	width, height int
}

func (m *model) setStatusMsg(msg string) {
	m.statusLine = msg
}

func (m *model) setStatus(s systemStatus) {
	m.status = s
	switch s {
	case statusAwaitingInput:
		m.statusLine = "Enter to send, Ctrl+C to quit"
		m.prompt.Prompt = "> "
	case statusAwaitingResponse:
		m.statusLine = "... Awaiting response ..."
		m.prompt.Prompt = "> "
	case statusAwaitingCommand:
		m.statusLine = "Enter command"
		m.prompt.Prompt = ""
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, updateViewportDelayed)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		txCmd tea.Cmd
		vpCmd tea.Cmd
		myCmd tea.Cmd
	)

	m.prompt, txCmd = m.prompt.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height = max(msg.Height, 6)
		m.width = min(msg.Width, 120)

		m.viewport.Width = m.width
		m.viewport.Height = m.height - 3
		m.prompt.SetWidth(m.width)

		myCmd = updateViewport
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyCtrlD:
			return m, tea.Quit
		case tea.KeyEsc:
			if m.status == statusAwaitingCommand {
				m.setStatus(statusAwaitingInput)
			} else if m.status != statusAwaitingResponse {
				m.setStatus(statusAwaitingCommand)
			}
		case tea.KeyEnter:
			currentPrompt := m.prompt.Value()
			if currentPrompt != "" {
				m.prompt.Placeholder = currentPrompt
				m.prompt.Reset()
				m.prompt.Blur()
				if currentPrompt[0] == ':' {
					m.setStatus(statusAwaitingCommand)
				}
				switch m.status {
				case statusAwaitingInput:
					m.setStatus(statusAwaitingResponse)
					myCmd = fetchResponse(currentPrompt, m)
				case statusAwaitingCommand:
					myCmd = executeCommand(currentPrompt, m.convo)
				}
			} else {
				myCmd = updateViewport
			}
		}
	case refresh:
		m.viewport.SetContent(renderMessages(m.convo, m.width))
		m.viewport.GotoBottom()
	case response:
		m.convo = msg.convo
		m.setStatus(statusAwaitingInput)
		m.prompt.Placeholder = ""
		m.prompt.Focus()
		myCmd = updateViewport
	case executionResult:
		if msg.stderr != "" {
			m.setStatusMsg(msg.stderr)
			myCmd = switchToAfter(statusAwaitingInput, 2)
		} else {
			myCmd = switchToAfter(statusAwaitingInput, 0)
		}
	case switchToStatus:
		m.prompt.Placeholder = ""
		m.prompt.Focus()
		m.setStatus(msg.status)
	}

	return m, tea.Batch(txCmd, vpCmd, myCmd)
}

func (m model) View() string {
	prompt := m.prompt.View()
	prompt = lipgloss.NewStyle().
		Width(m.width).
		Align(lipgloss.Center).
		Faint(true).
		Italic(true).
		Render(m.statusLine) + "\n" + prompt
	return fmt.Sprintf(
		"%s\n%s",
		m.viewport.View(),
		prompt,
	) + "\n\n"
}

func renderMessages(convo conversation, width int) string {
	box := lipgloss.NewStyle().Width(width)
	system := box.Copy().
		Width(width - 8).
		Foreground(lipgloss.Color("#EEEEEE")).
		Align(lipgloss.Center)
	systemHeader := system.Copy().Foreground(lipgloss.Color("#F1C40F"))
	user := box.Copy().
		Width(width - 8).
		Align(lipgloss.Right)
	userHeader := user.Copy().Foreground(lipgloss.Color("#27AE60"))
	gpt := box.Copy().
		Width(width - 8).
		Align(lipgloss.Left)
	gptHeader := gpt.Copy().Foreground(lipgloss.Color("#3498DB"))

	out := new(strings.Builder)
	for _, msg := range convo {
		headerStyle := lipgloss.NewStyle()
		style := lipgloss.NewStyle()
		switch msg.Role {
		case roleSystem:
			headerStyle = systemHeader
			style = system
		case roleUser:
			headerStyle = userHeader
			style = user
		case roleGpt:
			headerStyle = gptHeader
			style = gpt
		}

		var render string
		if msg.Role == roleGpt {
			if r, err := glamour.NewTermRenderer(
				glamour.WithStandardStyle("dark"),
				glamour.WithWordWrap(width-8),
			); err != nil {
				render = msg.Content
			} else {
				if mkd, err := r.Render(msg.Content); err != nil {
					render = msg.Content
				} else {
					render = mkd
				}
			}
		} else {
			render = msg.Content
		}

		out.WriteString(lipgloss.JoinVertical(lipgloss.Left,
			headerStyle.Render(string(msg.Role)),
			style.Render(render)))
		out.WriteString("\n")

	}

	return out.String()
}

func fetchResponse(prompt string, m model) tea.Cmd {
	return func() tea.Msg {
		if c, err := m.convo.Ask(prompt, m.opts); err != nil {
			c := append(c, message{Role: roleGpt, Content: err.Error()})
			return response{convo: c}
		} else {
			return response{convo: c}
		}
	}
}

type response struct {
	convo conversation
}

func updateViewport() tea.Msg {
	return refresh{}
}
func updateViewportDelayed() tea.Msg {
	time.Sleep(100 * time.Millisecond)
	return updateViewport()
}

type refresh struct{}

type executionResult struct {
	cmd    string
	status int
	stdout string
	stderr string
}

func executeCommand(prompt string, convo conversation) tea.Cmd {
	return func() tea.Msg {
		res := executionResult{cmd: prompt}
		cmd, err := parseCommand(prompt)
		if err != nil {
			res.stderr = err.Error()
		} else {
			if err := cmd.Exec(convo); err != nil {
				res.stderr = err.Error()
			} else {
				res.stdout = "yay!"
			}
		}

		return res
	}
}

type switchToStatus struct {
	status systemStatus
}

func switchToAfter(s systemStatus, secs int) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(time.Duration(secs) * time.Second)
		return switchToStatus{status: s}
	}
}

func max(from ...int) int {
	x := 0
	for _, y := range from {
		if y > x {
			x = y
		}
	}
	return x
}

func min(from ...int) int {
	x := math.MaxInt
	for _, y := range from {
		if y < x {
			x = y
		}
	}
	return x
}
