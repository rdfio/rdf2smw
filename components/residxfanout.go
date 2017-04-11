package components

type ResourceIndexFanOut struct {
	In  chan *map[string]*TripleAggregate
	Out map[string]chan *map[string]*TripleAggregate
}

func NewResourceIndexFanOut() *ResourceIndexFanOut {
	return &ResourceIndexFanOut{
		In:  make(chan *map[string]*TripleAggregate),
		Out: make(map[string]chan *map[string]*TripleAggregate),
	}
}

func (p *ResourceIndexFanOut) Run() {
	for _, outPort := range p.Out {
		defer close(outPort)
	}

	for idx := range p.In {
		for _, outPort := range p.Out {
			outPort <- idx
		}
	}
}
