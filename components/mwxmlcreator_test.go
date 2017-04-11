package components

import (
	"github.com/flowbase/flowbase"
	"testing"
)

// TestNewMWXMLCreator tests NewMWXMLCreator
func TestNewMWXMLCreator(t *testing.T) {
	flowbase.InitLogDebug()

	mxc := NewMWXMLCreator(true)

	if mxc.InWikiPage == nil {
		t.Error("InWikiPage is not initialized")
	}
	if mxc.OutTemplates == nil {
		t.Error("OutTemplates is not initialized")
	}
	if mxc.OutProperties == nil {
		t.Error("OutProperties is not initialized")
	}
	if mxc.OutPages == nil {
		t.Error("OutPages is not initialized")
	}
	if mxc.UseTemplates != true {
		t.Error("UseTemplates field is initialized wrongly")
	}
}
