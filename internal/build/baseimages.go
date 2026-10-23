package build

import (
	"bufio"
	"bytes"
	"os"
	"regexp"
	"strings"
)

var (
	argLine  = regexp.MustCompile(`(?i)^ARG\s+([A-Za-z_][A-Za-z0-9_]*)(?:=(.*))?$`)
	fromLine = regexp.MustCompile(`(?i)^FROM\s+(?:--platform=\S+\s+)?(\S+)(?:\s+AS\s+(\S+))?\s*$`)
	varRef   = regexp.MustCompile(`\$\{?([A-Za-z_][A-Za-z0-9_]*)\}?`)
)

// BaseImages lists the external images a Dockerfile builds FROM, so they can
// be pulled before the build. ARGs declared before the first FROM are
// substituted from args or their defaults. Stage references, "scratch", and
// images that still contain unresolved variables are skipped.
func BaseImages(dockerfile []byte, args map[string]string) []string {
	var (
		out    []string
		seen   = map[string]bool{}
		stages = map[string]bool{}
		global = map[string]string{}
		inBody bool
	)
	for _, line := range logicalLines(dockerfile) {
		if m := argLine.FindStringSubmatch(line); m != nil && !inBody {
			v, ok := args[m[1]]
			if !ok {
				v = strings.Trim(m[2], `"'`)
			}
			global[m[1]] = v
			continue
		}
		m := fromLine.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		inBody = true
		image := varRef.ReplaceAllStringFunc(m[1], func(ref string) string {
			name := varRef.FindStringSubmatch(ref)[1]
			if v, ok := global[name]; ok && v != "" {
				return v
			}
			return ref
		})
		if m[2] != "" {
			stages[strings.ToLower(m[2])] = true
		}
		if strings.Contains(image, "$") || strings.EqualFold(image, "scratch") || stages[strings.ToLower(image)] || seen[image] {
			continue
		}
		seen[image] = true
		out = append(out, image)
	}
	return out
}

// BaseImagesFromFile reads a Dockerfile and returns its base images.
func BaseImagesFromFile(path string, args map[string]string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return BaseImages(data, args), nil
}

// logicalLines joins backslash continuations and drops comments and blank lines.
func logicalLines(src []byte) []string {
	var (
		out []string
		cur strings.Builder
	)
	sc := bufio.NewScanner(bytes.NewReader(src))
	sc.Buffer(make([]byte, 64*1024), 1<<20)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if cur.Len() == 0 && (line == "" || strings.HasPrefix(line, "#")) {
			continue
		}
		if strings.HasSuffix(line, `\`) {
			cur.WriteString(strings.TrimSuffix(line, `\`))
			cur.WriteByte(' ')
			continue
		}
		cur.WriteString(line)
		out = append(out, strings.TrimSpace(cur.String()))
		cur.Reset()
	}
	if cur.Len() > 0 {
		out = append(out, strings.TrimSpace(cur.String()))
	}
	return out
}
