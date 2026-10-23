package spec

import (
	"fmt"
	"sort"
	"strings"
)

// Problem is one parse or validation error at a position.
type Problem struct {
	Pos     Pos
	Message string
}

func (p Problem) String() string {
	if p.Pos.Line == 0 {
		return p.Message
	}
	return fmt.Sprintf("line %d, column %d: %s", p.Pos.Line, p.Pos.Column, p.Message)
}

// Problems is a list of problems; it is an error when non-empty.
type Problems []Problem

func (ps Problems) Error() string {
	lines := make([]string, len(ps))
	for i, p := range ps {
		lines[i] = p.String()
	}
	return strings.Join(lines, "\n")
}

func (ps *Problems) add(pos Pos, format string, args ...any) {
	*ps = append(*ps, Problem{Pos: pos, Message: fmt.Sprintf(format, args...)})
}

// sorted returns ps ordered by position, or nil when empty.
func (ps Problems) sorted() error {
	if len(ps) == 0 {
		return nil
	}
	sort.SliceStable(ps, func(i, j int) bool {
		if ps[i].Pos.Line != ps[j].Pos.Line {
			return ps[i].Pos.Line < ps[j].Pos.Line
		}
		return ps[i].Pos.Column < ps[j].Pos.Column
	})
	return ps
}
