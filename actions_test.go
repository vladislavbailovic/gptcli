package main

import "testing"

func Test_parseAction_CopyGeneric(t *testing.T) {
	want, _ := parseAction("copy")
	suite := []string{
		"copy",
		"c",
		"yy",
	}
	for _, test := range suite {
		t.Run(test, func(t *testing.T) {
			got, err := parseAction(test)
			if err != nil {
				t.Error(err)
			}
			if got != want {
				t.Errorf("want %v (%T), got %v (%T)", want, want, got, got)
			}
		})
		t.Run(":"+test, func(t *testing.T) {
			got, err := parseAction(":" + test)
			if err != nil {
				t.Error(err)
			}
			if got != want {
				t.Errorf("want %v (%T), got %v (%T)", want, want, got, got)
			}
		})
	}
}

func Test_parseAction_CopyCode(t *testing.T) {
	want, _ := parseAction("copy code")
	suite := []string{
		"copy code",
		"cc",
		"yc",
	}
	for _, test := range suite {
		t.Run(test, func(t *testing.T) {
			got, err := parseAction(test)
			if err != nil {
				t.Error(err)
			}
			if got != want {
				t.Errorf("want %v (%T), got %v (%T)", want, want, got, got)
			}
		})
		t.Run(":"+test, func(t *testing.T) {
			got, err := parseAction(":" + test)
			if err != nil {
				t.Error(err)
			}
			if got != want {
				t.Errorf("want %v (%T), got %v (%T)", want, want, got, got)
			}
		})
	}
}

func Test_parseAction_CopyAll(t *testing.T) {
	doNotWant, _ := parseAction("copy code")
	want, _ := parseAction("copy all")
	suite := []string{
		"copy all",
		"ca",
		"ya",
	}
	for _, test := range suite {
		t.Run(test, func(t *testing.T) {
			got, err := parseAction(test)
			if err != nil {
				t.Error(err)
			}
			if got != want {
				t.Errorf("want %v (%T), got %v (%T)", want, want, got, got)
			}
			if got == doNotWant {
				t.Error("should not match the other command")
			}
		})
		t.Run(":"+test, func(t *testing.T) {
			got, err := parseAction(":" + test)
			if err != nil {
				t.Error(err)
			}
			if got != want {
				t.Errorf("want %v (%T), got %v (%T)", want, want, got, got)
			}
			if got == doNotWant {
				t.Error("should not match the other command")
			}
		})
	}
}
