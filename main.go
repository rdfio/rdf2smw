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
	ttlFileRead := NewOsTurtleFileReader()
	net.AddProcess(ttlFileRead)

	// TripleAggregator
	aggregator := NewTripleAggregator()
	net.AddProcess(aggregator)

	// Create an subject-indexed "index" of all triples
	indexCreator := NewResourceIndexCreator()
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
