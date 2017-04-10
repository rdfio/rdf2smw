package main

type CategoryFilterer struct {
	In         chan *WikiPage
	Out        chan *WikiPage
	Categories []string
}

func NewCategoryFilterer(categories []string) *CategoryFilterer {
	return &CategoryFilterer{
		In:         make(chan *WikiPage, BUFSIZE),
		Out:        make(chan *WikiPage, BUFSIZE),
		Categories: categories,
	}
}

func (p *CategoryFilterer) Run() {
	defer close(p.Out)
	for page := range p.In {
		for _, pageCat := range page.Categories {
			if stringInSlice(pageCat, p.Categories) {
				p.Out <- page
				break
			}
		}
	}
}
