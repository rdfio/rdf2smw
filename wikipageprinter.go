package main

import (
	"fmt"

	"github.com/flowbase/flowbase"
)

type WikiPagePrinter struct {
	In chan *WikiPage
}

func NewWikiPagePrinter() *WikiPagePrinter {
	return &WikiPagePrinter{
		In: make(chan *WikiPage, flowbase.BUFSIZE),
	}
}

func (p *WikiPagePrinter) Run() {
	for page := range p.In {
		fmt.Println("Title:", page.Title)
		for _, fact := range page.Facts {
			fmtFact(fact.Property, fact.Value)
		}
		for _, cat := range page.Categories {
			fmt.Print(fmtCategory(cat))
		}
		fmt.Println("") // Print an empty line
	}
}
