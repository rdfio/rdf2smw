package main

import (
	"bufio"
	"log"
	"os"

	"github.com/flowbase/flowbase"
)

// --------------------------------------------------------------------------------
// FileReader
// --------------------------------------------------------------------------------

// FileReader is a process that reads files, based on filenames it receives on the
// FileReader.InFileName port / channel, and writes out the output line by line
// as strings on the FileReader.OutLine port / channel.
type FileReader struct {
	InFileName chan string
	OutLine    chan string
}

// NewFileReader returns an initialized FileReader.
func NewFileReader() *FileReader {
	return &FileReader{
		InFileName: make(chan string, BUFSIZE),
		OutLine:    make(chan string, BUFSIZE),
	}
}

// Run runs the FileReader process.
func (p *FileReader) Run() {
	defer close(p.OutLine)

	flowbase.Debug.Println("Starting loop")
	for fileName := range p.InFileName {
		flowbase.Debug.Printf("Starting processing file %s\n", fileName)
		fh, err := os.Open(fileName)
		if err != nil {
			log.Fatal(err)
		}
		defer fh.Close()

		sc := bufio.NewScanner(fh)
		for sc.Scan() {
			if err := sc.Err(); err != nil {
				log.Fatal(err)
			}
			p.OutLine <- sc.Text()
		}
	}
}
