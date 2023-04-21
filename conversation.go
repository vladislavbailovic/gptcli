package main

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
)

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

func (x conversation) ParseCode() []string {
	code := []string{}
	for _, m := range x {
		if m.Role != roleGpt {
			continue
		}
		code = append(code, extractCodeFrom(m.Content)...)
	}
	return code
}

func (x conversation) Last() string {
	if len(x) == 0 {
		return ""
	}
	return x[len(x)-1].Content
}

func (x conversation) Ask(q string, opts options) (conversation, error) {
	if opts.token == "" {
		return x, errors.New("missing token")
	}
	query := make(conversation, 0, len(x)+2)
	query = append(query, x...)
	query = append(query, message{Role: roleUser, Content: q})

	var content []byte
	if fc, err := fromCache(q); err != nil {
		mdl := gptMsg{
			Model:    opts.model,
			Messages: query}
		body, err := json.Marshal(mdl)
		if err != nil {
			return x, err
		}

		req, err := http.NewRequest(
			http.MethodPost,
			"https://api.openai.com/v1/chat/completions",
			bytes.NewBuffer(body))
		if err != nil {
			return x, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", opts.token))

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return x, err
		}

		if resp.StatusCode != http.StatusOK {
			return x, errors.New("API returned error")
		}

		defer resp.Body.Close()
		cnt, err := io.ReadAll(resp.Body)
		if err != nil {
			return query, err
		}

		toCache(q, cnt)
		content = cnt
	} else {
		content = fc
	}

	raw, err := parseGptResponse(content)
	if err != nil {
		return query, err
	}

	for _, m := range raw.Choices {
		query = append(query, m.Message)
	}

	return query, nil
}

func extractCodeFrom(msg string) []string {
	inCode := false
	code := []string{}
	tmp := ""
	for _, l := range strings.Split(msg, "\n") {
		if inCode {
			if strings.HasPrefix(l, "```") {
				code = append(code, strings.TrimSpace(tmp))
				tmp = ""
				inCode = false
			} else {
				tmp += "\n" + l
			}
		} else if strings.HasPrefix(l, "```") {
			inCode = true
		}
	}
	return code
}

func fromCache(q string) ([]byte, error) {
	h := md5.New()
	io.WriteString(h, q)
	fname := path.Join(os.TempDir(), fmt.Sprintf("gptcli-%x", h.Sum(nil)))
	if cnt, err := os.ReadFile(fname); err != nil {
		return []byte{}, err
	} else {
		return cnt, err
	}
}

func toCache(q string, cnt []byte) error {
	h := md5.New()
	io.WriteString(h, q)
	fname := path.Join(os.TempDir(), fmt.Sprintf("gptcli-%x", h.Sum(nil)))
	return os.WriteFile(fname, cnt, 0666)
}
