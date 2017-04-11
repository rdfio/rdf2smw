package components

import (
	"github.com/flowbase/flowbase"
	"testing"
)

// TestNewResourceIndexFanOut tests NewResourceIndexFanOut
func TestNewResourceIndexFanOut(t *testing.T) {
	flowbase.InitLogDebug()

	rif := NewResourceIndexFanOut()

	if rif.In == nil {
		t.Error("In-port not initialized with channel")
	}
	if rif.Out == nil {
		t.Error("Out-port not initialized with map of channels")
	}
}

func TestResourceIndexFanOut(t *testing.T) {
	flowbase.InitLogDebug()

	rif := NewResourceIndexFanOut()
	rif.Out["out1"] = make(chan *map[string]*TripleAggregate)
	rif.Out["out2"] = make(chan *map[string]*TripleAggregate)

	resIdxInner := make(map[string]*TripleAggregate)
	resIdx := &resIdxInner

	go func() {
		defer close(rif.In)
		rif.In <- resIdx
	}()
	go rif.Run()

	resIdx1 := <-rif.Out["out1"]
	if resIdx1 == nil {
		t.Error("Got nil as output from out1")
	}
	resIdx2 := <-rif.Out["out2"]
	if resIdx2 == nil {
		t.Error("Got nil as output from out2")
	}
}
