package components

import (
	"fmt"
	"github.com/flowbase/flowbase"
	"github.com/knakk/rdf"
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

func TestResourceIndexCreator(t *testing.T) {
	flowbase.InitLogWarning()

	ric := NewResourceIndexCreator()

	var triples = []rdf.Triple{}

	go func() {
		defer close(ric.In)

		for i := 1; i <= 2; i++ {

			triples = []rdf.Triple{}
			s, serr := rdf.NewIRI(fmt.Sprintf("http://example.org/s%d", i))
			if serr != nil {
				t.Error("Could not create Subject IRI")
			}
			for j := 1; j <= 3; j++ {
				p, perr := rdf.NewIRI(fmt.Sprintf("http://example.org/p%d", j))
				if perr != nil {
					t.Error("Could not create Predicate IRI")
				}
				o, oerr := rdf.NewLiteral(fmt.Sprintf("o%d", j))
				if oerr != nil {
					t.Error("Could not create Object Literal")
				}
				tr := rdf.Triple{
					Subj: s,
					Pred: p,
					Obj:  o,
				}
				triples = append(triples, tr)
			}

			aggr := NewTripleAggregate(s, triples)
			ric.In <- aggr
		}
	}()

	go ric.Run()

	resIdx := <-ric.Out

	if (*resIdx)["http://example.org/s1"] == nil {
		t.Error("Resource index does not contain first subject")
	}

	if (*resIdx)["http://example.org/s1"].Subject.String() != "http://example.org/s1" {
		t.Error("Subject string in first subject is wrong")
	}

	if len((*resIdx)["http://example.org/s1"].Triples) != 3 {
		t.Error("Wrong number of triples for first subject")
	}

	if (*resIdx)["http://example.org/s2"] == nil {
		t.Error("Resource index does not contain second subject")
	}

	if (*resIdx)["http://example.org/s2"].Subject.String() != "http://example.org/s2" {
		t.Error("Subject string in second subject is wrong")
	}

	if len((*resIdx)["http://example.org/s2"].Triples) != 3 {
		t.Error("Wrong number of triples for second subject")
	}
}
