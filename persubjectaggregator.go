package main

import "github.com/knakk/rdf"

type AggregateTriplesPerSubject struct {
	In  chan rdf.Triple
	Out chan *TripleAggregate
}

func NewAggregateTriplesPerSubject() *AggregateTriplesPerSubject {
	return &AggregateTriplesPerSubject{
		In:  make(chan rdf.Triple, BUFSIZE),
		Out: make(chan *TripleAggregate, BUFSIZE),
	}
}

func (p *AggregateTriplesPerSubject) Run() {
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
