rdf2smw
=======

[![CircleCI](https://img.shields.io/circleci/project/github/rdfio/rdf2smw.svg)](https://circleci.com/gh/rdfio/rdf2smw)
[![Test Coverage](https://img.shields.io/codecov/c/github/rdfio/rdf2smw.svg)](https://codecov.io/gh/rdfio/rdf2smw)
[![Code Climate Rating](https://img.shields.io/codeclimate/github/rdfio/rdf2smw.svg)](https://codeclimate.com/github/rdfio/rdf2smw)
[![Code Climate Issues](https://img.shields.io/codeclimate/issues/github/rdfio/rdf2smw.svg)](https://codeclimate.com/github/rdfio/rdf2smw)
[![GoDoc](https://godoc.org/github.com/rdfio/rdf2smw?status.svg)](https://godoc.org/github.com/rdfio/rdf2smw)

News
----

**rdf2smw** was covered in a talk at SMWCon in Frankfurt, Sep 2017. See: [Talk page](https://www.semantic-mediawiki.org/wiki/SMWCon_Fall_2016/Batch_import_of_large_RDF_datasets_using_RDFIO_or_the_new_rdf2smw_tool), [Slides](https://www.slideshare.net/SamuelLampa/batch-import-of-large-rdf-datasets-into-semantic-mediawiki), [Video](https://www.youtube.com/watch?v=k70er1u1ZYs).

Import / convert RDF data into a Semantic MediaWiki
---------------------------------------------------

A commandline tool to convert from RDF triples to [Semantic MediaWiki](http://semantic-mediawiki.org) facts
in MediaWiki XML export format to be used with [MediaWiki](https://www.mediawiki.org)'s built-in
[XML import feature](https://www.mediawiki.org/wiki/Manual:Importing_XML_dumps).

This allows you to quickly and simply populate a Semantic MediaWiki page
structure, from an RDF data file.

It is written in Go for better performance than PHP. The latest version
processes triples into pages in the order of ~55K triples/sec converted into
~13K pages/sec on an 2014 i5 Haswell dual core processor, to give an idea.

rdf2smw is very similar to the RDF import function in the
[RDFIO](https://github.com/rdfio/RDFIO) Semantic MediaWiki extension, but takes
another approach: Whereas RDFIO converts RDF to wiki pages and imports them in
the same go, rdf2smw first converts RDF to an XML file outside of PHP (for
better performance), and then importing using MediaWiki's built-in import
function.

**Status:** The tool is pretty much feature complete, including ability to
write facts via template calls if a categorization (via owl:Class or rdf:type)
of the subject can be done.  What is lacking is more options to fine-tune
things. Right now you'll have to modify the source code yourself if you need
any customization. Hope to address this in the near future.

Dependencies
------------

The tool itself does not have any dependencies, apart from a unix-like
operating system. For importing the generated XML dump file to make sense
though, you will need a web server, PHP, MediaWiki and Semantic MediaWiki.

An automated virtualbox generation script (so valled "vagrant box"), with all
of this, plus the RDFIO extension, can be found
[here](https://github.com/samuell/rdfio-vagrantbox), and is highly recommended,
if you don't have a MediaWiki / SemanticMediawiki installation already!

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

In addition to the specified output file, there will be separate files for
templates and properties, named similar to the main output file, but replacing
`.xml` with `_templates.xml` and `_properties.xml` respectively.

These XML files can then be imported into MediaWiki / Semantic MediaWiki, via
the `importDump.php` maintenance script, located in the `maintenance` folder
under the main mediawiki folder.

```bash
php <wikidir>/maintenance/importDump.php semantic_mediawiki_pages_templates.xml
php <wikidir>/maintenance/importDump.php semantic_mediawiki_pages_properties.xml
php <wikidir>/maintenance/importDump.php semantic_mediawiki_pages.xml
```

Note that the order above is highly recommended (templates, then properties,
then the rest), so as to avoid unnecessary re-computing of semantic data after
the import is done.

Known limitations
-----------------

Only N-triples is supported as input format right now. We plan to add more formats shortly.

Technical notes
---------------

rdf2smw is based on the [FlowBase](https://github.com/flowbase/flowbase)
flow-based programming micro-framework.

Acknowledgements
----------------

rdf2smw makes heavy use of [Petter Goksøyr Åsen](https://github.com/boutros)'s awesome [RDF parsing library](https://github.com/knakk/rdf).
