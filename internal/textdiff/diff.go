// Package textdiff renders line-based unified diffs. It is meant for small
// texts such as app specs (LCS is quadratic in the number of lines).
package textdiff

import (
	"fmt"
	"strings"
)

// Op is the kind of a diff line.
type Op byte

// Diff line kinds.
const (
	Equal  Op = ' '
	Delete Op = '-'
	Insert Op = '+'
)

// Line is one line of an edit script.
type Line struct {
	Op   Op
	Text string
}

// Lines returns the edit script turning a into b, line by line.
func Lines(a, b string) []Line {
	x, y := split(a), split(b)
	// lcs[i][j] is the LCS length of x[i:] and y[j:].
	lcs := make([][]int, len(x)+1)
	for i := range lcs {
		lcs[i] = make([]int, len(y)+1)
	}
	for i := len(x) - 1; i >= 0; i-- {
		for j := len(y) - 1; j >= 0; j-- {
			if x[i] == y[j] {
				lcs[i][j] = lcs[i+1][j+1] + 1
			} else {
				lcs[i][j] = max(lcs[i+1][j], lcs[i][j+1])
			}
		}
	}
	var out []Line
	i, j := 0, 0
	for i < len(x) && j < len(y) {
		switch {
		case x[i] == y[j]:
			out = append(out, Line{Equal, x[i]})
			i++
			j++
		case lcs[i+1][j] >= lcs[i][j+1]:
			out = append(out, Line{Delete, x[i]})
			i++
		default:
			out = append(out, Line{Insert, y[j]})
			j++
		}
	}
	for ; i < len(x); i++ {
		out = append(out, Line{Delete, x[i]})
	}
	for ; j < len(y); j++ {
		out = append(out, Line{Insert, y[j]})
	}
	return out
}

// Unified renders a unified diff with the given lines of context. It returns
// "" when the texts are equal.
func Unified(fromName, toName, a, b string, context int) string {
	lines := Lines(a, b)
	var sb strings.Builder
	for _, h := range hunks(lines, context) {
		if sb.Len() == 0 {
			fmt.Fprintf(&sb, "--- %s\n+++ %s\n", fromName, toName)
		}
		fmt.Fprintf(&sb, "@@ -%s +%s @@\n", rangeOf(h.aStart, h.aLen), rangeOf(h.bStart, h.bLen))
		for _, l := range lines[h.from:h.to] {
			sb.WriteByte(byte(l.Op))
			sb.WriteString(l.Text)
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}

type hunk struct {
	from, to     int // indexes into the edit script
	aStart, aLen int
	bStart, bLen int
}

func hunks(lines []Line, context int) []hunk {
	var out []hunk
	n := len(lines)
	for i := 0; i < n; {
		if lines[i].Op == Equal {
			i++
			continue
		}
		start := max(0, i-context)
		end := i
		// Extend while changes are within 2*context equal lines of each other.
		for end < n {
			if lines[end].Op != Equal {
				end++
				continue
			}
			run := end
			for run < n && lines[run].Op == Equal {
				run++
			}
			if run == n || run-end > 2*context {
				end = min(n, end+context)
				break
			}
			end = run
		}
		out = append(out, newHunk(lines, start, end))
		i = end
	}
	return out
}

func newHunk(lines []Line, from, to int) hunk {
	h := hunk{from: from, to: to}
	aLine, bLine := 1, 1
	for _, l := range lines[:from] {
		if l.Op != Insert {
			aLine++
		}
		if l.Op != Delete {
			bLine++
		}
	}
	h.aStart, h.bStart = aLine, bLine
	for _, l := range lines[from:to] {
		if l.Op != Insert {
			h.aLen++
		}
		if l.Op != Delete {
			h.bLen++
		}
	}
	if h.aLen == 0 {
		h.aStart--
	}
	if h.bLen == 0 {
		h.bStart--
	}
	return h
}

func rangeOf(start, length int) string {
	if length == 1 {
		return fmt.Sprint(start)
	}
	return fmt.Sprintf("%d,%d", start, length)
}

func split(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(strings.TrimSuffix(s, "\n"), "\n")
}
