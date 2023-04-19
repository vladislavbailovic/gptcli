package main

import "encoding/json"

type gptModel string

const (
	gpt3 gptModel = "gpt-3.5-turbo"
	gpt4 gptModel = "gpt-4"
)

type gptMsg struct {
	Model    gptModel     `json:"model"`
	Messages conversation `json:"messages"`
}

type gptFinishReason string

const (
	gptFinishStop       gptFinishReason = "stop"
	gptFinishLength     gptFinishReason = "length"
	gptFinishFlt        gptFinishReason = "filter"
	gptFinishIncomplete gptFinishReason = "null"
)

func parseGptResponse(buf []byte) (gptResponse, error) {
	x := gptResponse{}
	if err := json.Unmarshal(buf, &x); err != nil {
		return x, err
	}

	return x, nil
}

type gptResponse struct {
	Choices []gptChoice `json:"choices"`
}

type gptChoice struct {
	Message message         `json:"message"`
	Reason  gptFinishReason `json:"finish_reason"`
}
