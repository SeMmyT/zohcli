package output

import (
	"io"
	"strings"

	"github.com/rodaine/table"
)

// RenderTable renders a table to the writer for rich mode
func RenderTable(w io.Writer, columns []Column, rows []map[string]string) {
	if len(rows) == 0 {
		return
	}

	// Create header row
	headerFmt := table.New(w)
	headers := make([]interface{}, len(columns))
	for i, col := range columns {
		headers[i] = col.Name
	}
	headerFmt.AddRow(headers...)

	// Add data rows
	for _, row := range rows {
		rowData := make([]interface{}, len(columns))
		for i, col := range columns {
			value := row[col.Key]
			// Truncate if width is specified and value exceeds it
			if col.Width > 0 && len(value) > col.Width {
				value = value[:col.Width-3] + "..."
			}
			rowData[i] = value
		}
		headerFmt.AddRow(rowData...)
	}
}

// TruncateString truncates a string to maxLen and adds "..." if needed
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen < 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// PadString pads a string to the specified width
func PadString(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}
