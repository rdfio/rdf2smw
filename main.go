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
	"github.com/rdfio/rdf2smw/components"
	"os"

	str "strings"

	"github.com/flowbase/flowbase"
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
	ttlFileRead := components.NewOsTurtleFileReader()
	net.AddProcess(ttlFileRead)

	// TripleAggregator
	aggregator := components.NewTripleAggregator()
	net.AddProcess(aggregator)

	// Create an subject-indexed "index" of all triples
	indexCreator := components.NewResourceIndexCreator()
	net.AddProcess(indexCreator)

	// Fan-out the triple index to the converter and serializer
	indexFanOut := components.NewResourceIndexFanOut()
	net.AddProcess(indexFanOut)

	// Serialize the index back to individual subject-tripleaggregates
	indexToAggr := components.NewResourceIndexToTripleAggregates()
	net.AddProcess(indexToAggr)

	// Convert TripleAggregate to WikiPage
	triplesToWikiConverter := components.NewTripleAggregateToWikiPageConverter()
	net.AddProcess(triplesToWikiConverter)

	//categoryFilterer := components.NewCategoryFilterer([]string{"DataEntry"})
	//net.AddProcess(categoryFilterer)

	// Pretty-print wiki page data
	//wikiPagePrinter := components.NewWikiPagePrinter()
	//net.AddProcess(wikiPagePrinter)

	useTemplates := true
	xmlCreator := components.NewMWXMLCreator(useTemplates)
	net.AddProcess(xmlCreator)

	//printer := components.NewStringPrinter()
	//net.AddProcess(printer)
	templateWriter := components.NewStringFileWriter(str.Replace(*outFileName, ".xml", "_templates.xml", 1))
	net.AddProcess(templateWriter)

	propertyWriter := components.NewStringFileWriter(str.Replace(*outFileName, ".xml", "_properties.xml", 1))
	net.AddProcess(propertyWriter)

	pageWriter := components.NewStringFileWriter(*outFileName)
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
