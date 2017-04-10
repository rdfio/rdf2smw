package main

import "github.com/knakk/rdf"

type TripleAggregator struct {
	In  chan rdf.Triple
	Out chan *TripleAggregate
}

func NewTripleAggregator() *TripleAggregator {
	return &TripleAggregator{
		In:  make(chan rdf.Triple, BUFSIZE),
		Out: make(chan *TripleAggregate, BUFSIZE),
	}
}

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
