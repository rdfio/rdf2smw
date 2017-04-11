package components

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
			fmt.Print(fact.asWikiFact())
		}
		for _, cat := range page.Categories {
			fmt.Print(cat.asWikiString())
		}
		fmt.Println("") // Print an empty line
	}
}
