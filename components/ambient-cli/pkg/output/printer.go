package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

type Format string

const (
	FormatTable Format = "table"
	FormatJSON  Format = "json"
	FormatWide  Format = "wide"
)

type Printer struct {
	writer io.Writer
	format Format
}

func NewPrinter(format Format) *Printer {
	return &Printer{
		writer: os.Stdout,
		format: format,
	}
}

func (p *Printer) Writer() io.Writer {
	return p.writer
}

func (p *Printer) Format() Format {
	return p.format
}

func (p *Printer) PrintJSON(v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal JSON: %w", err)
	}
	_, err = fmt.Fprintln(p.writer, string(data))
	return err
}
