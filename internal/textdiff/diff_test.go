package textdiff

import "testing"

func TestUnifiedEqual(t *testing.T) {
	if got := Unified("a", "b", "x\ny\n", "x\ny\n", 3); got != "" {
		t.Fatalf("equal texts: %q", got)
	}
}

func TestUnifiedSingleChange(t *testing.T) {
	a := "1\n2\n3\n4\n5\n6\n7\n8\n9\n"
	b := "1\n2\n3\n4\nfive\n6\n7\n8\n9\n"
	want := "--- v1\n+++ v2\n@@ -2,7 +2,7 @@\n 2\n 3\n 4\n-5\n+five\n 6\n 7\n 8\n"
	if got := Unified("v1", "v2", a, b, 3); got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestUnifiedSeparateHunks(t *testing.T) {
	a := "a\nb\nc\nd\ne\nf\ng\nh\ni\nj\n"
	b := "A\nb\nc\nd\ne\nf\ng\nh\ni\nJ\n"
	want := "--- x\n+++ y\n@@ -1,2 +1,2 @@\n-a\n+A\n b\n@@ -9,2 +9,2 @@\n i\n-j\n+J\n"
	if got := Unified("x", "y", a, b, 1); got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestUnifiedInsertAndDeleteAll(t *testing.T) {
	if got := Unified("x", "y", "", "new\n", 3); got != "--- x\n+++ y\n@@ -0,0 +1 @@\n+new\n" {
		t.Fatalf("insert into empty: %q", got)
	}
	if got := Unified("x", "y", "old\n", "", 3); got != "--- x\n+++ y\n@@ -1 +0,0 @@\n-old\n" {
		t.Fatalf("delete all: %q", got)
	}
}

func TestLinesPrefersLCS(t *testing.T) {
	lines := Lines("a\nb\nc\n", "a\nc\n")
	if len(lines) != 3 || lines[1] != (Line{Delete, "b"}) {
		t.Fatalf("%+v", lines)
	}
}
