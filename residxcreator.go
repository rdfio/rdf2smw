package main

type CreateResourceIndex struct {
	In  chan *TripleAggregate
	Out chan *map[string]*TripleAggregate
}

func NewCreateResourceIndex() *CreateResourceIndex {
	return &CreateResourceIndex{
		In:  make(chan *TripleAggregate, BUFSIZE),
		Out: make(chan *map[string]*TripleAggregate),
	}
}

func (p *CreateResourceIndex) Run() {
	defer close(p.Out)

	idx := make(map[string]*TripleAggregate)
	for aggr := range p.In {
		idx[aggr.SubjectStr] = aggr
	}

	p.Out <- &idx
}
