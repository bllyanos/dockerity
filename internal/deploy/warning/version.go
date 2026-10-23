// Package warning explains a deployment plan's downtime before it runs
// (docs/design.md section 10). It is pure.
package warning

import (
	"strconv"
	"strings"
)

// Tag returns the tag of an image reference, ignoring a registry port and a
// digest. An image without a tag has tag "latest".
func Tag(image string) string {
	ref := image
	if i := strings.IndexByte(ref, '@'); i >= 0 {
		ref = ref[:i]
	}
	slash := strings.LastIndexByte(ref, '/')
	if colon := strings.LastIndexByte(ref, ':'); colon > slash {
		return ref[colon+1:]
	}
	return "latest"
}

// Major returns the leading major version of a tag such as "8.4", "v2.1.0" or
// "16-alpine", or false when the tag does not start with a number.
func Major(tag string) (int, bool) {
	tag = strings.TrimPrefix(tag, "v")
	end := 0
	for end < len(tag) && tag[end] >= '0' && tag[end] <= '9' {
		end++
	}
	if end == 0 {
		return 0, false
	}
	n, err := strconv.Atoi(tag[:end])
	return n, err == nil
}

// MajorChange reports whether two image references have different major versions.
func MajorChange(oldImage, newImage string) bool {
	a, okA := Major(Tag(oldImage))
	b, okB := Major(Tag(newImage))
	return okA && okB && a != b
}
