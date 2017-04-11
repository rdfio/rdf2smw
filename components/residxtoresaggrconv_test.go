package components

import (
	"github.com/flowbase/flowbase"
	"github.com/knakk/rdf"
	"testing"
)

// TestNewResourceIndexToTripleAggregates tests NewResourceIndexToTripleAggregates
func TestNewResourceIndexToTripleAggregates(t *testing.T) {
	flowbase.InitLogDebug()

	rita := NewResourceIndexToTripleAggregates()

	if rita.In == nil {
		t.Error("In-port not initialized with map of channels")
	}
	if rita.Out == nil {
		t.Error("Out-port not initialized with channel")
	}
}

// TestResourceIndexToTripleAggregates tests ResourceIndexToTripleAggregates
func TestResourceIndexToTripleAggregates(t *testing.T) {
	flowbase.InitLogDebug()
	rita := NewResourceIndexToTripleAggregates()

	resIdxInner := make(map[string]*TripleAggregate)
	s, err := rdf.NewIRI("http://example.org/s")
	if err != nil {
		t.Error("Could not create subject IRI")
	}
	resIdxInner["aggr1"] = NewTripleAggregate(s, nil)
	resIdx := &resIdxInner

	go func() {
		defer close(rita.In)
		rita.In <- resIdx
	}()
	go rita.Run()
	aggr := <-rita.Out
	if aggr == nil {
		t.Error("Output from ResourceIndexToTripleAggregates was nil")
	}
}
