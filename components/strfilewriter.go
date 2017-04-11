package components

import (
	"os"

	"github.com/flowbase/flowbase"
)

type StringFileWriter struct {
	In       chan string
	OutDone  chan interface{}
	fileName string
}

func NewStringFileWriter(fileName string) *StringFileWriter {
	return &StringFileWriter{
		In:       make(chan string, BUFSIZE),
		OutDone:  make(chan interface{}, BUFSIZE),
		fileName: fileName,
	}
}

func (p *StringFileWriter) Run() {
	defer close(p.OutDone)

	fh, err := os.Create(p.fileName)
	if err != nil {
		panic("Could not create output file: " + err.Error())
	}
	defer fh.Close()
	for s := range p.In {
		fh.WriteString(s)
	}

	flowbase.Debug.Printf("Sending done signal on chan %v now in StringFileWriter ...\n", p.OutDone)
	p.OutDone <- &DoneSignal{}
}

type DoneSignal struct{}
