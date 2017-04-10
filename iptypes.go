package main

import (
	str "strings"

	"github.com/knakk/rdf"
)

// --------------------------------------------------------------------------------
// TripleAggregate
// --------------------------------------------------------------------------------

type TripleAggregate struct {
	Subject    rdf.Subject
	SubjectStr string
	Triples    []rdf.Triple
}

func NewTripleAggregate(subj rdf.Subject, triples []rdf.Triple) *TripleAggregate {
	return &TripleAggregate{
		Subject:    subj,
		SubjectStr: subj.String(),
		Triples:    triples,
	}
}

// --------------------------------------------------------------------------------
// WikiPage
// --------------------------------------------------------------------------------

type WikiPage struct {
	Title            string
	Type             int
	Facts            []*Fact
	Categories       []*Category
	SpecificCategory *Category
}

func NewWikiPage(title string, facts []*Fact, categories []*Category, specificCategory *Category, pageType int) *WikiPage {
	return &WikiPage{
		Title:            title,
		Facts:            facts,
		Categories:       categories,
		SpecificCategory: specificCategory,
		Type:             pageType,
	}
}

func (p *WikiPage) AddFact(fact *Fact) {
	p.Facts = append(p.Facts, fact)
}

func (p *WikiPage) AddFactUnique(fact *Fact) {
	factExists := false
	for _, existingFact := range p.Facts {
		if fact.Property == existingFact.Property && fact.Value == existingFact.Value {
			factExists = true
			break
		}
	}
	if !factExists {
		p.AddFact(fact)
	}
}

func (p *WikiPage) AddCategory(category *Category) {
	p.Categories = append(p.Categories, category)
}

func (p *WikiPage) AddCategoryUnique(category *Category) {
	catExists := false
	for _, existingCat := range p.Categories {
		if category.Name == existingCat.Name {
			catExists = true
			break
		}
	}
	if !catExists {
		p.AddCategory(category)
	}
}

// ------------------------------------------------------------
// Helper type: Fact
// ------------------------------------------------------------

type Fact struct {
	Property string
	Value    string
}

func NewFact(property string, value string) *Fact {
	return &Fact{
		Property: property,
		Value:    value,
	}
}

func (f *Fact) asWikiFact() string {
	return "[[" + f.Property + "::" + f.escapeWikiChars(f.Value) + "]]\n"
}

func (f *Fact) escapeWikiChars(inStr string) string {
	outStr := str.Replace(inStr, "[", "(", -1)
	outStr = str.Replace(outStr, "]", ")", -1)
	outStr = str.Replace(outStr, "|", ",", -1)
	outStr = str.Replace(outStr, "=", "-", -1)
	outStr = str.Replace(outStr, "<", "&lt;", -1)
	outStr = str.Replace(outStr, ">", "&gt;", -1)
	return outStr
}

// ------------------------------------------------------------
// Helper type: Category
// ------------------------------------------------------------

type Category struct {
	Name string
}

func NewCategory(name string) *Category {
	return &Category{
		Name: name,
	}
}

func (c *Category) asWikiString() string {
	return "[[Category:" + c.Name + "]]\n"
}
