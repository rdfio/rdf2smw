package main

import "fmt"

type StringPrinter struct {
	In chan string
}

func NewStringPrinter() *StringPrinter {
	return &StringPrinter{
		In: make(chan string, BUFSIZE),
	}
}

func (p *StringPrinter) Run() {
	for s := range p.In {
		fmt.Print(s)
	}
}
