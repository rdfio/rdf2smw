package components

import (
	"github.com/flowbase/flowbase"
	"github.com/spf13/afero"
	"testing"
)

// TestNewOSFileReader tests NewOSTurtleFileReader
func TestNewTurtleFileReader(t *testing.T) {
	flowbase.InitLogWarning()

	fr := NewOsTurtleFileReader()
	if fr.InFileName == nil {
		t.Error("In-port InFileName not initialized in New FileReader")
	}
	if fr.OutTriple == nil {
		t.Error("In-port InFileName not initialized in New FileReader")
	}

	go func() {
		fr.InFileName <- "teststring"
	}()
	teststr1 := <-fr.InFileName
	if teststr1 != "teststring" {
		t.Error("In-port InFileName is not a string channel")
		fr.InFileName <- "teststring"
	}
}

// Tests the main behavior of the TurtleFileReader process
func TestTurtleFileReader(t *testing.T) {
	flowbase.InitLogWarning()

	s1 := "http://example.org/s1"
	p1 := "http://example.org/p1"
	o1 := "string1"
	s2 := "http://example.org/p2"
	p2 := "http://example.org/p2"
	o2 := "string2"
	triple1 := "<" + s1 + "> <" + p1 + "> \"" + o1 + "\" ."
	triple2 := "<" + s2 + "> <" + p2 + "> \"" + o2 + "\" ."
	testContent := triple1 + "\n" + triple2

	fs := afero.NewMemMapFs()

	testFileName := "testfile.ttl"
	f, err := fs.Create(testFileName)
	if err != nil {
		t.Errorf("Could not create file %s in memory file system", testFileName)
	}
	f.WriteString(testContent)
	f.Close()

	fr := NewTurtleFileReader(fs)
	go func() {
		defer close(fr.InFileName)
		fr.InFileName <- testFileName
	}()

	go fr.Run()

	outTriple1 := <-fr.OutTriple
	outTriple2 := <-fr.OutTriple

	if outTriple1.Subj.String() != s1 {
		t.Error("Subject of first triple is wrong")
	}
	if outTriple1.Pred.String() != p1 {
		t.Error("Predicate of first triple is wrong")
	}
	if outTriple1.Obj.String() != o1 {
		t.Error("Object of first triple is wrong")
	}
	if outTriple2.Subj.String() != s2 {
		t.Error("Subject of second triple is wrong")
	}
	if outTriple2.Pred.String() != p2 {
		t.Error("Predicate of second triple is wrong")
	}
	if outTriple2.Obj.String() != o2 {
		t.Error("Object of second triple is wrong")
	}
}
