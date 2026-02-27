package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestNewPrinterDefaultsToStdout(t *testing.T) {
	p := NewPrinter(FormatJSON)
	if p.Writer() == nil {
		t.Error("expected non-nil writer")
	}
	if p.Format() != FormatJSON {
		t.Errorf("expected FormatJSON, got %s", p.Format())
	}
}

func TestNewPrinterCustomWriter(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinter(FormatTable, &buf)
	if p.Writer() != &buf {
		t.Error("expected custom writer")
	}
}

func TestPrintJSON(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinter(FormatJSON, &buf)

	data := map[string]string{"key": "value"}
	if err := p.PrintJSON(data); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, `"key"`) {
		t.Errorf("expected JSON key, got %s", out)
	}
	if !strings.Contains(out, `"value"`) {
		t.Errorf("expected JSON value, got %s", out)
	}
}
