package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/list"
)

const (
	actionCopySelected      string = "copyselected"
	actionCopyFromChat      string = "copy"
	actionSwitchToSelection string = "selcode"
)

type Action interface {
	Exec(model) (model, error)
}

func parseAction(prompt string) (Action, error) {
	if prompt[0] == ':' {
		prompt = prompt[1:]
	}
	parts := strings.SplitN(strings.TrimSpace(prompt), " ", 2)
	switch parts[0] {
	case "sc", actionSwitchToSelection:
		return SelectCodeAction{}, nil
	case actionCopySelected:
		return CopySelectedAction{}, nil
	case "cc", "yc":
		return CopyCodeAction{}, nil
	case "ca", "ya":
		return CopyAllAction{}, nil
	case "c", "yy", actionCopyFromChat:
		if len(parts) == 1 {
			return CopyAction{}, nil
		}
		if parts[1] == "code" {
			return CopyCodeAction{}, nil
		}
		if parts[1] == "all" {
			return CopyAllAction{}, nil
		}
		return nil, errors.New("not sure what you wanna copy")
	}
	return nil, errors.New("unknown command")
}

type CopyAction struct{}

func (x CopyAction) Exec(m model) (model, error) {
	code := m.convo.ParseCode()
	if len(code) == 0 {
		cmd := new(CopyAllAction)
		return cmd.Exec(m)
	} else {
		cmd := new(CopyCodeAction)
		return cmd.Exec(m)
	}
}

type CopyCodeAction struct{}

func (x CopyCodeAction) Exec(m model) (model, error) {
	code := m.convo.ParseCode()
	if len(code) == 0 {
		return m, errors.New("no code to copy")
	}
	return m, clipboard.WriteAll(strings.TrimSpace(strings.Join(code, "\n")))
}

type CopyAllAction struct{}

func (x CopyAllAction) Exec(m model) (model, error) {
	var content strings.Builder
	for _, m := range m.convo {
		if m.Role == roleSystem {
			continue
		}
		switch m.Role {
		case roleSystem:
			continue
		case roleUser:
			content.WriteString("- me: ")
		case roleGpt:
			content.WriteString("- gpt: ")
		}
		content.WriteString(m.Content)
		content.WriteString("\n\n")
	}
	return m, clipboard.WriteAll(content.String())
}

type SelectCodeAction struct{}

func (x SelectCodeAction) Exec(m model) (model, error) {
	m.setMode(modeSelectCode)
	code := m.convo.ParseCode()
	lst := make([]list.Item, 0, len(code))
	for idx, c := range code {
		lst = append(lst, codeItem{code: strings.TrimSpace(c), idx: idx + 1})
	}
	m.list.SetItems(lst)
	return m, nil
}

type codeItem struct {
	idx  int
	code string
}

func (x codeItem) FilterValue() string { return x.code }
func (x codeItem) Title() string       { return fmt.Sprintf("Code snippet %0d", x.idx) }
func (x codeItem) Description() string { return x.code }

type CopySelectedAction struct{}

func (x CopySelectedAction) Exec(m model) (model, error) {
	c, ok := m.list.SelectedItem().(codeItem)
	if !ok {
		return m, errors.New("no item selected")
	}
	m.list.SetItems([]list.Item{})
	m.setMode(modeChat)
	return m, clipboard.WriteAll(strings.TrimSpace(c.code))
}
