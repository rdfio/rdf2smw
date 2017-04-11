package main

import (
	"github.com/flowbase/flowbase"
	"testing"
)

// TestNewTripleAggregateToWikiPageConverter tests NewTripleAggregateToWikiPageConverter()
func TestNewTripleAggregateToWikiPageConverter(t *testing.T) {
	flowbase.InitLogDebug()

	mxc := NewTripleAggregateToWikiPageConverter()

	if mxc.InAggregate == nil {
		t.Error("InAggregate is not initialized")
	}
	if mxc.InIndex == nil {
		t.Error("InIndex is not initialized")
	}
	if mxc.OutPage == nil {
		t.Error("OutPage is not initialized")
	}
	if mxc.cleanUpRegexes == nil {
		t.Error("cleanUpRegexes is not initialized")
	}
}
