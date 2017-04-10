package main

import "github.com/knakk/rdf"

// TripleAggregator aggregates triples by subject into a TripleAggregate object
// per subject, containing all the triples for that subject.
type TripleAggregator struct {
	In  chan rdf.Triple
	Out chan *TripleAggregate
}

// NewTripleAggregator returns an initialized TripleAggregator process.
func NewTripleAggregator() *TripleAggregator {
	return &TripleAggregator{
		In:  make(chan rdf.Triple, BUFSIZE),
		Out: make(chan *TripleAggregate, BUFSIZE),
	}
}

// Run runs the TripleAggregator process.
func (p *TripleAggregator) Run() {
	defer close(p.Out)
	resourceIndex := make(map[rdf.Subject][]rdf.Triple)
	for triple := range p.In {
		resourceIndex[triple.Subj] = append(resourceIndex[triple.Subj], triple)
	}
	for subj, triples := range resourceIndex {
		tripleAggregate := NewTripleAggregate(subj, triples)
		p.Out <- tripleAggregate
	}
}
