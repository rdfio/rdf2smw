RDF2SMW
=======

A tool to convert from RDF triples to Semantic MediaWiki facts (in MediaWiki
XML export format).

It allows you too import RDF data into a Semantic MediaWiki, via MediaWiki's
robust built-in XML import feature.

RDF2SMW is based on the [FlowBase](https://github.com/flowbase/flowbase)
flow-based programming micro-framework.

RDF2SMW is very similar to the RDF import function in the
[RDFIO](https://github.com/rdfio/RDFIO) Semantic MediaWiki extension, but takes
another approach: Whereas RDFIO converts RDF to wiki pages and imports them in
the same go, RDF2SMW first converts RDF to an XML file outside of PHP (for
better performance), and then importing using MediaWiki's built-in import
function.

**Status:** Basic MediaWiki XML generation now works. Work is being done on
more features and fixing bugs.

For more detailed status, see [TODO.md](https://github.com/samuell/rdf2smw/blob/master/TODO.md)

Installation
------------

For linux 64 bit:

1. Download the file `rdf2smw_linux64.gz`, on the [latest release](https://github.com/samuell/rdf2smw/releases).
2. Unpack it with: `gunzip rdf2smw_linux64.gz`
3. Call it, on the commandline (see the usage section below).


Usage
-----

(Note, you will not get XML output yet, just some intermediate representation!)

```bash
go build
./rdf2smw --infile triples.nt > semantic_mediawiki_pages.xml
php <wikidir>/maintenance/importDump.php semantic_mediawiki_pages.xml
```
