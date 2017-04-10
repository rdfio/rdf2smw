package main

import (
	"io"
	"log"
	"os"

	"github.com/flowbase/flowbase"
	"github.com/knakk/rdf"
)

type TurtleFileReader struct {
	InFileName chan string
	OutTriple  chan rdf.Triple
}

func NewTurtleFileReader() *TurtleFileReader {
	return &TurtleFileReader{
		InFileName: make(chan string, BUFSIZE),
		OutTriple:  make(chan rdf.Triple, BUFSIZE),
	}
}

func (p *TurtleFileReader) Run() {
	defer close(p.OutTriple)

	flowbase.Debug.Println("Starting loop")
	for fileName := range p.InFileName {
		flowbase.Debug.Printf("Starting processing file %s\n", fileName)
		fh, err := os.Open(fileName)
		if err != nil {
			log.Fatal(err)
		}
		defer fh.Close()

		dec := rdf.NewTripleDecoder(fh, rdf.Turtle)
		for triple, err := dec.Decode(); err != io.EOF; triple, err = dec.Decode() {
			if err != nil {
				log.Fatal("Could not encode to triple: ", err.Error())
			} else if triple.Subj != nil && triple.Pred != nil && triple.Obj != nil {
				p.OutTriple <- triple
			} else {
				log.Fatal("Something was encoded as nil in the triple:", triple)
			}
		}
	}
}
