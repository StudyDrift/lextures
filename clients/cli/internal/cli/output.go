package cli

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
)

// Format selects how structured data is rendered.
type Format string

const (
	FormatTable  Format = "table"
	FormatJSON   Format = "json"
	FormatNDJSON Format = "ndjson"
	FormatCSV    Format = "csv"
)

// ParseFormat normalizes a format flag value.
func ParseFormat(raw string, jsonAlias bool) Format {
	if jsonAlias {
		return FormatJSON
	}
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "table":
		return FormatTable
	case "json":
		return FormatJSON
	case "ndjson":
		return FormatNDJSON
	case "csv":
		return FormatCSV
	default:
		return FormatTable
	}
}

// Options controls output rendering.
type Options struct {
	Format    Format
	Quiet     bool
	NoHeaders bool
	NoColor   bool
	Stdout    io.Writer
	Stderr    io.Writer
}

func (o Options) out() io.Writer {
	if o.Stdout != nil {
		return o.Stdout
	}
	return os.Stdout
}

func (o Options) err() io.Writer {
	if o.Stderr != nil {
		return o.Stderr
	}
	return os.Stderr
}

// WriteJSON writes raw JSON bytes or encodes v when raw is nil.
func (o Options) WriteJSON(raw []byte, v any) error {
	if o.Quiet {
		return nil
	}
	if len(raw) > 0 {
		_, err := o.out().Write(raw)
		return err
	}
	enc := json.NewEncoder(o.out())
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// WriteRows renders rows in the selected format. headers may be empty for JSON passthrough.
func (o Options) WriteRows(headers []string, rows [][]string, rawJSON []byte) error {
	if o.Quiet {
		return nil
	}
	switch o.Format {
	case FormatJSON:
		if len(rawJSON) > 0 {
			_, err := o.out().Write(rawJSON)
			return err
		}
		return json.NewEncoder(o.out()).Encode(map[string]any{"rows": rows})
	case FormatNDJSON:
		enc := json.NewEncoder(o.out())
		for _, row := range rows {
			obj := map[string]string{}
			for i, h := range headers {
				if i < len(row) {
					obj[h] = row[i]
				}
			}
			if err := enc.Encode(obj); err != nil {
				return err
			}
		}
		return nil
	case FormatCSV:
		w := csv.NewWriter(o.out())
		if !o.NoHeaders && len(headers) > 0 {
			if err := w.Write(headers); err != nil {
				return err
			}
		}
		for _, row := range rows {
			if err := w.Write(row); err != nil {
				return err
			}
		}
		w.Flush()
		return w.Error()
	default:
		tw := tabwriter.NewWriter(o.out(), 0, 0, 2, ' ', 0)
		if !o.NoHeaders && len(headers) > 0 {
			if _, err := fmt.Fprintln(tw, strings.Join(headers, "\t")); err != nil {
				return err
			}
		}
		for _, row := range rows {
			if _, err := fmt.Fprintln(tw, strings.Join(row, "\t")); err != nil {
				return err
			}
		}
		return tw.Flush()
	}
}

// Printf writes to stderr unless quiet.
func (o Options) Printf(format string, args ...any) {
	if o.Quiet {
		return
	}
	_, _ = fmt.Fprintf(o.err(), format, args...)
}