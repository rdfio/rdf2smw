package components

type ResourceIndexToTripleAggregates struct {
	In  chan *map[string]*TripleAggregate
	Out chan *TripleAggregate
}

func NewResourceIndexToTripleAggregates() *ResourceIndexToTripleAggregates {
	return &ResourceIndexToTripleAggregates{
		In:  make(chan *map[string]*TripleAggregate, BUFSIZE),
		Out: make(chan *TripleAggregate, BUFSIZE),
	}
}

func (p *ResourceIndexToTripleAggregates) Run() {
	defer close(p.Out)

	for idx := range p.In {
		for _, aggr := range *idx {
			p.Out <- aggr
		}
	}
}
