package components

import (
	"io"
	"log"
	str "strings"

	"github.com/knakk/rdf"
)

type TripleParser struct {
	In  chan string
	Out chan rdf.Triple
}

func NewTripleParser() *TripleParser {
	return &TripleParser{
		In:  make(chan string, BUFSIZE),
		Out: make(chan rdf.Triple, BUFSIZE),
	}
}

func (p *TripleParser) Run() {
	defer close(p.Out)
	for line := range p.In {
		lineReader := str.NewReader(line)
		dec := rdf.NewTripleDecoder(lineReader, rdf.Turtle)
		for triple, err := dec.Decode(); err != io.EOF; triple, err = dec.Decode() {
			if err != nil {
				log.Fatal("Could not encode to triple: ", err.Error())
			} else if triple.Subj != nil && triple.Pred != nil && triple.Obj != nil {
				p.Out <- triple
			} else {
				log.Fatal("Something was encoded as nil in the triple:", triple)
			}
		}
	}
}
