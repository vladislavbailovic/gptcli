package main

import (
	"os"
	"testing"
)

func Test_parseGptResponse(t *testing.T) {
	buf, _ := os.ReadFile("resp.json")
	_, err := parseGptResponse(buf)
	if err != nil {
		t.Error(err)
	}
}

func Test_extractCodeFrom(t *testing.T) {
	buf, _ := os.ReadFile("resp.json")
	x, _ := parseGptResponse(buf)
	code := extractCodeFrom(x.Choices[0].Message.Content)

	if len(code) != 1 {
		t.Log(code)
		t.Errorf("expected exact amount of code segments")
	}
}
