package util

import (
	"fmt"
	"strings"
)

// FormatOutput formats the output of a tool in a user-friendly way
func FormatOutput(result map[string]any) {
	for key, value := range result {
		fmt.Printf("%s: ", key)
		formatValue(value, 0)
		fmt.Println()
	}
}

// formatValue formats a value based on its type
func formatValue(value interface{}, indent int) {
	if value == nil {
		fmt.Print("null")
		return
	}

	switch v := value.(type) {
	case string:
		fmt.Print(v)
	case bool, int, int32, int64, float32, float64:
		fmt.Print(v)
	case []interface{}:
		// Print arrays
		if len(v) == 0 {
			fmt.Print("[]")
			return
		}

		fmt.Println()
		for i, item := range v {
			fmt.Print(strings.Repeat("  ", indent+1))
			fmt.Printf("%d: ", i)
			formatValue(item, indent+1)
			fmt.Println()
		}
	case map[string]interface{}:
		// Print nested maps
		if len(v) == 0 {
			fmt.Print("{}")
			return
		}

		fmt.Println()
		for k, val := range v {
			fmt.Print(strings.Repeat("  ", indent+1))
			fmt.Printf("%s: ", k)
			formatValue(val, indent+1)
			fmt.Println()
		}
	default:
		// Use default format for other types
		fmt.Print(v)
	}
}
