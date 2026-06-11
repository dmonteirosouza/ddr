package app

import (
	"os"
	"strconv"
	"strings"
	"time"
)

func nonEmptyLines(text string) []string {
	rawLines := strings.Split(text, "\n")
	lines := make([]string, 0, len(rawLines))
	for _, line := range rawLines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			lines = append(lines, trimmed)
		}
	}
	return lines
}

func timestamp() string {
	return time.Now().Format("20060102150405")
}

func parseHumanSize(value string) uint64 {
	value = strings.TrimSpace(strings.Trim(value, "()"))
	if value == "" || value == "0B" || value == "0" {
		return 0
	}

	numberPart := ""
	unitPart := ""
	for _, char := range value {
		if (char >= '0' && char <= '9') || char == '.' {
			numberPart += string(char)
		} else {
			unitPart += string(char)
		}
	}

	number, err := strconv.ParseFloat(numberPart, 64)
	if err != nil {
		return 0
	}

	switch strings.ToUpper(strings.TrimSpace(unitPart)) {
	case "B", "":
		return uint64(number)
	case "K", "KB", "KIB":
		return uint64(number * 1024)
	case "M", "MB", "MIB":
		return uint64(number * 1024 * 1024)
	case "G", "GB", "GIB":
		return uint64(number * 1024 * 1024 * 1024)
	case "T", "TB", "TIB":
		return uint64(number * 1024 * 1024 * 1024 * 1024)
	default:
		return 0
	}
}

func parseMemoryToBytes(value string) uint64 {
	return parseHumanSize(value)
}

func mustHomeDir() string {
	dir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	return dir
}

func stripJSONComments(input []byte) []byte {
	output := make([]byte, 0, len(input))
	inString := false
	inLineComment := false
	inBlockComment := false
	escaped := false

	for i := 0; i < len(input); i++ {
		char := input[i]
		var next byte
		if i+1 < len(input) {
			next = input[i+1]
		}

		if inLineComment {
			if char == '\n' {
				inLineComment = false
				output = append(output, char)
			}
			continue
		}

		if inBlockComment {
			if char == '*' && next == '/' {
				inBlockComment = false
				i++
			}
			continue
		}

		if !inString && char == '/' && next == '/' {
			inLineComment = true
			i++
			continue
		}

		if !inString && char == '/' && next == '*' {
			inBlockComment = true
			i++
			continue
		}

		output = append(output, char)

		if char == '"' && !escaped {
			inString = !inString
		}

		escaped = char == '\\' && !escaped
		if char != '\\' {
			escaped = false
		}
	}

	return output
}
