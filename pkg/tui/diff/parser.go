package diff

import (
	"regexp"
	"strconv"
	"strings"
)

// Line represents a single line in a diff
type Line struct {
	Type    LineType
	Content string
	OldNum  int // Line number in old file (0 if added)
	NewNum  int // Line number in new file (0 if removed)
}

// LineType represents the type of diff line
type LineType int

const (
	LineContext LineType = iota
	LineAdded
	LineRemoved
	LineHeader
	LineHunk
)

// ParsedDiff represents a parsed unified diff
type ParsedDiff struct {
	Lines    []Line
	OldLabel string // "--- a/ConfigMap..."
	NewLabel string // "+++ b/ConfigMap..."
}

// ParseUnifiedDiff parses unified diff text into structured format
func ParseUnifiedDiff(diffText string) (*ParsedDiff, error) {
	lines := strings.Split(diffText, "\n")
	parsed := &ParsedDiff{
		Lines: make([]Line, 0, len(lines)),
	}

	oldLineNum := 0
	newLineNum := 0
	hunkRegex := regexp.MustCompile(`^@@ -(\d+),?\d* \+(\d+),?\d* @@`)

	for _, line := range lines {
		if strings.HasPrefix(line, "---") {
			parsed.OldLabel = strings.TrimPrefix(line, "--- ")
			parsed.Lines = append(parsed.Lines, Line{
				Type:    LineHeader,
				Content: line,
			})
		} else if strings.HasPrefix(line, "+++") {
			parsed.NewLabel = strings.TrimPrefix(line, "+++ ")
			parsed.Lines = append(parsed.Lines, Line{
				Type:    LineHeader,
				Content: line,
			})
		} else if strings.HasPrefix(line, "@@") {
			// Parse hunk header to get starting line numbers
			matches := hunkRegex.FindStringSubmatch(line)
			if len(matches) >= 3 {
				oldStart, _ := strconv.Atoi(matches[1])
				newStart, _ := strconv.Atoi(matches[2])
				oldLineNum = oldStart - 1 // Will be incremented on first use
				newLineNum = newStart - 1
			}
			parsed.Lines = append(parsed.Lines, Line{
				Type:    LineHunk,
				Content: line,
			})
		} else if strings.HasPrefix(line, "+") {
			newLineNum++
			parsed.Lines = append(parsed.Lines, Line{
				Type:    LineAdded,
				Content: line[1:], // Remove + prefix
				OldNum:  0,
				NewNum:  newLineNum,
			})
		} else if strings.HasPrefix(line, "-") {
			oldLineNum++
			parsed.Lines = append(parsed.Lines, Line{
				Type:    LineRemoved,
				Content: line[1:], // Remove - prefix
				OldNum:  oldLineNum,
				NewNum:  0,
			})
		} else if line != "" {
			// Context line (unchanged)
			oldLineNum++
			newLineNum++
			parsed.Lines = append(parsed.Lines, Line{
				Type:    LineContext,
				Content: line,
				OldNum:  oldLineNum,
				NewNum:  newLineNum,
			})
		}
	}

	return parsed, nil
}
