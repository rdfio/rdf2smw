package components

import (
	"fmt"

	"github.com/knakk/rdf"
)

type TriplePrinter struct {
	In chan rdf.Triple
}

func NewTriplePrinter() *TriplePrinter {
	return &TriplePrinter{
		In: make(chan rdf.Triple, BUFSIZE),
	}
}

func (p *TriplePrinter) Run() {
	for tr := range p.In {
		fmt.Printf("S: %s\nP: %s\nO: %s\n\n", tr.Subj.String(), tr.Pred.String(), tr.Obj.String())
	}
}
