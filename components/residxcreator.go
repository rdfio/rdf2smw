package components

type ResourceIndexCreator struct {
	In  chan *TripleAggregate
	Out chan *map[string]*TripleAggregate
}

func NewResourceIndexCreator() *ResourceIndexCreator {
	return &ResourceIndexCreator{
		In:  make(chan *TripleAggregate, BUFSIZE),
		Out: make(chan *map[string]*TripleAggregate),
	}
}

func (p *ResourceIndexCreator) Run() {
	defer close(p.Out)

	idx := make(map[string]*TripleAggregate)
	for aggr := range p.In {
		idx[aggr.SubjectStr] = aggr
	}

	p.Out <- &idx
}
