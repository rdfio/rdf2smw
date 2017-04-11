package components

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
