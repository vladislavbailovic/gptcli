package main

import (
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/glamour"
)

func bootChat(opts options, convo conversation) model {
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

	ls := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)

	m := model{
		mode:     modeChat,
		opts:     opts,
		convo:    convo,
		prompt:   tx,
		viewport: vp,
		list:     ls,
		width:    width,
	}
	m.setStatus(statusAwaitingInput)
	return m
}

func chat(opts options, convo conversation) {
	m := bootChat(opts, convo)
	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}

type systemStatus uint8

const (
	statusAwaitingInput systemStatus = iota
	statusAwaitingResponse
	statusAwaitingAction
)

type renderMode uint8

const (
	modeChat renderMode = iota
	modeSelectCode
)

type model struct {
	mode renderMode

	opts  options
	convo conversation

	status     systemStatus
	statusLine string

	prompt   textarea.Model
	viewport viewport.Model
	list     list.Model

	width, height int
}

func (m *model) setStatusMsg(msg string) {
	m.statusLine = msg
}

func (m *model) setStatus(s systemStatus) {
	m.status = s
	switch s {
	case statusAwaitingInput:
		m.statusLine = "Enter to send, Ctrl+D to quit"
		m.prompt.Prompt = "> "
	case statusAwaitingResponse:
		m.statusLine = "... Awaiting response ..."
		m.prompt.Prompt = "> "
	case statusAwaitingAction:
		m.statusLine = "Enter command"
		m.prompt.Prompt = ""
	default:
		m.statusLine = ""
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, updateViewportDelayed)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		txCmd tea.Cmd
		vpCmd tea.Cmd
		lsCmd tea.Cmd
		myCmd tea.Cmd
	)

	m.prompt, txCmd = m.prompt.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)
	m.list, lsCmd = m.list.Update(msg)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height = max(msg.Height, 6)
		m.width = min(msg.Width, 120)

		m.viewport.Width = m.width
		m.viewport.Height = m.height - 3
		m.prompt.SetWidth(m.width)
		m.list.SetSize(m.width, m.height)

		myCmd = updateViewport
	case tea.KeyMsg:
		if "q" == msg.String() {
			return m, nil
		}
		switch msg.Type {
		case tea.KeyCtrlD:
			return m, tea.Quit
		case tea.KeyEsc:
			if m.mode == modeChat {
				if m.status == statusAwaitingAction {
					m.setStatus(statusAwaitingInput)
				} else if m.status != statusAwaitingResponse {
					m.setStatus(statusAwaitingAction)
				}
			} else {
				m.setMode(modeChat)
			}
			lsCmd = nil
		case tea.KeyCtrlQ:
			lsCmd = nil
		case tea.KeyCtrlS:
			myCmd = executeAction(actionSwitchToSelection, m)
		case tea.KeyCtrlC:
			if m.mode == modeSelectCode {
				myCmd = executeAction(actionCopySelected, m)
			} else {
				myCmd = executeAction(actionCopyFromChat, m)
			}
			txCmd = nil
			vpCmd = nil
			lsCmd = nil
		case tea.KeyEnter:
			if m.mode == modeSelectCode {
				myCmd = executeAction(actionCopySelected, m)
			} else {
				currentPrompt := m.prompt.Value()
				if currentPrompt != "" {
					m.prompt.Placeholder = currentPrompt
					m.prompt.Reset()
					m.prompt.Blur()
					if currentPrompt[0] == ':' {
						m.setStatus(statusAwaitingAction)
					}
					switch m.status {
					case statusAwaitingInput:
						m.setStatus(statusAwaitingResponse)
						myCmd = fetchResponse(currentPrompt, m)
					case statusAwaitingAction:
						myCmd = executeAction(currentPrompt, m)
					}
				} else {
					myCmd = updateViewport
				}
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
		var cmd tea.Cmd
		if msg.err != nil {
			m.setStatusMsg(msg.err.Error())
			cmd = switchToAfter(statusAwaitingInput, 2)
		} else {
			m = msg.model
			cmd = switchToAfter(statusAwaitingInput, 0)
		}
		myCmd = tea.Batch(cmd, updateViewport)
	case switchToStatus:
		m.prompt.Placeholder = ""
		m.prompt.Focus()
		m.setStatus(msg.status)
	}

	return m, tea.Batch(txCmd, vpCmd, myCmd, lsCmd)
}

func (m *model) setMode(mode renderMode) {
	m.mode = mode
}

func (m model) View() string {
	switch m.mode {
	case modeChat:
		return m.viewChat()
	case modeSelectCode:
		return m.viewCodeSelection()
	}
	return ""
}

func (m model) viewPrompt() string {
	prompt := m.prompt.View()
	return lipgloss.NewStyle().
		Width(m.width).
		Align(lipgloss.Center).
		Faint(true).
		Italic(true).
		Render(m.statusLine) + "\n" + prompt
}

func (m model) viewCodeSelection() string {
	return fmt.Sprintf(
		"%s\n%s",
		m.list.View(),
		m.viewPrompt(),
	) + "\n\n"
}

func (m model) viewChat() string {
	return fmt.Sprintf(
		"%s\n%s",
		m.viewport.View(),
		m.viewPrompt(),
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
	err   error
	model model
}

func executeAction(prompt string, m model) tea.Cmd {
	return func() tea.Msg {
		res := executionResult{}
		cmd, err := parseAction(prompt)
		if err != nil {
			res.err = err
		} else {
			if m, err := cmd.Exec(m); err != nil {
				res.err = err
			} else {
				res.model = m
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
