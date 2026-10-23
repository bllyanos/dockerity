// Package imagegc removes images of old builds. Selection is pure; the
// collector gathers what to keep and removes the rest.
package imagegc

import (
	"sort"
	"strings"
)

// Image is a local Docker image.
type Image struct {
	ID   string
	Tags []string
}

// Keep says what must survive a collection.
type Keep struct {
	// Tags of recent builds and of the last succeeded deployments.
	Tags map[string]bool
	// InUse are image references (tags or IDs) of existing containers.
	InUse map[string]bool
}

// managedPrefix starts every image tag Dockerity builds (domain.ImageTag).
const managedPrefix = "dkt/"

// Select returns the tags to remove: those of images built by Dockerity that
// are neither kept nor used by a container. An image with any tag outside
// dkt/ is never selected, and neither is an untagged one.
func Select(images []Image, keep Keep) []string {
	var out []string
	for _, img := range images {
		if !managed(img) || keep.InUse[img.ID] {
			continue
		}
		if anyTag(img.Tags, keep.Tags) || anyTag(img.Tags, keep.InUse) {
			continue
		}
		out = append(out, img.Tags...)
	}
	sort.Strings(out)
	return out
}

func managed(img Image) bool {
	if len(img.Tags) == 0 {
		return false
	}
	for _, t := range img.Tags {
		if !strings.HasPrefix(t, managedPrefix) {
			return false
		}
	}
	return true
}

func anyTag(tags []string, set map[string]bool) bool {
	for _, t := range tags {
		if set[t] {
			return true
		}
	}
	return false
}
