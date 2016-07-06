RDF2SMW
=======

A tool to convert from RDF triples to Semantic MediaWiki facts (in MediaWiki
XML export format)

RDF2SMW is based on the [FlowBase](https://github.com/flowbase/flowbase)
flow-based programming micro-framework.

**Status:** Under heavy development ... only the fist few components are being fleshed out now, no XML generation yet, etc!

Usage
-----

(Note, you will not get XML output yet, just some intermediate representation!)

```bash
go build
./rdf2smw --infile triples.nt | less -S
```
