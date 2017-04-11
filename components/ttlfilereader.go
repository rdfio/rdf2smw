package components

import (
	"io"
	"log"

	"github.com/flowbase/flowbase"
	"github.com/knakk/rdf"
	"github.com/spf13/afero"
)

// TurtleFileReader is a process that reads turtle files (Files in the turtle
// RDF format), based on file names it receives on the FileReader.InFileName
// port / channel, and writes out the output line by line as strings on the
// FileReader.OutLine port / channel.
type TurtleFileReader struct {
	InFileName chan string
	OutTriple  chan rdf.Triple
	fs         afero.Fs
}

// NewOsTurtleFileReader returns an initialized TurtleFileReader, with an OS
// (normal) file system
func NewOsTurtleFileReader() *TurtleFileReader {
	return NewTurtleFileReader(afero.NewOsFs())
}

// NewTurtleFileReader returns an initialized TurtleFileReader, initialized
// with the afero file system provided provided as an argument
func NewTurtleFileReader(fileSystem afero.Fs) *TurtleFileReader {
	return &TurtleFileReader{
		InFileName: make(chan string, BUFSIZE),
		OutTriple:  make(chan rdf.Triple, BUFSIZE),
		fs:         fileSystem,
	}
}

// Run runs the TurtleFileReader process. It does not spawn a separate
// go-routine, so you have to prepend the go keyword when calling it, in order
// to have it run in a separate go-routine.
func (p *TurtleFileReader) Run() {
	defer close(p.OutTriple)

	flowbase.Debug.Println("Starting loop")
	for fileName := range p.InFileName {
		flowbase.Debug.Printf("Starting processing file %s\n", fileName)
		fh, err := p.fs.Open(fileName)
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
