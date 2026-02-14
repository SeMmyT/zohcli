package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"

	"github.com/charmbracelet/lipgloss/v2"
	"github.com/muesli/termenv"
)

// Formatter is the interface for output formatting
type Formatter interface {
	Print(data any) error
	PrintList(items any, columns []Column) error
	PrintError(err error)
	PrintHint(msg string)
}

// Column defines a column for table/list output
type Column struct {
	Name  string // Display name
	Key   string // Struct field name or map key
	Width int    // Width for rich mode (0 = auto)
}

// New creates a formatter for the specified mode
func New(mode string) Formatter {
	switch mode {
	case "json":
		return &jsonFormatter{}
	case "plain":
		return &plainFormatter{}
	case "rich":
		profile := termenv.ColorProfile()
		return &richFormatter{profile: profile}
	default:
		return &plainFormatter{}
	}
}

// NewJSON creates a JSON formatter with optional results-only mode
func NewJSON(resultsOnly bool) Formatter {
	return &jsonFormatter{resultsOnly: resultsOnly}
}

// jsonFormatter outputs JSON to stdout
type jsonFormatter struct {
	resultsOnly bool
}

func (f *jsonFormatter) Print(data any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

func (f *jsonFormatter) PrintList(items any, columns []Column) error {
	// If results-only mode, print raw array
	if f.resultsOnly {
		return f.Print(items)
	}

	// Otherwise, wrap in envelope with metadata
	v := reflect.ValueOf(items)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	count := 0
	if v.Kind() == reflect.Slice {
		count = v.Len()
	}

	envelope := map[string]any{
		"data":  items,
		"count": count,
	}

	return f.Print(envelope)
}

func (f *jsonFormatter) PrintError(err error) {
	errObj := map[string]string{"error": err.Error()}
	enc := json.NewEncoder(os.Stderr)
	enc.SetIndent("", "  ")
	enc.Encode(errObj)
}

func (f *jsonFormatter) PrintHint(msg string) {
	// Only print hints in verbose mode for JSON
	// For now, skip hints in JSON mode
}

// plainFormatter outputs tab-separated values
type plainFormatter struct{}

func (f *plainFormatter) Print(data any) error {
	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() == reflect.Struct {
		t := v.Type()
		for i := 0; i < v.NumField(); i++ {
			field := t.Field(i)
			value := v.Field(i)
			fmt.Fprintf(os.Stdout, "%s\t%v\n", field.Name, value.Interface())
		}
		return nil
	}

	// For non-struct types, just print the value
	fmt.Fprintf(os.Stdout, "%v\n", data)
	return nil
}

func (f *plainFormatter) PrintList(items any, columns []Column) error {
	v := reflect.ValueOf(items)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Slice {
		return fmt.Errorf("PrintList requires a slice")
	}

	// Print header
	headers := make([]string, len(columns))
	for i, col := range columns {
		headers[i] = col.Name
	}
	fmt.Fprintf(os.Stdout, "%s\n", strings.Join(headers, "\t"))

	// Print rows
	for i := 0; i < v.Len(); i++ {
		item := v.Index(i)
		if item.Kind() == reflect.Ptr {
			item = item.Elem()
		}

		values := make([]string, len(columns))
		for j, col := range columns {
			if item.Kind() == reflect.Map {
				mapVal := item.MapIndex(reflect.ValueOf(col.Key))
				if mapVal.IsValid() {
					values[j] = fmt.Sprintf("%v", mapVal.Interface())
				}
			} else if item.Kind() == reflect.Struct {
				field := item.FieldByName(col.Key)
				if field.IsValid() {
					values[j] = fmt.Sprintf("%v", field.Interface())
				}
			}
		}
		fmt.Fprintf(os.Stdout, "%s\n", strings.Join(values, "\t"))
	}

	return nil
}

func (f *plainFormatter) PrintError(err error) {
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
}

func (f *plainFormatter) PrintHint(msg string) {
	fmt.Fprintf(os.Stderr, "hint: %v\n", msg)
}

// richFormatter outputs styled content for terminal
type richFormatter struct {
	profile termenv.Profile
}

func (f *richFormatter) Print(data any) error {
	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() == reflect.Struct {
		t := v.Type()
		for i := 0; i < v.NumField(); i++ {
			field := t.Field(i)
			value := v.Field(i)

			keyStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("33"))
			valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("15"))

			fmt.Fprintf(os.Stdout, "%s: %s\n",
				keyStyle.Render(field.Name),
				valueStyle.Render(fmt.Sprintf("%v", value.Interface())),
			)
		}
		return nil
	}

	// For non-struct types, just print the value
	fmt.Fprintf(os.Stdout, "%v\n", data)
	return nil
}

func (f *richFormatter) PrintList(items any, columns []Column) error {
	v := reflect.ValueOf(items)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Slice {
		return fmt.Errorf("PrintList requires a slice")
	}

	// Convert to map format for table rendering
	rows := make([]map[string]string, v.Len())
	for i := 0; i < v.Len(); i++ {
		item := v.Index(i)
		if item.Kind() == reflect.Ptr {
			item = item.Elem()
		}

		row := make(map[string]string)
		for _, col := range columns {
			if item.Kind() == reflect.Map {
				mapVal := item.MapIndex(reflect.ValueOf(col.Key))
				if mapVal.IsValid() {
					row[col.Key] = fmt.Sprintf("%v", mapVal.Interface())
				}
			} else if item.Kind() == reflect.Struct {
				field := item.FieldByName(col.Key)
				if field.IsValid() {
					row[col.Key] = fmt.Sprintf("%v", field.Interface())
				}
			}
		}
		rows[i] = row
	}

	RenderTable(os.Stdout, columns, rows)
	return nil
}

func (f *richFormatter) PrintError(err error) {
	errorStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("9"))

	fmt.Fprintf(os.Stderr, "%s\n", errorStyle.Render("error: "+err.Error()))
}

func (f *richFormatter) PrintHint(msg string) {
	hintStyle := lipgloss.NewStyle().
		Faint(true).
		Foreground(lipgloss.Color("8"))

	fmt.Fprintf(os.Stderr, "%s\n", hintStyle.Render("hint: "+msg))
}

// Helper to write to writer for table rendering
type writerWrapper struct {
	w io.Writer
}

func (ww *writerWrapper) Write(p []byte) (n int, err error) {
	return ww.w.Write(p)
}
