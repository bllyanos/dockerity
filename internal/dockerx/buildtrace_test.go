package dockerx

import (
	"bytes"
	"encoding/base64"
	"testing"
	"time"

	"google.golang.org/protobuf/encoding/protowire"
)

func pbString(b []byte, num protowire.Number, s string) []byte {
	b = protowire.AppendTag(b, num, protowire.BytesType)
	return protowire.AppendString(b, s)
}

func pbMessage(b []byte, num protowire.Number, msg []byte) []byte {
	b = protowire.AppendTag(b, num, protowire.BytesType)
	return protowire.AppendBytes(b, msg)
}

func pbVarint(b []byte, num protowire.Number, v uint64) []byte {
	b = protowire.AppendTag(b, num, protowire.VarintType)
	return protowire.AppendVarint(b, v)
}

func pbTime(t time.Time) []byte {
	return pbVarint(pbVarint(nil, 1, uint64(t.Unix())), 2, uint64(t.Nanosecond()))
}

func TestDecodeAndRenderTrace(t *testing.T) {
	t0 := time.Unix(1700000000, 0)
	from := pbString(pbString(nil, 1, "sha256:a"), 3, "[1/2] FROM busybox")
	from = pbVarint(from, 4, 1)
	run := pbString(pbString(nil, 1, "sha256:b"), 3, "[2/2] RUN echo hi")
	run = pbMessage(run, 5, pbTime(t0))
	run = pbVarint(run, 99, 7) // unknown fields are skipped
	log := pbMessage(pbString(nil, 1, "sha256:b"), 4, []byte("hi\nthere"))
	done := pbMessage(pbMessage(pbString(nil, 1, "sha256:b"), 5, pbTime(t0)), 6, pbTime(t0.Add(1500*time.Millisecond)))

	var out bytes.Buffer
	r := newTraceRenderer(&out)
	for _, msg := range [][]byte{
		pbMessage(pbMessage(nil, 1, from), 1, run),
		pbMessage(nil, 3, log),
		pbMessage(nil, 1, done),
		pbMessage(nil, 1, done), // repeated updates print nothing new
	} {
		s, err := decodeStatus(msg)
		if err != nil {
			t.Fatal(err)
		}
		if err := r.render(s); err != nil {
			t.Fatal(err)
		}
	}
	want := "#1 [1/2] FROM busybox\n#1 CACHED\n#2 [2/2] RUN echo hi\n#2 hi\n#2 there\n#2 DONE 1.5s\n"
	if out.String() != want {
		t.Fatalf("got:\n%s\nwant:\n%s", out.String(), want)
	}
}

func TestDecodeRealTraceMessage(t *testing.T) {
	// A log message captured from Docker 29 during the BuildKit spike (#65).
	raw, err := base64.StdEncoding.DecodeString("GmEKR3NoYTI1Njo1NjdmZjJlMjU2OWVjNjE0NGM2Nzk1YmY1N2U0MDE2OTliNjVjNDQxNjhjNjJjZWVmNTJjZDIzZTVlYjZkNjNmEgwI2dnr1gYQwMuUyAIYASIGaGVsbG8K")
	if err != nil {
		t.Fatal(err)
	}
	s, err := decodeStatus(raw)
	if err != nil || len(s.logs) != 1 || string(s.logs[0].msg) != "hello\n" {
		t.Fatalf("decoded %+v %v", s, err)
	}
}

func TestDecodeRejectsGarbage(t *testing.T) {
	if _, err := decodeStatus([]byte{0x0a, 0xff}); err == nil {
		t.Fatal("truncated message accepted")
	}
}
