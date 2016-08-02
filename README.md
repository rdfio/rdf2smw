RDF2SMW
=======

A tool to convert from RDF triples to Semantic MediaWiki facts (in MediaWiki
XML export format)

RDF2SMW is based on the [FlowBase](https://github.com/flowbase/flowbase)
flow-based programming micro-framework.

**Status:** Basic MediaWiki import XML now works. Work is being done on more features and fixing bugs.

For more detailed status, see [TODO.md](https://github.com/samuell/rdf2smw/blob/master/TODO.md)

Usage
-----

(Note, you will not get XML output yet, just some intermediate representation!)

```bash
go build
./rdf2smw --infile triples.nt > semantic_mediawiki_pages.xml
php <wikidir>/maintenance/importDump.php semantic_mediawiki_pages.xml
```
