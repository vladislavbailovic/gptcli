package main

import (
	"errors"
	"strings"

	"github.com/atotto/clipboard"
)

type Action interface {
	Exec(conversation) error
}

func parseAction(prompt string) (Action, error) {
	if prompt[0] == ':' {
		prompt = prompt[1:]
	}
	parts := strings.SplitN(strings.TrimSpace(prompt), " ", 2)
	switch parts[0] {
	case "cc", "yc":
		return CopyCodeAction{}, nil
	case "ca", "ya":
		return CopyAllAction{}, nil
	case "c", "yy", "copy":
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

func (x CopyAction) Exec(c conversation) error {
	code := c.ParseCode()
	if len(code) == 0 {
		cmd := new(CopyAllAction)
		return cmd.Exec(c)
	} else {
		cmd := new(CopyCodeAction)
		return cmd.Exec(c)
	}
}

type CopyCodeAction struct{}

func (x CopyCodeAction) Exec(c conversation) error {
	code := c.ParseCode()
	if len(code) == 0 {
		return errors.New("no code to copy")
	}
	return clipboard.WriteAll(strings.TrimSpace(strings.Join(code, "\n\n")))
}

type CopyAllAction struct{}

func (x CopyAllAction) Exec(c conversation) error {
	var content strings.Builder
	for _, m := range c {
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
	return clipboard.WriteAll(content.String())
}
