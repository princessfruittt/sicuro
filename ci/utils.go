package ci

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
)

func findCommit(f *os.File, input []byte, cmmt, stts string) (exist bool) {
	var isContains = false
	var output = ""
	lines := strings.Split(string(input), "\n")

	for i, line := range lines {
		if strings.Contains(line, cmmt) {
			lines[i] = fmt.Sprintf("%s=%s", cmmt, stts)
			isContains = true
		}
	}
	if !isContains {
		output = fmt.Sprintf("%s=%s\n", cmmt, stts)
	}
	output = fmt.Sprintf("%s%s", output, strings.Join(lines[:], "\n"))
	f.WriteAt([]byte(output), 0)
	return
}

func readByte(f *os.File) (input []byte) {
	stat, _ := f.Stat()
	input = make([]byte, stat.Size())
	_, err := f.Read(input)
	if err != nil {
		log.Printf("Something wrong with bisect file. %s", err)
	}
	return input
}

func parse(input []byte) (res string) {
	lines := strings.Split(string(input), "\n")
	for i, line := range lines {
		if strings.Contains(line, "success") {
			return strings.Split(lines[i], "=")[0]
		}
	}
	return
}

var exportRegex = regexp.MustCompile(`^\s*(?:export\s+)?(.*?)\s*$`)

var (
	singleQuotesRegex  = regexp.MustCompile(`\A'(.*)'\z`)
	doubleQuotesRegex  = regexp.MustCompile(`\A"(.*)"\z`)
	escapeRegex        = regexp.MustCompile(`\\.`)
	unescapeCharsRegex = regexp.MustCompile(`\\([^$])`)
)

func parseValue(value string, envMap map[string]string) string {

	// trim
	value = strings.Trim(value, " ")

	// check if we've got quoted values or possible escapes
	if len(value) > 1 {
		singleQuotes := singleQuotesRegex.FindStringSubmatch(value)

		doubleQuotes := doubleQuotesRegex.FindStringSubmatch(value)

		if singleQuotes != nil || doubleQuotes != nil {
			// pull the quotes off the edges
			value = value[1 : len(value)-1]
		}

		if doubleQuotes != nil {
			// expand newlines
			value = escapeRegex.ReplaceAllStringFunc(value, func(match string) string {
				c := strings.TrimPrefix(match, `\`)
				switch c {
				case "n":
					return "\n"
				case "r":
					return "\r"
				default:
					return match
				}
			})
			// unescape characters
			value = unescapeCharsRegex.ReplaceAllString(value, "$1")
		}

		if singleQuotes == nil {
			value = expandVariables(value, envMap)
		}
	}

	return value
}

var expandVarRegex = regexp.MustCompile(`(\\)?(\$)(\()?\{?([A-Z0-9_]+)?\}?`)

func expandVariables(v string, m map[string]string) string {
	return expandVarRegex.ReplaceAllStringFunc(v, func(s string) string {
		submatch := expandVarRegex.FindStringSubmatch(s)

		if submatch == nil {
			return s
		}
		if submatch[1] == "\\" || submatch[2] == "(" {
			return submatch[0][1:]
		} else if submatch[4] != "" {
			return m[submatch[4]]
		}
		return s
	})
}
