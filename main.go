// Workflow written in SciPipe.
// For more information about SciPipe, see: http://scipipe.org
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	str "strings"

	"github.com/flowbase/flowbase"
	"github.com/knakk/rdf"
)

const (
	BUFSIZE = 16
)

func main() {
	flowbase.InitLogInfo()

	inFileName := flag.String("infile", "", "The input file name")
	flag.Parse()
	if *inFileName == "" {
		fmt.Println("No filename specified to --infile")
		os.Exit(1)
	}

	// ------------------------------------------
	// Initialize processes
	// ------------------------------------------

	// Create a pipeline runner
	pipeRunner := flowbase.NewPipelineRunner()

	// Read in-file
	fileRead := NewFileReader()
	pipeRunner.AddProcess(fileRead)

	// Parse triples
	parser := NewTripleParser()
	pipeRunner.AddProcess(parser)

	// Aggregate per subject
	aggregator := NewAggregateTriplesPerSubject()
	pipeRunner.AddProcess(aggregator)

	// Create an subject-indexed "index" of all triples
	indexCreator := NewCreateTripleIndex()
	pipeRunner.AddProcess(indexCreator)

	// Fan-out the triple index to the converter and serializer
	indexFanOut := NewTripleIndexFanOut()
	pipeRunner.AddProcess(indexFanOut)

	// Serialize the index back to individual subject-tripleaggregates
	indexToAggr := NewTripleIndexToTripleAggregates()
	pipeRunner.AddProcess(indexToAggr)

	// Convert TripleAggregate to WikiPage
	triplesToWikiConverter := NewTripleAggregateToWikiPageConverter()
	pipeRunner.AddProcess(triplesToWikiConverter)

	// Pretty-print wiki page data
	wikiPagePrinter := NewWikiPagePrinter()
	pipeRunner.AddProcess(wikiPagePrinter)

	// ------------------------------------------
	// Connect network
	// ------------------------------------------

	fileRead.OutLine = parser.In
	parser.Out = aggregator.In

	aggregator.Out = indexCreator.In

	indexCreator.Out = indexFanOut.In
	indexFanOut.Out["serialize"] = indexToAggr.In
	indexFanOut.Out["conv"] = triplesToWikiConverter.InIndex

	indexToAggr.Out = triplesToWikiConverter.InAggregate

	triplesToWikiConverter.OutPage = wikiPagePrinter.In

	// ------------------------------------------
	// Send in-data and run
	// ------------------------------------------

	go func() {
		defer close(fileRead.InFileName)
		fileRead.InFileName <- *inFileName
	}()

	pipeRunner.Run()

}

// ================================================================================
// Components
// ================================================================================

// --------------------------------------------------------------------------------
// FileReader
// --------------------------------------------------------------------------------

type FileReader struct {
	InFileName chan string
	OutLine    chan string
}

func NewFileReader() *FileReader {
	return &FileReader{
		InFileName: make(chan string, BUFSIZE),
		OutLine:    make(chan string, BUFSIZE),
	}
}

func (p *FileReader) Run() {
	defer close(p.OutLine)

	flowbase.Debug.Println("Starting loop")
	for fileName := range p.InFileName {
		flowbase.Debug.Printf("Starting processing file %s\n", fileName)
		fh, err := os.Open(fileName)
		if err != nil {
			log.Fatal(err)
		}
		defer fh.Close()

		sc := bufio.NewScanner(fh)
		for sc.Scan() {
			if err := sc.Err(); err != nil {
				log.Fatal(err)
			}
			p.OutLine <- sc.Text()
		}
	}
}

// --------------------------------------------------------------------------------
// TripleParser
// --------------------------------------------------------------------------------

type TripleParser struct {
	In  chan string
	Out chan rdf.Triple
}

func NewTripleParser() *TripleParser {
	return &TripleParser{
		In:  make(chan string, BUFSIZE),
		Out: make(chan rdf.Triple, BUFSIZE),
	}
}

func (p *TripleParser) Run() {
	defer close(p.Out)
	for line := range p.In {
		lineReader := str.NewReader(line)
		dec := rdf.NewTripleDecoder(lineReader, rdf.Turtle)
		for triple, err := dec.Decode(); err != io.EOF; triple, err = dec.Decode() {
			p.Out <- triple
		}
	}
}

// --------------------------------------------------------------------------------
// AggregateTriplesPerSubject
// --------------------------------------------------------------------------------

type AggregateTriplesPerSubject struct {
	In  chan rdf.Triple
	Out chan *TripleAggregate
}

func NewAggregateTriplesPerSubject() *AggregateTriplesPerSubject {
	return &AggregateTriplesPerSubject{
		In:  make(chan rdf.Triple, BUFSIZE),
		Out: make(chan *TripleAggregate, BUFSIZE),
	}
}

func (p *AggregateTriplesPerSubject) Run() {
	defer close(p.Out)
	tripleIndex := make(map[rdf.Subject][]rdf.Triple)
	for triple := range p.In {
		tripleIndex[triple.Subj] = append(tripleIndex[triple.Subj], triple)
	}
	for subj, triples := range tripleIndex {
		tripleAggregate := NewTripleAggregate(subj, triples)
		p.Out <- tripleAggregate
	}
}

type FanOutTripleAggregate struct {
	In  chan *TripleAggregate
	Out map[string](chan *TripleAggregate)
}

// NewFanOut creates a new FanOut process
func NewFanOutTripleAggregate() *FanOutTripleAggregate {
	return &FanOutTripleAggregate{
		In:  make(chan *TripleAggregate, BUFSIZE),
		Out: make(map[string](chan *TripleAggregate)),
	}
}

// Run runs the FanOut process
func (proc *FanOutTripleAggregate) Run() {
	for _, outPort := range proc.Out {
		defer close(outPort)
	}

	for ft := range proc.In {
		for _, outPort := range proc.Out {
			outPort <- ft
		}
	}
}

// --------------------------------------------------------------------------------
// Create Triple Index
// --------------------------------------------------------------------------------

type CreateTripleIndex struct {
	In  chan *TripleAggregate
	Out chan map[string]*TripleAggregate
}

func NewCreateTripleIndex() *CreateTripleIndex {
	return &CreateTripleIndex{
		In:  make(chan *TripleAggregate, BUFSIZE),
		Out: make(chan map[string]*TripleAggregate),
	}
}

func (p *CreateTripleIndex) Run() {
	defer close(p.Out)

	idx := make(map[string]*TripleAggregate)
	for aggr := range p.In {
		idx[aggr.SubjectStr] = aggr
	}

	p.Out <- idx
}

// --------------------------------------------------------------------------------
// Triple Index FanOut
// --------------------------------------------------------------------------------

type TripleIndexFanOut struct {
	In  chan map[string]*TripleAggregate
	Out map[string]chan map[string]*TripleAggregate
}

func NewTripleIndexFanOut() *TripleIndexFanOut {
	return &TripleIndexFanOut{
		In:  make(chan map[string]*TripleAggregate),
		Out: make(map[string]chan map[string]*TripleAggregate),
	}
}

func (p *TripleIndexFanOut) Run() {
	for _, outPort := range p.Out {
		defer close(outPort)
	}

	for idx := range p.In {
		for _, outPort := range p.Out {
			outPort <- idx
		}
	}
}

// --------------------------------------------------------------------------------
// Triple Index To Triple Aggregates
// --------------------------------------------------------------------------------

type TripleIndexToTripleAggregates struct {
	In  chan map[string]*TripleAggregate
	Out chan *TripleAggregate
}

func NewTripleIndexToTripleAggregates() *TripleIndexToTripleAggregates {
	return &TripleIndexToTripleAggregates{
		In:  make(chan map[string]*TripleAggregate, BUFSIZE),
		Out: make(chan *TripleAggregate, BUFSIZE),
	}
}

func (p *TripleIndexToTripleAggregates) Run() {
	defer close(p.Out)

	for idx := range p.In {
		for _, aggr := range idx {
			p.Out <- aggr
		}
	}
}

// --------------------------------------------------------------------------------
// TripleAggregateToWikiPageConverter
// --------------------------------------------------------------------------------

var titleProperties = []string{
	"http://semantic-mediawiki.org/swivt/1.0#page",
	"http://www.w3.org/2000/01/rdf-schema#label",
	"http://purl.org/dc/elements/1.1/title",
	"http://www.w3.org/2004/02/skos/core#preferredLabel",
	"http://xmlns.com/foaf/0.1/name",
}

var namespaceAbbreviations = map[string]string{
	"http://www.opentox.org/api/1.1#": "opentox",
}

type TripleAggregateToWikiPageConverter struct {
	InAggregate chan *TripleAggregate
	InIndex     chan map[string]*TripleAggregate
	OutPage     chan *WikiPage
}

func NewTripleAggregateToWikiPageConverter() *TripleAggregateToWikiPageConverter {
	return &TripleAggregateToWikiPageConverter{
		InAggregate: make(chan *TripleAggregate, BUFSIZE),
		InIndex:     make(chan map[string]*TripleAggregate, BUFSIZE),
		OutPage:     make(chan *WikiPage, BUFSIZE),
	}
}

func (p *TripleAggregateToWikiPageConverter) Run() {
	defer close(p.OutPage)
	tripleIndex := <-p.InIndex
	for aggr := range p.InAggregate {
		pageTitle, _ := p.convertUriToWikiTitle(aggr.SubjectStr, false, tripleIndex)

		page := NewWikiPage(pageTitle, []*Fact{})
		for _, tr := range aggr.Triples {
			fact := NewFact(tr.Pred.String(), tr.Obj.String())
			page.AddFact(fact)
		}
		p.OutPage <- page
	}
}

// For properties, the factTitle and pageTitle will be different (The page
// title including the "Property:" prefix), while for normal pages, they will
// be the same.
func (p *TripleAggregateToWikiPageConverter) convertUriToWikiTitle(uri string,
	isProperty bool, tripleIndex map[string]*TripleAggregate) (pageTitle string, factTitle string) {

	aggr := tripleIndex[uri]

	// Conversion strategies:
	// 1. Existing wiki title (in wiki, or cache)
	// 2. Use configured title-deciding properties
	for _, titleProp := range titleProperties {
		for _, tr := range aggr.Triples {
			if tr.Pred.String() == titleProp {
				factTitle = tr.Obj.String()
			}
		}
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

	if isProperty {
		pageTitle = "Property:" + factTitle
	} else {
		pageTitle = factTitle
	}

	return pageTitle, factTitle
}

// --------------------------------------------------------------------------------
// TriplePrinter
// --------------------------------------------------------------------------------

type TriplePrinter struct {
	In chan rdf.Triple
}

func NewTriplePrinter() *TriplePrinter {
	return &TriplePrinter{
		In: make(chan rdf.Triple, BUFSIZE),
	}
}

func (p *TriplePrinter) Run() {
	for tr := range p.In {
		fmt.Printf("S: %s\nP: %s\nO: %s\n\n", tr.Subj.String(), tr.Pred.String(), tr.Obj.String())
	}
}

// --------------------------------------------------------------------------------
// TripleAggregatePrinter
// --------------------------------------------------------------------------------

type TripleAggregatePrinter struct {
	In chan *TripleAggregate
}

func NewTripleAggregatePrinter() *TripleAggregatePrinter {
	return &TripleAggregatePrinter{
		In: make(chan *TripleAggregate, BUFSIZE),
	}
}

func (p *TripleAggregatePrinter) Run() {
	for trAggr := range p.In {
		fmt.Printf("Subject: %s\nTriples:\n", trAggr.Subject)
		for _, tr := range trAggr.Triples {
			fmt.Printf("\t<%s> <%s> <%s>\n", tr.Subj.String(), tr.Pred.String(), tr.Obj.String())
		}
		fmt.Println()
	}
}

// --------------------------------------------------------------------------------
// WikiPagePrinter
// --------------------------------------------------------------------------------

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
		fmt.Println("Title: ", page.Title)
		for _, fact := range page.Facts {
			fmt.Printf("[[%s::%s]]\n", fact.Property, fact.Value)
		}
		fmt.Println("") // Print an empty line
	}
}

// --------------------------------------------------------------------------------
// IP: RDFTriple
// --------------------------------------------------------------------------------
//
//type RDFTriple struct {
//	Subject   string
//	Predicate string
//	Object    string
//}
//
//func NewRDFTriple() *RDFTriple {
//	return &RDFTriple{}
//}

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
	Title string
	Facts []*Fact
}

func NewWikiPage(title string, facts []*Fact) *WikiPage {
	return &WikiPage{
		Title: title,
		Facts: facts,
	}
}

func (p *WikiPage) AddFact(fact *Fact) {
	p.Facts = append(p.Facts, fact)
}

// Helper type: Fact

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
