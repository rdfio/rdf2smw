// Workflow written in SciPipe.
// For more information about SciPipe, see: http://scipipe.org
package main

import (
	"bufio"
	"flag"
	"github.com/flowbase/flowbase"
	"log"
	"os"
)

func main() {
	flowbase.InitLogDebug()

	inFileName := flag.String("infile", "", "The input file name")
	flag.Parse()
	if *inFileName == "" {
		flowbase.Error.Println("No filename specified to --infile")
		os.Exit(1)
	}

	// Create a pipeline runner
	pipeRunner := flowbase.NewPipelineRunner()

	// Initialize processes and add to runner
	fileReader := NewFileReader()
	pipeRunner.AddProcess(fileReader)

	sink := flowbase.NewSinkString()
	pipeRunner.AddProcess(sink)

	// Connect workflow dependency network
	sink.Connect(fileReader.OutLine)

	// Run the pipeline!
	go func() {
		defer close(fileReader.InFileName)
		fileReader.InFileName <- *inFileName
	}()

	pipeRunner.Run()

}

// ================================================================================
// Components
// ================================================================================

type FileReader struct {
	InFileName chan string
	OutLine    chan string
}

func NewFileReader() *FileReader {
	return &FileReader{
		InFileName: make(chan string, flowbase.BUFSIZE),
		OutLine:    make(chan string, flowbase.BUFSIZE),
	}
}

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
