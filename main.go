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
	"io"
	"log"
	"os"
	"regexp"
	str "strings"
	"time"

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

// ================================================================================
// Components
// ================================================================================

// --------------------------------------------------------------------------------
// TurtleFileReader
// --------------------------------------------------------------------------------

type TurtleFileReader struct {
	InFileName chan string
	OutTriple  chan rdf.Triple
}

func NewTurtleFileReader() *TurtleFileReader {
	return &TurtleFileReader{
		InFileName: make(chan string, BUFSIZE),
		OutTriple:  make(chan rdf.Triple, BUFSIZE),
	}
}

func (p *TurtleFileReader) Run() {
	defer close(p.OutTriple)

	flowbase.Debug.Println("Starting loop")
	for fileName := range p.InFileName {
		flowbase.Debug.Printf("Starting processing file %s\n", fileName)
		fh, err := os.Open(fileName)
		if err != nil {
			log.Fatal(err)
		}
		defer fh.Close()

		dec := rdf.NewTripleDecoder(fh, rdf.Turtle)
		for triple, err := dec.Decode(); err != io.EOF; triple, err = dec.Decode() {
			if err != nil {
				log.Fatal("Could not encode to triple: ", err.Error())
			} else if triple.Subj != nil && triple.Pred != nil && triple.Obj != nil {
				p.OutTriple <- triple
			} else {
				log.Fatal("Something was encoded as nil in the triple:", triple)
			}
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
			if err != nil {
				log.Fatal("Could not encode to triple: ", err.Error())
			} else if triple.Subj != nil && triple.Pred != nil && triple.Obj != nil {
				p.Out <- triple
			} else {
				log.Fatal("Something was encoded as nil in the triple:", triple)
			}
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

// --------------------------------------------------------------------------------
// CategoryFilterer
// --------------------------------------------------------------------------------

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

// --------------------------------------------------------------------------------
// MW XML Creator
// --------------------------------------------------------------------------------

type MWXMLCreator struct {
	InWikiPage    chan *WikiPage
	OutTemplates  chan string
	OutProperties chan string
	OutPages      chan string
	UseTemplates  bool
}

func NewMWXMLCreator(useTemplates bool) *MWXMLCreator {
	return &MWXMLCreator{
		InWikiPage:    make(chan *WikiPage, BUFSIZE),
		OutTemplates:  make(chan string, BUFSIZE),
		OutProperties: make(chan string, BUFSIZE),
		OutPages:      make(chan string, BUFSIZE),
		UseTemplates:  useTemplates,
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
	URITypeTemplate:  10,
	URITypePredicate: 102,
	URITypeUndefined: 0,
}

func (p *MWXMLCreator) Run() {
	tplPropertyIdx := make(map[string]map[string]int)

	defer close(p.OutTemplates)
	defer close(p.OutProperties)
	defer close(p.OutPages)

	p.OutPages <- "<mediawiki>\n"
	p.OutProperties <- "<mediawiki>\n"

	for page := range p.InWikiPage {

		wikiText := ""

		if p.UseTemplates && len(page.Categories) > 0 { // We need at least one category, as to name the (to-be) template

			var templateName string
			if page.SpecificCategory != "" {
				templateName = page.SpecificCategory
			} else {
				// Pick last item (biggest chance to be pretty specific?)
				templateName = page.Categories[len(page.Categories)-1]
				//println("Page ", page.Title, " | Didn't have a specific catogory, so selected ", templateName)
			}
			templateTitle := "Template:" + templateName

			// Make sure template page exists
			if tplPropertyIdx[templateTitle] == nil {
				tplPropertyIdx[templateTitle] = make(map[string]int)
			}

			wikiText += "{{" + templateName + "\n" // TODO: What to do when we have multipel categories?

			// Add facts as parameters to the template call
			var lastProperty string
			for _, fact := range page.Facts {
				// Write facts to template call on current page

				val := escapeWikiChars(fact.Value)
				if fact.Property == lastProperty {
					wikiText += "," + val + "\n"
				} else {
					wikiText += "|" + spacesToUnderscores(fact.Property) + "=" + val + "\n"
				}

				lastProperty = fact.Property

				// Add fact to the relevant template page
				tplPropertyIdx[templateTitle][fact.Property] = 1
			}

			// Add categories as multi-valued call to the "categories" value of the template
			wikiText += "|Categories="
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
				wikiText += fmtFact(fact.Property, escapeWikiChars(fact.Value))
			}

			// Add category statements
			for _, cat := range page.Categories {
				wikiText += fmtCategory(cat)
			}

		}

		xmlData := fmt.Sprintf(wikiXmlTpl, page.Title, pageTypeToMWNamespace[page.Type], time.Now().Format("2006-01-02T15:04:05Z"), wikiText)

		// Print out the generated XML one line at a time
		if page.Type == URITypePredicate {
			p.OutProperties <- xmlData
		} else {
			p.OutPages <- xmlData
		}
	}
	p.OutPages <- "</mediawiki>\n"
	p.OutProperties <- "</mediawiki>\n"

	p.OutTemplates <- "<mediawiki>\n"
	// Create template pages
	for tplName, tplProperties := range tplPropertyIdx {
		tplText := `{|class="wikitable smwtable"
!colspan="2"| ` + str.Replace(tplName, "Template:", "", -1) + `: {{PAGENAMEE}}
`
		for property := range tplProperties {
			argName := spacesToUnderscores(property)
			tplText += fmt.Sprintf("|-\n!%s\n|{{#arraymap:{{{%s|}}}|,|x|[[%s::x]]|,}}\n", property, argName, property)
		}
		tplText += "|}\n\n"
		// Add categories
		tplText += "{{#arraymap:{{{Categories}}}|,|x|[[Category:x]]|}}\n"

		xmlData := fmt.Sprintf(wikiXmlTpl, tplName, pageTypeToMWNamespace[URITypeTemplate], time.Now().Format("2006-01-02T15:04:05Z"), tplText)
		p.OutTemplates <- xmlData
	}
	p.OutTemplates <- "</mediawiki>\n"
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
// String File Writer
// --------------------------------------------------------------------------------

type StringFileWriter struct {
	In       chan string
	OutDone  chan interface{}
	fileName string
}

func NewStringFileWriter(fileName string) *StringFileWriter {
	return &StringFileWriter{
		In:       make(chan string, BUFSIZE),
		OutDone:  make(chan interface{}, BUFSIZE),
		fileName: fileName,
	}
}

func (p *StringFileWriter) Run() {
	defer close(p.OutDone)

	fh, err := os.Create(p.fileName)
	if err != nil {
		panic("Could not create output file: " + err.Error())
	}
	defer fh.Close()
	for s := range p.In {
		fh.WriteString(s)
	}

	flowbase.Debug.Printf("Sending done signal on chan %v now in StringFileWriter ...\n", p.OutDone)
	p.OutDone <- &DoneSignal{}
}

type DoneSignal struct{}

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
