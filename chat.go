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
)

func chat(_ options, convo conversation) {
	width := 30

	tx := textarea.New()

	tx.FocusedStyle.CursorLine = lipgloss.NewStyle()
	tx.ShowLineNumbers = false
	tx.Prompt = "> "
	tx.CharLimit = 140

	tx.SetWidth(width)
	tx.SetHeight(1)

	tx.Focus()
	tx.KeyMap.InsertNewline.SetEnabled(false)

	vp := viewport.New(width, 5)

	m := model{
		convo:    convo,
		status:   statusAwaitingInput,
		prompt:   tx,
		viewport: vp,
		width:    width,
	}

	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}

type message struct {
	Role    role   `json:"role"`
	Content string `json:"content"`
}

type role string

const (
	roleSystem role = "system"
	roleUser   role = "user"
	roleGpt    role = "assistant"
)

type conversation []message

type systemStatus uint8

const (
	statusAwaitingInput systemStatus = iota
	statusAwaitingResponse
	statusAwaitingCommand
)

type model struct {
	convo  conversation
	status systemStatus

	prompt   textarea.Model
	viewport viewport.Model

	width, height int
}

func (m model) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, updateViewportDelayed)
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
		m.width = min(msg.Width, 80)

		m.viewport.Width = m.width
		m.viewport.Height = m.height - 3
		m.prompt.SetWidth(m.width)
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyCtrlD:
			return m, tea.Quit
		case tea.KeyEsc:
			m.status = statusAwaitingCommand
			m.prompt.Prompt = ""
		case tea.KeyEnter:
			currentPrompt := m.prompt.Value()
			m.prompt.Placeholder = currentPrompt
			m.prompt.Reset()
			m.prompt.Blur()
			switch m.status {
			case statusAwaitingInput:
				m.status = statusAwaitingResponse
				myCmd = fetchResponse(currentPrompt)
			case statusAwaitingCommand:
				myCmd = executeCommand(currentPrompt)
			}
		}
	case refresh:
		switch m.status {
		case statusAwaitingInput:
			m.prompt.Prompt = "> "
		case statusAwaitingCommand:
			m.prompt.Prompt = ""
		}
		m.viewport.SetContent(renderMessages(m.convo, m.width))
		m.viewport.GotoBottom()
	case response:
		m.convo = append(
			m.convo,
			msg.me,
			msg.gpt)
		m.status = statusAwaitingInput
		m.prompt.Placeholder = ""
		m.prompt.Focus()
		myCmd = updateViewport
	case executionResult:
		m.status = statusAwaitingInput
		m.prompt.Placeholder = ""
		m.prompt.Focus()
		m.viewport.SetContent(fmt.Sprintf(
			"$ %s\n\nexit code: %d\n\noutput: %s\n\nerror: %s\n\n",
			msg.cmd, msg.status, msg.stdout, msg.stderr))
		myCmd = clearAfter(2)
	}

	return m, tea.Batch(txCmd, vpCmd, myCmd)
}

func (m model) View() string {
	prompt := m.prompt.View()
	if m.status == statusAwaitingResponse {
		prompt = lipgloss.NewStyle().
			Width(m.width).
			Align(lipgloss.Center).
			Faint(true).
			Italic(true).
			Render("... waiting for response...") + "\n" + prompt
	}
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

		out.WriteString(lipgloss.JoinVertical(lipgloss.Left,
			headerStyle.Render(string(msg.Role)),
			style.Render(msg.Content)))
		out.WriteString("\n")

	}

	return out.String()
}

func fetchResponse(prompt string) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(2 * time.Second)
		return response{
			me:  message{Role: roleUser, Content: prompt},
			gpt: message{Role: roleGpt, Content: "API response"},
		}
	}
}

type response struct {
	me  message
	gpt message
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

func executeCommand(cmd string) tea.Cmd {
	return func() tea.Msg {
		return executionResult{
			cmd:    cmd,
			stdout: "Yay",
		}
	}
}

func clearAfter(secs int) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(time.Duration(secs) * time.Second)
		return refresh{}
	}
}
