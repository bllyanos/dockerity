package build

import (
	"slices"
	"testing"
)

func TestBaseImages(t *testing.T) {
	df := `# syntax=docker/dockerfile:1
ARG GO_VERSION=1.27
ARG BASE="debian:12"
ARG UNSET

FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine AS build
RUN go build ./...

FROM node:22 as web
FROM build AS test
FROM \
  ${BASE}
FROM scratch
FROM golang:${GO_VERSION}-alpine
FROM ${UNSET}
FROM gcr.io/distroless/static-debian12
`
	got := BaseImages([]byte(df), map[string]string{"BASE": "debian:13"})
	want := []string{"golang:1.27-alpine", "node:22", "debian:13", "gcr.io/distroless/static-debian12"}
	if !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestBaseImagesArgAfterFromIsNotGlobal(t *testing.T) {
	df := "FROM alpine:3.20\nARG TAG=1\nFROM busybox:${TAG}\n"
	if got := BaseImages([]byte(df), nil); !slices.Equal(got, []string{"alpine:3.20"}) {
		t.Fatalf("got %v", got)
	}
}
