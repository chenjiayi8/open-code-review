package main

import (
	"strings"
	"testing"
)

func TestRunConfigSetRequiresValue(t *testing.T) {
	err := runConfig([]string{"set", "language"})
	if err == nil || !strings.Contains(err.Error(), "accepts 2 arg") {
		t.Fatalf("err = %v", err)
	}
}

func TestRunConfigProviderSubcommandRemoved(t *testing.T) {
	err := runConfig([]string{"provider"})
	if err == nil || !strings.Contains(err.Error(), "unknown command") {
		t.Fatalf("err = %v", err)
	}
}
