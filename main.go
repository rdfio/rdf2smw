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
	"time"

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
	indexCreator := NewCreateResourceIndex()
	pipeRunner.AddProcess(indexCreator)

	// Fan-out the triple index to the converter and serializer
	indexFanOut := NewResourceIndexFanOut()
	pipeRunner.AddProcess(indexFanOut)

	// Serialize the index back to individual subject-tripleaggregates
	indexToAggr := NewResourceIndexToTripleAggregates()
	pipeRunner.AddProcess(indexToAggr)

	// Convert TripleAggregate to WikiPage
	triplesToWikiConverter := NewTripleAggregateToWikiPageConverter()
	pipeRunner.AddProcess(triplesToWikiConverter)

	// Pretty-print wiki page data
	//wikiPagePrinter := NewWikiPagePrinter()
	//pipeRunner.AddProcess(wikiPagePrinter)

	useTemplates := true
	xmlCreator := NewMWXMLCreator(useTemplates)
	pipeRunner.AddProcess(xmlCreator)

	printer := NewStringPrinter()
	pipeRunner.AddProcess(printer)

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

	triplesToWikiConverter.OutPage = xmlCreator.InWikiPage

	xmlCreator.Out = printer.In

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
	resourceIndex := make(map[rdf.Subject][]rdf.Triple)
	for triple := range p.In {
		resourceIndex[triple.Subj] = append(resourceIndex[triple.Subj], triple)
	}
	for subj, triples := range resourceIndex {
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

type CreateResourceIndex struct {
	In  chan *TripleAggregate
	Out chan *map[string]*TripleAggregate
}

func NewCreateResourceIndex() *CreateResourceIndex {
	return &CreateResourceIndex{
		In:  make(chan *TripleAggregate, BUFSIZE),
		Out: make(chan *map[string]*TripleAggregate),
	}
}

func (p *CreateResourceIndex) Run() {
	defer close(p.Out)

	idx := make(map[string]*TripleAggregate)
	for aggr := range p.In {
		idx[aggr.SubjectStr] = aggr
	}

	p.Out <- &idx
}

// --------------------------------------------------------------------------------
// Triple Index FanOut
// --------------------------------------------------------------------------------

type ResourceIndexFanOut struct {
	In  chan *map[string]*TripleAggregate
	Out map[string]chan *map[string]*TripleAggregate
}

func NewResourceIndexFanOut() *ResourceIndexFanOut {
	return &ResourceIndexFanOut{
		In:  make(chan *map[string]*TripleAggregate),
		Out: make(map[string]chan *map[string]*TripleAggregate),
	}
}

func (p *ResourceIndexFanOut) Run() {
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
// Resource Index To Resource Aggregates
// --------------------------------------------------------------------------------

type ResourceIndexToTripleAggregates struct {
	In  chan *map[string]*TripleAggregate
	Out chan *TripleAggregate
}

func NewResourceIndexToTripleAggregates() *ResourceIndexToTripleAggregates {
	return &ResourceIndexToTripleAggregates{
		In:  make(chan *map[string]*TripleAggregate, BUFSIZE),
		Out: make(chan *TripleAggregate, BUFSIZE),
	}
}

func (p *ResourceIndexToTripleAggregates) Run() {
	defer close(p.Out)

	for idx := range p.In {
		for _, aggr := range *idx {
			p.Out <- aggr
		}
	}
}

// -----------------------------------------------------------------------------
// TripleAggregateToWikiPageConverter
// -----------------------------------------------------------------------------

// Constants etc ---------------------------------------------------------------

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
	_ = iota
	URITypeUndefined
	URITypePredicate
	URITypeClass
)

// Code -----------------------------------------------------------------------

type TripleAggregateToWikiPageConverter struct {
	InAggregate chan *TripleAggregate
	InIndex     chan *map[string]*TripleAggregate
	OutPage     chan *WikiPage
}

func NewTripleAggregateToWikiPageConverter() *TripleAggregateToWikiPageConverter {
	return &TripleAggregateToWikiPageConverter{
		InAggregate: make(chan *TripleAggregate, BUFSIZE),
		InIndex:     make(chan *map[string]*TripleAggregate, BUFSIZE),
		OutPage:     make(chan *WikiPage, BUFSIZE),
	}
}

func (p *TripleAggregateToWikiPageConverter) Run() {
	defer close(p.OutPage)
	resourceIndex := <-p.InIndex
	for aggr := range p.InAggregate {
		pageType := p.determineType(aggr)

		pageTitle, _ := p.convertUriToWikiTitle(aggr.SubjectStr, pageType, resourceIndex)

		page := NewWikiPage(pageTitle, []*Fact{}, []string{}, pageType)

		for _, tr := range aggr.Triples {

			_, propertyStr := p.convertUriToWikiTitle(tr.Pred.String(), URITypePredicate, resourceIndex) // Here we know it is a predicate, simply because its location in a triple

			valueAggr := (*resourceIndex)[tr.Obj.String()]
			valueUriType := p.determineType(valueAggr)
			_, valueStr := p.convertUriToWikiTitle(tr.Obj.String(), valueUriType, resourceIndex)

			if valueUriType == URITypeClass && (tr.Pred.String() == typePropertyURI || tr.Pred.String() == subClassPropertyURI) {
				page.AddCategory(valueStr)
			} else {
				fact := NewFact(propertyStr, valueStr)
				page.AddFact(fact)
			}
		}

		// Add Equivalent URI fact
		equivURIFact := NewFact("Equivalent URI", aggr.Subject.String())
		page.AddFact(equivURIFact)

		p.OutPage <- page
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
	for _, titleProp := range titleProperties {
		if aggr != nil {
			for _, tr := range aggr.Triples {
				if tr.Pred.String() == titleProp {
					factTitle = tr.Obj.String()
				}
			}
		} else {
			factTitle = ""
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

	// Clean up strange characters
	factTitle = str.Replace(factTitle, "[", "(", -1)
	factTitle = str.Replace(factTitle, "]", ")", -1)

	if uriType == URITypePredicate {
		pageTitle = "Property:" + factTitle
	} else if uriType == URITypeClass {
		pageTitle = "Category:" + factTitle
	} else {
		pageTitle = factTitle
	}

	return pageTitle, factTitle
}

// --------------------------------------------------------------------------------
// MW XML Creator
// --------------------------------------------------------------------------------

type MWXMLCreator struct {
	InWikiPage   chan *WikiPage
	Out          chan string
	UseTemplates bool
}

func NewMWXMLCreator(useTemplates bool) *MWXMLCreator {
	return &MWXMLCreator{
		InWikiPage:   make(chan *WikiPage, BUFSIZE),
		Out:          make(chan string, BUFSIZE),
		UseTemplates: useTemplates,
	}
}

const wikiXmlTpl = `
	<page>
		<title>%s</title>
		<ns>%d</ns>
		<revision>
			<timestamp>%s</timestamp>
			<contributor>
				<ip>127.0.0.1</ip>
			</contributor>
			<comment>Page created by RDF2SMW commandline tool</comment>
			<model>wikitext</model>
			<format>text/x-wiki</format>
			<text xml:space="preserve">
%s</text>
		</revision>
	</page>
`

var pageTypeToMWNamespace = map[int]int{
	URITypeClass:     14,
	URITypePredicate: 102,
	URITypeUndefined: 0,
}

func (p *MWXMLCreator) Run() {
	defer close(p.Out)

	p.Out <- "<mediawiki>\n"

	for page := range p.InWikiPage {

		wikiText := ""

		if p.UseTemplates && len(page.Categories) > 0 { // We need at least one category, as to name the (to-be) template

			wikiText += "{{" + page.Categories[0] + "\n" // TODO: What to do when we have multipel categories?

			// Add facts as parameters to the template
			for _, fact := range page.Facts {
				wikiText += "|" + str.Replace(fact.Property, " ", "_", -1) + "=" + fact.Value + "\n"
			}

			// Add categories as multi-valued call to the "categories" value of the template
			wikiText += "|categories="
			for i, cat := range page.Categories {
				if i == 0 {
					wikiText += cat
				} else {
					wikiText += "," + cat
				}
			}

			wikiText += "\n}}"
		} else {

			// Add fact statements
			for _, fact := range page.Facts {
				wikiText += fmtFact(fact.Property, fact.Value)
			}

			// Add category statements
			for _, cat := range page.Categories {
				wikiText += fmtCategory(cat)
			}

		}

		xmlData := fmt.Sprintf(wikiXmlTpl, page.Title, pageTypeToMWNamespace[page.Type], time.Now().Format("2006-01-02T15:04:05Z"), wikiText)

		// Print out the generated XML one line at a time
		p.Out <- xmlData
	}

	p.Out <- "</mediawiki>\n"
}

// --------------------------------------------------------------------------------
// SMWTemplateCallFormatter
// --------------------------------------------------------------------------------

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

// --------------------------------------------------------------------------------
// String Printer
// --------------------------------------------------------------------------------

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
	Title      string
	Type       int
	Facts      []*Fact
	Categories []string
}

func NewWikiPage(title string, facts []*Fact, categories []string, pageType int) *WikiPage {
	return &WikiPage{
		Title:      title,
		Facts:      facts,
		Categories: categories,
		Type:       pageType,
	}
}

func (p *WikiPage) AddFact(fact *Fact) {
	p.Facts = append(p.Facts, fact)
}

func (p *WikiPage) AddCategory(category string) {
	p.Categories = append(p.Categories, category)
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

// Helper functions

func fmtFact(property string, value string) string {
	return "[[" + property + "::" + value + "]]\n"
}

func fmtCategory(category string) string {
	return "[[Category:" + category + "]]\n"
}
