package main

import "fmt"

type SMWTemplateCallFormatter struct {
	InWikiPage     chan *WikiPage
	OutWikiPageXML chan string
}

func NewSMWTemplateCallFormatter() *SMWTemplateCallFormatter {
	return &SMWTemplateCallFormatter{
		InWikiPage:     make(chan *WikiPage, BUFSIZE),
		OutWikiPageXML: make(chan string, BUFSIZE),
	}
}

func (p *SMWTemplateCallFormatter) Run() {
	fmt.Println("Running SMWTemplateCallFormatter ...")
}
