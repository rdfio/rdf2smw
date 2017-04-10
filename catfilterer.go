package main

type CategoryFilterer struct {
	In         chan *WikiPage
	Out        chan *WikiPage
	Categories []*Category
}

func NewCategoryFilterer(categories []*Category) *CategoryFilterer {
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
			if catInArray(pageCat, p.Categories) {
				p.Out <- page
				break
			}
		}
	}
}

func catInArray(searchCat *Category, cats []*Category) bool {
	for _, cat := range cats {
		if searchCat.Name == cat.Name {
			return true
		}
	}
	return false
}
