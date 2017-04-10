package main

import (
	"fmt"
	"github.com/flowbase/flowbase"
	"github.com/knakk/rdf"
	"strings"
	"testing"
)

// TestNewNewAggregateTriplesPerSubject tests NewAggregateTriplesPerSubject
func TestNewTripleAggregator(t *testing.T) {
	flowbase.InitLogWarning()

	aggr := NewTripleAggregator()

	if aggr.In == nil {
		t.Error("In-port not initialized")
	}
	if aggr.Out == nil {
		t.Error("Out-port not initialized")
	}
}

func TestTripleAggregator(t *testing.T) {
	flowbase.InitLogWarning()

	tripleAggregatorTestIndata := `
<http://example.org/s1> <http://example.org/p1> "o1" .
<http://example.org/s1> <http://example.org/p2> "o2" .
<http://example.org/s1> <http://example.org/p3> "o3" .
<http://example.org/s2> <http://example.org/p4> "o4" .
<http://example.org/s2> <http://example.org/p5> "o5" .
<http://example.org/s2> <http://example.org/p6> "o6" .
`

	strReader := strings.NewReader(tripleAggregatorTestIndata)
	dec := rdf.NewTripleDecoder(strReader, rdf.NTriples)
	triples, err := dec.DecodeAll()
	if err != nil {
		t.Error("Could not decode n-triples test data")
	}

	aggregator := NewTripleAggregator()
	go func() {
		defer close(aggregator.In)
		for _, tr := range triples {
			aggregator.In <- tr
		}
	}()
	go aggregator.Run()

	aggr1 := <-aggregator.Out
	if aggr1.Subject.String() != "http://example.org/s1" {
		t.Error("Subject of first aggregate is wrong")
	}
	for i, tr := range aggr1.Triples {
		j := i + 1
		if tr.Subj.String() != "http://example.org/s1" {
			t.Errorf("Subject in triple %d of first aggregate is wrong", j)
		}
		if tr.Pred.String() != fmt.Sprintf("http://example.org/p%d", j) {
			t.Errorf("Subject in triple %d of first aggregate is wrong", j)
		}
		if tr.Obj.String() != fmt.Sprintf("o%d", j) {
			t.Errorf("Subject in triple %d of first aggregate is wrong", j)
		}
	}

	aggr2 := <-aggregator.Out
	if aggr2.Subject.String() != "http://example.org/s2" {
		t.Error("Subject of second aggregate is wrong")
	}
	for i, tr := range aggr2.Triples {
		j := i + 4
		if tr.Subj.String() != "http://example.org/s2" {
			t.Errorf("Subject in triple %d of second aggregate is wrong", j)
		}
		if tr.Pred.String() != fmt.Sprintf("http://example.org/p%d", j) {
			t.Errorf("Subject in triple %d of second aggregate is wrong", j)
		}
		if tr.Obj.String() != fmt.Sprintf("o%d", j) {
			t.Errorf("Subject in triple %d of second aggregate is wrong", j)
		}
	}

}
