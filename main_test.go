package main

import (
	"testing"
)

func TestNewFileReader(t *testing.T) {
	fr := NewFileReader()
	if fr.InFileName == nil {
		t.Error("In-port InFileName not initialized in New FileReader")
	}
	if fr.OutLine == nil {
		t.Error("In-port InFileName not initialized in New FileReader")
	}
}
