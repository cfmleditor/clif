package main

import (
	"bytes"
	"encoding/json/v2"
	"strings"
	"testing"
)

func TestWriteScanMarkdownAndJSON(t *testing.T) {
	errs := []scanError{
		{
			file:    "views/broken.cfm",
			lang:    "cfml",
			message: `parse error near "<cfif>"`,
			source:  "<cfif>unclosed",
			line:    2,
			col:     1,
		},
	}

	var md bytes.Buffer
	if err := writeScanMarkdown(&md, errs, 5); err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{"# Parse Errors", "Scanned 5 files: 1 parse error(s) in 1 file(s).", "## `views/broken.cfm` (1)", "`views/broken.cfm:2:1`", "<cfif>unclosed"} {
		if !strings.Contains(md.String(), want) {
			t.Errorf("markdown report missing %q:\n%s", want, md.String())
		}
	}

	var raw bytes.Buffer
	if err := writeScanJSON(&raw, errs, 5); err != nil {
		t.Fatal(err)
	}

	var decoded struct {
		Scanned     int `json:"scanned"`
		FailedFiles int `json:"failedFiles"`
		Errors      []struct {
			File   string `json:"file"`
			Line   int    `json:"line"`
			Col    int    `json:"col"`
			Source string `json:"source"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(raw.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid json: %v\n%s", err, raw.String())
	}

	if decoded.Scanned != 5 || decoded.FailedFiles != 1 || len(decoded.Errors) != 1 || decoded.Errors[0].Source != "<cfif>unclosed" {
		t.Errorf("unexpected json output: %+v", decoded)
	}
}
