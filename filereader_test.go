package main

import (
	"github.com/flowbase/flowbase"
	"github.com/spf13/afero"
	"testing"
)

// TestNewOSFileReader tests NewOSFileReader
func TestNewOSFileReader(t *testing.T) {
	flowbase.InitLogWarning()

	fr := NewOsFileReader()
	if fr.InFileName == nil {
		t.Error("In-port InFileName not initialized in New FileReader")
	}
	if fr.OutLine == nil {
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

	go func() {
		fr.OutLine <- "teststring"
	}()
	teststr2 := <-fr.OutLine
	if teststr2 != "teststring" {
		t.Error("Out-port OutLine is not a string channel")
	}
}

// Tests the main behavior of the FileReader process
func TestFileReader(t *testing.T) {
	flowbase.InitLogWarning()

	testFileName := "testfile.txt"
	line1 := "line one"
	line2 := "line two"
	testContent := line1 + "\n" + line2

	fs := afero.NewMemMapFs()

	f, err := fs.Create(testFileName)
	if err != nil {
		t.Errorf("Could not create file %s in memory file system", testFileName)
	}
	f.WriteString(testContent)
	f.Close()

	tmp := []byte{}
	f.Read(tmp)

	println(string(tmp))

	fr := NewFileReader(fs)
	go func() {
		defer close(fr.InFileName)
		fr.InFileName <- testFileName
	}()

	go fr.Run()

	outStr1 := <-fr.OutLine
	outStr2 := <-fr.OutLine

	if outStr1 != line1 {
		t.Error("First output from file reader does not match first line in file")
	}
	if outStr2 != line2 {
		t.Error("Second output from file reader does not match second line in file")
	}
}
