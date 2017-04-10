package main

import (
	"github.com/flowbase/flowbase"
	"testing"
)

// TestNewResouceIndexCreator tests NewResourceIndexCreator
func TestNewResourceIndexCreator(t *testing.T) {
	flowbase.InitLogWarning()

	ric := NewResourceIndexCreator()

	if ric.In == nil {
		t.Error("In-port not initialized")
	}
	if ric.Out == nil {
		t.Error("Out-port not initialized")
	}
}
