package components

import "fmt"

type TripleAggregatePrinter struct {
	In chan *TripleAggregate
}

func NewTripleAggregatePrinter() *TripleAggregatePrinter {
	return &TripleAggregatePrinter{
		In: make(chan *TripleAggregate, BUFSIZE),
	}
}

func (p *TripleAggregatePrinter) Run() {
	for trAggr := range p.In {
		fmt.Printf("Subject: %s\nTriples:\n", trAggr.Subject)
		for _, tr := range trAggr.Triples {
			fmt.Printf("\t<%s> <%s> <%s>\n", tr.Subj.String(), tr.Pred.String(), tr.Obj.String())
		}
		fmt.Println()
	}
}
