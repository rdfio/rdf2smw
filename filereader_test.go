package main

import (
	"testing"
)

// -------------------------------------------------------------------------------
// FileReader
// -------------------------------------------------------------------------------

func TestNewFileReader(t *testing.T) {
	fr := NewFileReader()
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
