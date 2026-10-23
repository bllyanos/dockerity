package dockerx

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"google.golang.org/protobuf/encoding/protowire"
)

// BuildKit progress arrives as a StatusResponse protobuf (docs/design.md 6.1).
// Only the fields Dockerity shows are decoded:
//
//	StatusResponse { repeated Vertex vertexes = 1; repeated VertexLog logs = 3; }
//	Vertex         { string digest = 1; string name = 3; bool cached = 4;
//	                 Timestamp started = 5; Timestamp completed = 6; string error = 7; }
//	VertexLog      { string vertex = 1; int64 stream = 3; bytes msg = 4; }
//	Timestamp      { int64 seconds = 1; int32 nanos = 2; }

type vertex struct {
	digest    string
	name      string
	cached    bool
	started   time.Time
	completed time.Time
	err       string
}

type vertexLog struct {
	vertex string
	msg    []byte
}

type status struct {
	vertexes []vertex
	logs     []vertexLog
}

var errTruncated = errors.New("truncated protobuf")

// fields calls fn for every field of a protobuf message.
func fields(b []byte, fn func(num protowire.Number, typ protowire.Type, val []byte, v uint64) error) error {
	for len(b) > 0 {
		num, typ, n := protowire.ConsumeTag(b)
		if n < 0 {
			return errTruncated
		}
		b = b[n:]
		var (
			val []byte
			v   uint64
		)
		switch typ {
		case protowire.VarintType:
			v, n = protowire.ConsumeVarint(b)
		case protowire.BytesType:
			val, n = protowire.ConsumeBytes(b)
		default:
			n = protowire.ConsumeFieldValue(num, typ, b)
		}
		if n < 0 {
			return errTruncated
		}
		b = b[n:]
		if err := fn(num, typ, val, v); err != nil {
			return err
		}
	}
	return nil
}

func decodeStatus(b []byte) (status, error) {
	var s status
	err := fields(b, func(num protowire.Number, typ protowire.Type, val []byte, _ uint64) error {
		if typ != protowire.BytesType {
			return nil
		}
		switch num {
		case 1:
			v, err := decodeVertex(val)
			if err != nil {
				return err
			}
			s.vertexes = append(s.vertexes, v)
		case 3:
			l, err := decodeLog(val)
			if err != nil {
				return err
			}
			s.logs = append(s.logs, l)
		}
		return nil
	})
	return s, err
}

func decodeVertex(b []byte) (vertex, error) {
	var v vertex
	err := fields(b, func(num protowire.Number, _ protowire.Type, val []byte, x uint64) error {
		var err error
		switch num {
		case 1:
			v.digest = string(val)
		case 3:
			v.name = string(val)
		case 4:
			v.cached = x != 0
		case 5:
			v.started, err = decodeTimestamp(val)
		case 6:
			v.completed, err = decodeTimestamp(val)
		case 7:
			v.err = string(val)
		}
		return err
	})
	return v, err
}

func decodeLog(b []byte) (vertexLog, error) {
	var l vertexLog
	err := fields(b, func(num protowire.Number, _ protowire.Type, val []byte, _ uint64) error {
		switch num {
		case 1:
			l.vertex = string(val)
		case 4:
			l.msg = append([]byte(nil), val...)
		}
		return nil
	})
	return l, err
}

func decodeTimestamp(b []byte) (time.Time, error) {
	var sec, nanos int64
	err := fields(b, func(num protowire.Number, _ protowire.Type, _ []byte, x uint64) error {
		switch num {
		case 1:
			sec = int64(x)
		case 2:
			nanos = int64(x)
		}
		return nil
	})
	return time.Unix(sec, nanos), err
}

// traceRenderer turns BuildKit status updates into plain log lines, in the
// style of `docker build --progress=plain`.
type traceRenderer struct {
	out     io.Writer
	numbers map[string]int
	shown   map[string]bool // started line printed
	done    map[string]bool // final line printed
}

func newTraceRenderer(out io.Writer) *traceRenderer {
	return &traceRenderer{out: out, numbers: map[string]int{}, shown: map[string]bool{}, done: map[string]bool{}}
}

func (r *traceRenderer) number(digest string) int {
	n, ok := r.numbers[digest]
	if !ok {
		n = len(r.numbers) + 1
		r.numbers[digest] = n
	}
	return n
}

func (r *traceRenderer) render(s status) error {
	var sb strings.Builder
	for _, v := range s.vertexes {
		n := r.number(v.digest)
		if !r.shown[v.digest] && (!v.started.IsZero() || v.cached) {
			r.shown[v.digest] = true
			fmt.Fprintf(&sb, "#%d %s\n", n, v.name)
		}
		if r.done[v.digest] {
			continue
		}
		switch {
		case v.err != "":
			r.done[v.digest] = true
			fmt.Fprintf(&sb, "#%d ERROR: %s\n", n, v.err)
		case v.cached:
			r.done[v.digest] = true
			fmt.Fprintf(&sb, "#%d CACHED\n", n)
		case !v.completed.IsZero():
			r.done[v.digest] = true
			fmt.Fprintf(&sb, "#%d DONE %.1fs\n", n, v.completed.Sub(v.started).Seconds())
		}
	}
	for _, l := range s.logs {
		n := r.number(l.vertex)
		for _, line := range strings.SplitAfter(string(l.msg), "\n") {
			if line == "" {
				continue
			}
			fmt.Fprintf(&sb, "#%d %s", n, line)
			if !strings.HasSuffix(line, "\n") {
				sb.WriteByte('\n')
			}
		}
	}
	_, err := io.WriteString(r.out, sb.String())
	return err
}
