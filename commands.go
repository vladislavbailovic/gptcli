package main

import (
	"errors"
	"strings"

	"github.com/atotto/clipboard"
)

type Command interface {
	Exec(conversation) error
}

func parseCommand(prompt string) (Command, error) {
	if prompt[0] == ':' {
		prompt = prompt[1:]
	}
	switch strings.TrimSpace(prompt) {
	case "cc", "copy code":
		return CopyCodeCommand{}, nil
	case "ca", "copy all":
		return CopyAllCommand{}, nil
	case "c", "copy":
		return CopyCommand{}, nil
	}
	return nil, errors.New("unknown command")
}

type CopyCommand struct{}

func (x CopyCommand) Exec(c conversation) error {
	code := c.ParseCode()
	if len(code) == 0 {
		cmd := new(CopyAllCommand)
		return cmd.Exec(c)
	} else {
		cmd := new(CopyCodeCommand)
		return cmd.Exec(c)
	}
}

type CopyCodeCommand struct{}

func (x CopyCodeCommand) Exec(c conversation) error {
	code := c.ParseCode()
	if len(code) == 0 {
		return errors.New("no code to copy")
	}
	return clipboard.WriteAll(strings.TrimSpace(strings.Join(code, "\n\n")))
}

type CopyAllCommand struct{}

func (x CopyAllCommand) Exec(c conversation) error {
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
