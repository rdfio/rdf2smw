package main

import (
	"bufio"
	"log"

	"github.com/flowbase/flowbase"
	"github.com/spf13/afero"
)

// --------------------------------------------------------------------------------
// FileReader
// --------------------------------------------------------------------------------

// FileReader is a process that reads files, based on file names it receives on the
// FileReader.InFileName port / channel, and writes out the output line by line
// as strings on the FileReader.OutLine port / channel.
type FileReader struct {
	InFileName chan string
	OutLine    chan string
	fs         afero.Fs
}

// NewOsFileReader returns an initialized FileReader, initialized with an OS
// (normal) file system
func NewOsFileReader() *FileReader {
	return NewFileReader(afero.NewOsFs())
}

// NewFileReader returns an initialized FileReader, initialized with an afero
// file system provided as a parameter
func NewFileReader(fileSystem afero.Fs) *FileReader {
	return &FileReader{
		InFileName: make(chan string, BUFSIZE),
		OutLine:    make(chan string, BUFSIZE),
		fs:         fileSystem,
	}
}

// Run runs the FileReader process.
func (p *FileReader) Run() {
	defer close(p.OutLine)

	//flowbase.Debug.Println("Starting loop")
	for fileName := range p.InFileName {
		flowbase.Debug.Printf("Starting processing file %s\n", fileName)
		fh, err := p.fs.Open(fileName)
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
