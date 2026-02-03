// Package output handles formatting and displaying CLI output.
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/ramarlina/mesh-cli/pkg/api"
)

// Format represents the output format type.
type Format int

const (
	FormatHuman Format = iota
	FormatJSON
	FormatRaw
)

// Printer handles output formatting.
type Printer struct {
	writer io.Writer
	format Format
	quiet  bool
	noANSI bool
}

// New creates a new output printer.
func New(format Format, quiet, noANSI bool) *Printer {
	return &Printer{
		writer: os.Stdout,
		format: format,
		quiet:  quiet,
		noANSI: noANSI,
	}
}

// Success prints a success response.
func (p *Printer) Success(result interface{}) error {
	switch p.format {
	case FormatJSON:
		return p.printJSON(api.Response[interface{}]{
			OK:     true,
			Result: result,
		})
	case FormatRaw:
		// For raw output, just print the result as-is
		fmt.Fprintf(p.writer, "%v\n", result)
		return nil
	default:
		// Human-readable format
		if !p.quiet {
			fmt.Fprintf(p.writer, "%v\n", result)
		}
		return nil
	}
}

// Error prints an error response.
func (p *Printer) Error(err error) error {
	switch p.format {
	case FormatJSON:
		return p.printJSON(api.Response[interface{}]{
			OK: false,
			Error: &api.Error{
				Code:    "error",
				Message: err.Error(),
			},
		})
	default:
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return nil
	}
}

// APIError prints an API error response.
func (p *Printer) APIError(apiErr *api.Error) error {
	switch p.format {
	case FormatJSON:
		return p.printJSON(api.Response[interface{}]{
			OK:    false,
			Error: apiErr,
		})
	default:
		fmt.Fprintf(os.Stderr, "error: %s: %s\n", apiErr.Code, apiErr.Message)
		if len(apiErr.Details) > 0 {
			fmt.Fprintf(os.Stderr, "details: %v\n", apiErr.Details)
		}
		return nil
	}
}

// Print prints arbitrary data.
func (p *Printer) Print(format string, args ...interface{}) {
	if p.quiet && p.format != FormatJSON {
		return
	}
	fmt.Fprintf(p.writer, format, args...)
}

// Printf prints formatted data.
func (p *Printer) Printf(format string, args ...interface{}) {
	if p.quiet && p.format != FormatJSON {
		return
	}
	fmt.Fprintf(p.writer, format, args...)
}

// Println prints a line of arbitrary data.
func (p *Printer) Println(args ...interface{}) {
	if p.quiet && p.format != FormatJSON {
		return
	}
	fmt.Fprintln(p.writer, args...)
}

// Table prints data in table format (only in human mode).
func (p *Printer) Table(headers []string, rows [][]string) error {
	if p.format != FormatHuman {
		return nil
	}

	if len(headers) == 0 || len(rows) == 0 {
		return nil
	}

	// Calculate column widths
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}

	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	// Print header
	for i, h := range headers {
		fmt.Fprintf(p.writer, "%-*s  ", widths[i], h)
	}
	fmt.Fprintln(p.writer)

	// Print separator
	for i := range headers {
		for j := 0; j < widths[i]; j++ {
			fmt.Fprint(p.writer, "-")
		}
		fmt.Fprint(p.writer, "  ")
	}
	fmt.Fprintln(p.writer)

	// Print rows
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) {
				fmt.Fprintf(p.writer, "%-*s  ", widths[i], cell)
			}
		}
		fmt.Fprintln(p.writer)
	}

	return nil
}

// printJSON marshals and prints JSON output.
func (p *Printer) printJSON(v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}

	fmt.Fprintf(p.writer, "%s\n", data)
	return nil
}

// IsJSON returns true if the output format is JSON.
func (p *Printer) IsJSON() bool {
	return p.format == FormatJSON
}

// IsRaw returns true if the output format is raw.
func (p *Printer) IsRaw() bool {
	return p.format == FormatRaw
}

// IsQuiet returns true if quiet mode is enabled.
func (p *Printer) IsQuiet() bool {
	return p.quiet
}
