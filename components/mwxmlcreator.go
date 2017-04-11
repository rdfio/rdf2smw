package components

import (
	"fmt"
	str "strings"
	"time"
)

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
			if page.SpecificCategory.Name != "" {
				templateName = page.SpecificCategory.Name
			} else {
				// Pick last item (biggest chance to be pretty specific?)
				templateName = page.Categories[len(page.Categories)-1].Name
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
					wikiText += cat.Name
				} else {
					wikiText += "," + cat.Name
				}
			}

			wikiText += "\n}}"
		} else {

			// Add fact statements
			for _, fact := range page.Facts {
				wikiText += fact.asWikiFact()
			}

			// Add category statements
			for _, cat := range page.Categories {
				wikiText += cat.asWikiString()
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

func spacesToUnderscores(inStr string) string {
	return str.Replace(inStr, " ", "_", -1)
}

// TODO: Probably move out to separate component!
func escapeWikiChars(inStr string) string {
	outStr := str.Replace(inStr, "[", "(", -1)
	outStr = str.Replace(outStr, "]", ")", -1)
	outStr = str.Replace(outStr, "|", ",", -1)
	outStr = str.Replace(outStr, "=", "-", -1)
	outStr = str.Replace(outStr, "<", "&lt;", -1)
	outStr = str.Replace(outStr, ">", "&gt;", -1)
	return outStr
}
