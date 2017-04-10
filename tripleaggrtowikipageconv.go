package main

import (
	"regexp"
	str "strings"

	"github.com/knakk/rdf"
)

// Constants etc ---------------------------------------------------------------

var titleProperties = []string{
	"http://semantic-mediawiki.org/swivt/1.0#page",
	"http://www.w3.org/2000/01/rdf-schema#label",
	"http://purl.org/dc/elements/1.1/title",
	"http://purl.org/dc/terms/title",
	"http://www.w3.org/2004/02/skos/core#preferredLabel",
	"http://xmlns.com/foaf/0.1/name",
}

var namespaceAbbreviations = map[string]string{
	"http://www.opentox.org/api/1.1#": "opentox",
}

var propertyTypes = []string{
	"http://www.w3.org/2002/07/owl#AnnotationProperty",
	"http://www.w3.org/2002/07/owl#DatatypeProperty",
	"http://www.w3.org/2002/07/owl#ObjectProperty",
}

var categoryTypes = []string{
	"http://www.w3.org/2002/07/owl#Class",
}

const (
	typePropertyURI     = "http://www.w3.org/1999/02/22-rdf-syntax-ns#type"
	subClassPropertyURI = "http://www.w3.org/2000/01/rdf-schema#subClassOf"
)

const (
	dataTypeURIString     = "http://www.w3.org/2001/XMLSchema#string"
	dataTypeURILangString = "http://www.w3.org/1999/02/22-rdf-syntax-ns#langString"
	dataTypeURIInteger    = "http://www.w3.org/2001/XMLSchema#integer"
	dataTypeURIFloat      = "http://www.w3.org/2001/XMLSchema#float"
)

const (
	_ = iota
	URITypeUndefined
	URITypePredicate
	URITypeClass
	URITypeTemplate
)

// Code -----------------------------------------------------------------------

type TripleAggregateToWikiPageConverter struct {
	InAggregate    chan *TripleAggregate
	InIndex        chan *map[string]*TripleAggregate
	OutPage        chan *WikiPage
	cleanUpRegexes []*regexp.Regexp
}

func NewTripleAggregateToWikiPageConverter() *TripleAggregateToWikiPageConverter {
	return &TripleAggregateToWikiPageConverter{
		InAggregate: make(chan *TripleAggregate, BUFSIZE),
		InIndex:     make(chan *map[string]*TripleAggregate, BUFSIZE),
		OutPage:     make(chan *WikiPage, BUFSIZE),
		cleanUpRegexes: []*regexp.Regexp{
			regexp.MustCompile(" [(][^)]*:[^)]*[)]"),
			regexp.MustCompile(" [[][^]]*:[^]]*[]]"),
		},
	}
}

func (p *TripleAggregateToWikiPageConverter) Run() {
	defer close(p.OutPage)

	predPageIndex := make(map[string]*WikiPage)

	resourceIndex := <-p.InIndex

	for aggr := range p.InAggregate {
		pageType := p.determineType(aggr)

		pageTitle, _ := p.convertUriToWikiTitle(aggr.SubjectStr, pageType, resourceIndex)

		page := NewWikiPage(pageTitle, []*Fact{}, []string{}, "", pageType)

		topSuperCatsCnt := 0
		for _, tr := range aggr.Triples {

			predTitle, propertyStr := p.convertUriToWikiTitle(tr.Pred.String(), URITypePredicate, resourceIndex) // Here we know it is a predicate, simply because its location in a triple

			// Make sure property page exists
			if predPageIndex[predTitle] == nil {
				predPageIndex[predTitle] = NewWikiPage(predTitle, []*Fact{}, []string{}, "", URITypePredicate)
			}

			var valueStr string

			if tr.Obj.Type() == rdf.TermIRI {

				valueAggr := (*resourceIndex)[tr.Obj.String()]
				valueUriType := p.determineType(valueAggr)
				_, valueStr = p.convertUriToWikiTitle(tr.Obj.String(), valueUriType, resourceIndex)

				predPageIndex[predTitle].AddFactUnique(NewFact("Has type", "Page"))

			} else if tr.Obj.Type() == rdf.TermLiteral {

				valueStr = tr.Obj.String()

				for _, r := range p.cleanUpRegexes {
					valueStr = r.ReplaceAllString(valueStr, "")
				}

				dataTypeStr := tr.Obj.(rdf.Literal).DataType.String()

				// Add type info on the current property's page
				switch dataTypeStr {
				case dataTypeURIString:
					predPageIndex[predTitle].AddFactUnique(NewFact("Has type", "Text"))
				case dataTypeURILangString:
					predPageIndex[predTitle].AddFactUnique(NewFact("Has type", "Text"))
				case dataTypeURIInteger:
					predPageIndex[predTitle].AddFactUnique(NewFact("Has type", "Number"))
				case dataTypeURIFloat:
					predPageIndex[predTitle].AddFactUnique(NewFact("Has type", "Number"))
				}
			}

			if tr.Pred.String() == typePropertyURI || tr.Pred.String() == subClassPropertyURI {
				page.AddCategoryUnique(valueStr)
				superCatsCnt := countSuperCategories(tr, resourceIndex)
				if superCatsCnt > topSuperCatsCnt {
					topSuperCatsCnt = superCatsCnt
					page.SpecificCategory = valueStr
					//println("Page:", page.Title, " | Adding cat", valueStr, "since has", superCatsCnt, "super categories.")
				}
			} else {
				page.AddFactUnique(NewFact(propertyStr, valueStr))
			}
		}

		// Add Equivalent URI fact
		equivURIFact := NewFact("Equivalent URI", aggr.Subject.String())
		page.AddFactUnique(equivURIFact)

		// Don't send predicates just yet (we want to gather facts about them,
		// and send at the end) ...
		if pageType == URITypePredicate {
			if predPageIndex[page.Title] != nil {
				// Add facts and categories to existing page
				for _, fact := range page.Facts {
					predPageIndex[page.Title].AddFactUnique(fact)
				}
				for _, cat := range page.Categories {
					predPageIndex[page.Title].AddCategoryUnique(cat)
				}
			} else {
				// If page does not exist, use the newly created one
				predPageIndex[page.Title] = page
			}
		} else {
			p.OutPage <- page
		}
	}

	for _, predPage := range predPageIndex {
		p.OutPage <- predPage
	}
}

func (p *TripleAggregateToWikiPageConverter) determineType(uriAggr *TripleAggregate) int {
	if uriAggr != nil {
		if uriAggr.Triples != nil {
			for _, tr := range uriAggr.Triples {
				for _, propType := range propertyTypes {
					if tr.Pred.String() == typePropertyURI && tr.Obj.String() == propType {
						return URITypePredicate
					}
				}
				for _, catType := range categoryTypes {
					if tr.Pred.String() == typePropertyURI && tr.Obj.String() == catType {
						return URITypeClass
					}
				}
			}
		}
	}
	return URITypeUndefined
}

// For properties, the factTitle and pageTitle will be different (The page
// title including the "Property:" prefix), while for normal pages, they will
// be the same.
func (p *TripleAggregateToWikiPageConverter) convertUriToWikiTitle(uri string, uriType int, resourceIndex *map[string]*TripleAggregate) (pageTitle string, factTitle string) {

	aggr := (*resourceIndex)[uri]

	// Conversion strategies:
	// 1. Existing wiki title (in wiki, or cache)
	// 2. Use configured title-deciding properties
	if aggr != nil {
		factTitle = p.findTitleInTriples(aggr.Triples)
	}

	// 3. Shorten URI namespace to alias (e.g. http://purl.org/dc -> dc:)
	//    (Does this apply for properties only?)

	// 4. Remove namespace, keep only local part of URL (Split on '/' or '#')
	if factTitle == "" {
		bits := str.Split(uri, "#")
		lastBit := bits[len(bits)-1]
		bits = str.Split(lastBit, "/")
		lastBit = bits[len(bits)-1]
		factTitle = lastBit
	}

	// Clean up strange characters
	factTitle = str.Replace(factTitle, "[", "(", -1)
	factTitle = str.Replace(factTitle, "]", ")", -1)
	factTitle = str.Replace(factTitle, "{", "(", -1)
	factTitle = str.Replace(factTitle, "}", ")", -1)
	factTitle = str.Replace(factTitle, "|", " ", -1)
	factTitle = str.Replace(factTitle, "#", " ", -1)
	factTitle = str.Replace(factTitle, "<", "less than", -1)
	factTitle = str.Replace(factTitle, ">", "greater than", -1)
	factTitle = str.Replace(factTitle, "?", " ", -1)
	factTitle = str.Replace(factTitle, "&", " ", -1)
	factTitle = str.Replace(factTitle, ",", " ", -1) // Can't allow comma's as we use it as a separator in template variables
	factTitle = str.Replace(factTitle, ".", " ", -1)
	factTitle = str.Replace(factTitle, "=", "-", -1)

	// Clean up according to regexes
	for _, r := range p.cleanUpRegexes {
		factTitle = r.ReplaceAllString(factTitle, "")
	}

	// Limit to max 255 chars (due to MediaWiki limitation)
	titleIsShortened := false
	for len(factTitle) >= 250 {
		factTitle = removeLastWord(factTitle)
		titleIsShortened = true
	}

	if titleIsShortened {
		factTitle += " ..."
	}

	factTitle = upperCaseFirst(factTitle)

	if uriType == URITypePredicate {
		pageTitle = "Property:" + factTitle
	} else if uriType == URITypeClass {
		pageTitle = "Category:" + factTitle
	} else {
		pageTitle = factTitle
	}

	return pageTitle, factTitle
}

func (p *TripleAggregateToWikiPageConverter) findTitleInTriples(triples []rdf.Triple) string {
	for _, titleProp := range titleProperties {
		for _, tr := range triples {
			if tr.Pred.String() == titleProp {
				return tr.Obj.String()
			}
		}
	}
	return ""
}
