RDF2SMW
=======

A (commandline) tool to convert from RDF triples to Semantic MediaWiki facts
(in MediaWiki XML export format).

It allows you too import RDF data into a [Semantic MediaWiki](http://semantic-mediawiki.org), via [MediaWiki](https://www.mediawiki.org)'s
robust built-in [XML import feature](https://www.mediawiki.org/wiki/Manual:Importing_XML_dumps).

It is written in Go for better performance (than PHP). The latest version
processes triples into pages in the order of ~55K triples/sec converted into
~13K pages/sec on an 2014 i5 Haswell, dual core processor (max 2.1GHz I think)
running Xubuntu, although these numbers probably depend a lot on the structure
of the dataset.

RDF2SMW is very similar to the RDF import function in the
[RDFIO](https://github.com/rdfio/RDFIO) Semantic MediaWiki extension, but takes
another approach: Whereas RDFIO converts RDF to wiki pages and imports them in
the same go, RDF2SMW first converts RDF to an XML file outside of PHP (for
better performance), and then importing using MediaWiki's built-in import
function.

**Status:** The tool is now feature complete, including ability to write facts
via template calls, if a categorization (via owl:Class) of the subject can be
done.  What is lacking is more options to fine-tune things. Right now you'll
have to modify the source code yourself if you need any customization. Hope to
address this in the near future.

For more detailed status, see [TODO.md](https://github.com/samuell/rdf2smw/blob/master/TODO.md)

Installation
------------

For linux 64 bit:

1. Download the file `rdf2smw_linux64.gz`, on the [latest release](https://github.com/samuell/rdf2smw/releases).
2. Unpack it with: `gunzip rdf2smw_linux64.gz`
3. Call it, on the commandline (see the usage section below).


Usage
-----

Call the rdf2smw binary, specifying a file with triples in n-triples or turtle
format, with the `--in` flag, and an output file in XML format with the
`--out` flag, like so:

```bash
./rdf2smw --in triples.nt --out semantic_mediawiki_pages.xml
```

The resulting XML file, can then be imported into MediaWiki / Semantic
MediaWiki, via the `importDump.php` maintenance script, located in the
`maintenance` folder under the main mediawiki folder:

```bash
php <wikidir>/maintenance/importDump.php semantic_mediawiki_pages.xml
```

Technical notes
---------------

RDF2SMW is based on the [FlowBase](https://github.com/flowbase/flowbase)
flow-based programming micro-framework.

