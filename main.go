/*
rdf2smw is a commandline tool to convert from RDF data to MediaWiki XML Dump
files, for import using MediaWiki's built in importDump.php script.

Usage

	./rdf2smw -in <infile> -out <outfile>

Flags

	-in  Input file in RDF N-triples format
	-out Output file in (MediaWiki) XML format

Example usage

	./rdf2smw -in mydata.nt -out mydata.xml

For importing the generated XML Dumps into MediaWiki, see this page:
https://www.mediawiki.org/wiki/Manual:Importing_XML_dumps
*/
package main

import (
	"flag"
	"fmt"
	"os"
	str "strings"

	"github.com/flowbase/flowbase"
	"github.com/knakk/rdf"
)

const (
	BUFSIZE = 16
)

func main() {
	//flowbase.InitLogDebug()

	inFileName := flag.String("in", "", "The input file name")
	outFileName := flag.String("out", "", "The output file name")
	flag.Parse()

	doExit := false
	if *inFileName == "" {
		fmt.Println("No filename specified to --in")
		doExit = true
	} else if *outFileName == "" {
		fmt.Println("No filename specified to --out")
		doExit = true
	}

	if doExit {
		os.Exit(1)
	}

	// ------------------------------------------
	// Initialize processes
	// ------------------------------------------

	// Create a pipeline runner
	net := flowbase.NewNet()

	// Read in-file
	ttlFileRead := NewTurtleFileReader()
	net.AddProcess(ttlFileRead)

	// Aggregate per subject
	aggregator := NewAggregateTriplesPerSubject()
	net.AddProcess(aggregator)

	// Create an subject-indexed "index" of all triples
	indexCreator := NewCreateResourceIndex()
	net.AddProcess(indexCreator)

	// Fan-out the triple index to the converter and serializer
	indexFanOut := NewResourceIndexFanOut()
	net.AddProcess(indexFanOut)

	// Serialize the index back to individual subject-tripleaggregates
	indexToAggr := NewResourceIndexToTripleAggregates()
	net.AddProcess(indexToAggr)

	// Convert TripleAggregate to WikiPage
	triplesToWikiConverter := NewTripleAggregateToWikiPageConverter()
	net.AddProcess(triplesToWikiConverter)

	//categoryFilterer := NewCategoryFilterer([]string{"DataEntry"})
	//net.AddProcess(categoryFilterer)

	// Pretty-print wiki page data
	//wikiPagePrinter := NewWikiPagePrinter()
	//net.AddProcess(wikiPagePrinter)

	useTemplates := true
	xmlCreator := NewMWXMLCreator(useTemplates)
	net.AddProcess(xmlCreator)

	//printer := NewStringPrinter()
	//net.AddProcess(printer)
	templateWriter := NewStringFileWriter(str.Replace(*outFileName, ".xml", "_templates.xml", 1))
	net.AddProcess(templateWriter)

	propertyWriter := NewStringFileWriter(str.Replace(*outFileName, ".xml", "_properties.xml", 1))
	net.AddProcess(propertyWriter)

	pageWriter := NewStringFileWriter(*outFileName)
	net.AddProcess(pageWriter)

	snk := flowbase.NewSink()
	net.AddProcess(snk)

	// ------------------------------------------
	// Connect network
	// ------------------------------------------

	ttlFileRead.OutTriple = aggregator.In

	aggregator.Out = indexCreator.In

	indexCreator.Out = indexFanOut.In
	indexFanOut.Out["serialize"] = indexToAggr.In
	indexFanOut.Out["conv"] = triplesToWikiConverter.InIndex

	indexToAggr.Out = triplesToWikiConverter.InAggregate

	//triplesToWikiConverter.OutPage = categoryFilterer.In
	//categoryFilterer.Out = xmlCreator.InWikiPage

	triplesToWikiConverter.OutPage = xmlCreator.InWikiPage

	xmlCreator.OutTemplates = templateWriter.In
	xmlCreator.OutProperties = propertyWriter.In
	xmlCreator.OutPages = pageWriter.In

	snk.Connect(templateWriter.OutDone)
	snk.Connect(propertyWriter.OutDone)
	snk.Connect(pageWriter.OutDone)

	// ------------------------------------------
	// Send in-data and run
	// ------------------------------------------

	go func() {
		defer close(ttlFileRead.InFileName)
		ttlFileRead.InFileName <- *inFileName
	}()

	net.Run()
}

// --------------------------------------------------------------------------------
// IP: TripleAggregate
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
// IP: WikiPage
// --------------------------------------------------------------------------------

type WikiPage struct {
	Title            string
	Type             int
	Facts            []*Fact
	Categories       []string
	SpecificCategory string
}

func NewWikiPage(title string, facts []*Fact, categories []string, specificCategory string, pageType int) *WikiPage {
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

func (p *WikiPage) AddCategory(category string) {
	p.Categories = append(p.Categories, category)
}

func (p *WikiPage) AddCategoryUnique(category string) {
	catExists := false
	for _, existingCat := range p.Categories {
		if category == existingCat {
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

// ------------------------------------------------------------
// Helper functions
// ------------------------------------------------------------

func fmtFact(property string, value string) string {
	return "[[" + property + "::" + value + "]]\n"
}

func fmtCategory(category string) string {
	return "[[Category:" + category + "]]\n"
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func removeLastWord(inStr string) string {
	bits := str.Split(inStr, " ")
	outStr := str.Join(append(bits[:len(bits)-1]), " ")
	return outStr
}

func spacesToUnderscores(inStr string) string {
	return str.Replace(inStr, " ", "_", -1)
}

func upperCaseFirst(inStr string) string {
	var outStr string
	if inStr != "" {
		outStr = str.ToUpper(inStr[0:1]) + inStr[1:]
	}
	return outStr
}

func escapeWikiChars(inStr string) string {
	outStr := str.Replace(inStr, "[", "(", -1)
	outStr = str.Replace(outStr, "]", ")", -1)
	outStr = str.Replace(outStr, "|", ",", -1)
	outStr = str.Replace(outStr, "=", "-", -1)
	outStr = str.Replace(outStr, "<", "&lt;", -1)
	outStr = str.Replace(outStr, ">", "&gt;", -1)
	return outStr
}

func countSuperCategories(tr rdf.Triple, ri *map[string]*TripleAggregate) int {
	catPage := (*ri)[tr.Obj.String()]
	topSuperCatsCnt := 0
	if catPage != nil {
		for _, subTr := range catPage.Triples {
			if subTr.Pred.String() == typePropertyURI || subTr.Pred.String() == subClassPropertyURI {
				superCatsCnt := countSuperCategories(subTr, ri) + 1
				if superCatsCnt > topSuperCatsCnt {
					topSuperCatsCnt = superCatsCnt
				}
			}
		}
	}
	return topSuperCatsCnt
}
