package buildlog

import (
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ExpiredNotice replaces the log of a finished build or deployment whose
// log file was deleted by retention.
const ExpiredNotice = "[dockerity] log expired\n"

// File is a stored log.
type File struct {
	ID      int64
	ModTime time.Time
}

// Expired returns the IDs of logs last written before cutoff, ascending.
func Expired(files []File, cutoff time.Time) []int64 {
	var out []int64
	for _, f := range files {
		if f.ModTime.Before(cutoff) {
			out = append(out, f.ID)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// List returns the stored logs. Other files in Dir are ignored.
func (s Store) List() ([]File, error) {
	entries, err := os.ReadDir(s.Dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []File
	for _, e := range entries {
		name, ok := strings.CutSuffix(e.Name(), ".log")
		if !ok || e.IsDir() {
			continue
		}
		id, err := strconv.ParseInt(name, 10, 64)
		if err != nil {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue // removed meanwhile
		}
		out = append(out, File{ID: id, ModTime: info.ModTime()})
	}
	return out, nil
}

// Expire deletes logs last written before cutoff and returns their IDs.
func (s Store) Expire(cutoff time.Time) ([]int64, error) {
	files, err := s.List()
	if err != nil {
		return nil, err
	}
	ids := Expired(files, cutoff)
	for _, id := range ids {
		if err := s.Remove(id); err != nil {
			return nil, err
		}
	}
	return ids, nil
}
