package main

import (
	"testing"
)

func TestPrintHelp(t *testing.T) {

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("printHelp panicked: %v", r)
		}
	}()
	printHelp()
}

func TestHandleGeneratePassword(t *testing.T) {
	// This test just ensures handleGeneratePassword runs without panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("handleGeneratePassword panicked: %v", r)
		}
	}()
	handleGeneratePassword()
}
