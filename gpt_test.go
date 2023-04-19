package main

import (
	"os"
	"testing"
)

func Test_parseGptResponse(t *testing.T) {
	buf, _ := os.ReadFile("testdata/resp.json")
	_, err := parseGptResponse(buf)
	if err != nil {
		t.Error(err)
	}
}
