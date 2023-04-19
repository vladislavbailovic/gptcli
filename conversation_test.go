package main

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"testing"
)

func TestExtractCodeFrom(t *testing.T) {
	// Test with no code blocks
	msg := `This is a message without any code blocks.
	`
	expected := []string{}
	actual := extractCodeFrom(msg)
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Expected %v but got %v", expected, actual)
	}

	// Test with one code block
	msg = "This is a message with a code block:\n" +
		"```\n" +
		`fmt.Println("Hello, World!")` + "\n" +
		"```\n"
	expected = []string{"\nfmt.Println(\"Hello, World!\")"}
	actual = extractCodeFrom(msg)
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Expected %v but got %v", expected, actual)
	}

	// Test with multiple code blocks
	msg = "This is a message with multiple code blocks:\n" +
		"```\n" +
		`fmt.Println("Hello, World!")` + "\n" +
		"```\n" +
		"Some text here.\n" +
		"```\n" +
		`fmt.Println("Goodbye, World!")` + "\n" +
		"```\n"
	expected = []string{"\nfmt.Println(\"Hello, World!\")", "\nfmt.Println(\"Goodbye, World!\")"}
	actual = extractCodeFrom(msg)
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Expected %v but got %v", expected, actual)
	}
}

func TestFromCache(t *testing.T) {
	// Create a temporary file to use for testing
	tmpfile, err := ioutil.TempFile("", "test")
	if err != nil {
		t.Fatalf("Error creating temporary file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	// Write some test data to the file
	testdata := []byte("test data")
	if _, err := tmpfile.Write(testdata); err != nil {
		t.Fatalf("Error writing to temporary file: %v", err)
	}

	// Generate an MD5 hash of the test data
	h := md5.New()
	_, err = h.Write(testdata)
	if err != nil {
		t.Fatalf("Error creating MD5 hash: %v", err)
	}
	expectedFilename := path.Join(os.TempDir(), fmt.Sprintf("gptcli-%x", h.Sum(nil)))

	// Move the temporary file to the expected filename
	if err := os.Rename(tmpfile.Name(), expectedFilename); err != nil {
		t.Fatalf("Error moving temporary file: %v", err)
	}
	defer os.Remove(expectedFilename)

	// Call the fromCache function and verify that it returns the expected data
	data, err := fromCache(string(testdata))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !bytes.Equal(data, testdata) {
		t.Fatalf("Expected %v, but got %v", testdata, data)
	}
}

func TestLast(t *testing.T) {
	// test case 1: empty conversation
	conv1 := make(conversation, 0)
	if conv1.Last() != "" {
		t.Errorf("expected empty string, but got '%s'", conv1.Last())
	}

	// test case 2: non-empty conversation
	conv2 := make(conversation, 3)
	conv2[0] = message{Content: "hello"}
	conv2[1] = message{Content: "world"}
	conv2[2] = message{Content: "!"}
	if conv2.Last() != "!" {
		t.Errorf("expected '!', but got '%s'", conv2.Last())
	}
}
