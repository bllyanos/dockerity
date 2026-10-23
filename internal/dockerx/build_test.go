package dockerx

import (
	"archive/tar"
	"bytes"
	"context"
	"strings"
	"testing"
	"time"
)

func TestReadBuildStream(t *testing.T) {
	stream := `{"id":"moby.buildkit.trace","aux":"GmEKR3NoYTI1Njo1NjdmZjJlMjU2OWVjNjE0NGM2Nzk1YmY1N2U0MDE2OTliNjVjNDQxNjhjNjJjZWVmNTJjZDIzZTVlYjZkNjNmEgwI2dnr1gYQwMuUyAIYASIGaGVsbG8K"}
{"stream":"legacy line\n"}
{"id":"moby.image.id","aux":{"ID":"sha256:abc"}}
`
	var logs bytes.Buffer
	id, err := readBuildStream(strings.NewReader(stream), &logs)
	if err != nil || id != "sha256:abc" {
		t.Fatalf("got %q %v", id, err)
	}
	if !strings.Contains(logs.String(), "#1 hello\n") || !strings.Contains(logs.String(), "legacy line") {
		t.Fatalf("logs %q", logs.String())
	}
}

func TestReadBuildStreamError(t *testing.T) {
	stream := `{"errorDetail":{"message":"failed to solve: process did not complete"},"error":"short"}`
	var logs bytes.Buffer
	if _, err := readBuildStream(strings.NewReader(stream), &logs); err == nil || !strings.Contains(err.Error(), "failed to solve") {
		t.Fatalf("got %v", err)
	}
	if !strings.Contains(logs.String(), "ERROR: failed to solve") {
		t.Fatalf("error not logged: %q", logs.String())
	}
	if _, err := readBuildStream(strings.NewReader(""), &logs); err == nil {
		t.Fatal("missing image id accepted")
	}
}

func testContext(t *testing.T, files map[string]string) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	for name, body := range files {
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(body))}); err != nil {
			t.Fatal(err)
		}
		_, _ = tw.Write([]byte(body))
	}
	_ = tw.Close()
	return &buf
}

// TestBuildImageWithDocker runs against the local Docker engine when available.
func TestBuildImageWithDocker(t *testing.T) {
	c, err := New()
	if err != nil {
		t.Skip("docker client unavailable:", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	if err := c.Ping(ctx); err != nil {
		t.Skip("docker engine unavailable:", err)
	}
	tag := "dkt-test/build:" + time.Now().Format("150405.000000")
	tag = strings.ReplaceAll(tag, ".", "")
	t.Cleanup(func() { _ = c.RemoveImage(context.Background(), tag) })

	files := map[string]string{
		"Dockerfile": "FROM busybox:1.37\nARG GREETING\nCOPY hello.txt /hello.txt\nRUN echo \"$GREETING\" && cat /hello.txt\n",
		"hello.txt":  "hello from dockerity\n",
	}
	build := func() (string, string) {
		var logs bytes.Buffer
		id, err := c.BuildImage(ctx, BuildRequest{
			Context: testContext(t, files), Dockerfile: "Dockerfile", Tags: []string{tag},
			Args: map[string]string{"GREETING": "hi"}, Labels: map[string]string{"dockerity.test": "true"},
		}, &logs)
		if err != nil {
			t.Fatalf("build: %v\n%s", err, logs.String())
		}
		return id, logs.String()
	}

	first, logs := build()
	if !strings.Contains(logs, "hello from dockerity") {
		t.Fatalf("RUN output missing:\n%s", logs)
	}
	second, logs := build()
	if second != first || !strings.Contains(logs, "CACHED") {
		t.Fatalf("rebuild should hit the cache (ids %s vs %s):\n%s", first, second, logs)
	}
}
