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

type FanOutTripleAggregate struct {
	In  chan *TripleAggregate
	Out map[string](chan *TripleAggregate)
}

// NewFanOut creates a new FanOut process
func NewFanOutTripleAggregate() *FanOutTripleAggregate {
	return &FanOutTripleAggregate{
		In:  make(chan *TripleAggregate, BUFSIZE),
		Out: make(map[string](chan *TripleAggregate)),
	}
}

// Run runs the FanOut process
func (proc *FanOutTripleAggregate) Run() {
	for _, outPort := range proc.Out {
		defer close(outPort)
	}

	for ft := range proc.In {
		for _, outPort := range proc.Out {
			outPort <- ft
		}
	}
}
