package spec

import (
	"strconv"
	"time"

	"go.yaml.in/yaml/v3"
)

// decoder turns yaml.Nodes into values, collecting problems instead of
// stopping at the first one.
type decoder struct {
	probs Problems
}

func posOf(n *yaml.Node) Pos { return Pos{Line: n.Line, Column: n.Column} }

func kindName(n *yaml.Node) string {
	switch n.Kind {
	case yaml.MappingNode:
		return "a mapping"
	case yaml.SequenceNode:
		return "a list"
	case yaml.ScalarNode:
		if n.Tag == "!!null" {
			return "null"
		}
		return "a value"
	default:
		return "an unsupported node"
	}
}

// resolve follows aliases so anchors work like plain YAML.
func resolve(n *yaml.Node) *yaml.Node {
	for n != nil && n.Kind == yaml.AliasNode {
		n = n.Alias
	}
	return n
}

// field is one key/value pair of a mapping.
type field struct {
	key   string
	pos   Pos
	value *yaml.Node
}

// mapping returns the pairs of a mapping node, reporting non-string and
// duplicate keys. what describes the mapping for messages.
func (d *decoder) mapping(n *yaml.Node, what string) ([]field, bool) {
	n = resolve(n)
	if n.Kind != yaml.MappingNode {
		d.probs.add(posOf(n), "%s must be a mapping, got %s", what, kindName(n))
		return nil, false
	}
	var out []field
	seen := map[string]bool{}
	for i := 0; i+1 < len(n.Content); i += 2 {
		k, v := resolve(n.Content[i]), n.Content[i+1]
		if k.Kind != yaml.ScalarNode || k.Value == "" {
			d.probs.add(posOf(k), "%s keys must be non-empty strings", what)
			continue
		}
		if seen[k.Value] {
			d.probs.add(posOf(k), "duplicate key %q in %s", k.Value, what)
			continue
		}
		seen[k.Value] = true
		out = append(out, field{key: k.Value, pos: posOf(k), value: v})
	}
	return out, true
}

func (d *decoder) str(n *yaml.Node, what string) string {
	n = resolve(n)
	if n.Kind != yaml.ScalarNode || n.Tag == "!!null" {
		d.probs.add(posOf(n), "%s must be a string, got %s", what, kindName(n))
		return ""
	}
	return n.Value
}

func (d *decoder) strList(n *yaml.Node, what string) []string {
	n = resolve(n)
	if n.Kind != yaml.SequenceNode {
		d.probs.add(posOf(n), "%s must be a list of strings, got %s", what, kindName(n))
		return nil
	}
	out := make([]string, 0, len(n.Content))
	for _, item := range n.Content {
		out = append(out, d.str(item, what+" item"))
	}
	return out
}

func (d *decoder) integer(n *yaml.Node, what string) int {
	n = resolve(n)
	v, err := strconv.Atoi(n.Value)
	if n.Kind != yaml.ScalarNode || err != nil {
		d.probs.add(posOf(n), "%s must be an integer", what)
		return 0
	}
	return v
}

func (d *decoder) duration(n *yaml.Node, what string) time.Duration {
	n = resolve(n)
	v, err := time.ParseDuration(n.Value)
	if n.Kind != yaml.ScalarNode || err != nil || v <= 0 {
		d.probs.add(posOf(n), "%s must be a positive duration such as 10s or 1m", what)
		return 0
	}
	return v
}

// envVars decodes a name: value mapping. Values may be any scalar and are
// kept as written.
func (d *decoder) envVars(n *yaml.Node, what string) []EnvVar {
	fields, ok := d.mapping(n, what)
	if !ok {
		return nil
	}
	out := make([]EnvVar, 0, len(fields))
	for _, f := range fields {
		v := resolve(f.value)
		if v.Kind != yaml.ScalarNode {
			d.probs.add(posOf(v), "%s.%s must be a string, got %s", what, f.key, kindName(v))
			continue
		}
		value := v.Value
		if v.Tag == "!!null" {
			value = ""
		}
		out = append(out, EnvVar{Name: f.key, Value: value, Pos: f.pos})
	}
	return out
}

// unknown reports a field that is not part of the spec, with a hint for
// compose fields that Dockerity deliberately does not support.
func (d *decoder) unknown(f field, where string) {
	if hint, ok := unsupportedHints[f.key]; ok {
		d.probs.add(f.pos, "%q is not allowed in %s: %s", f.key, where, hint)
		return
	}
	d.probs.add(f.pos, "unknown field %q in %s", f.key, where)
}

var unsupportedHints = map[string]string{
	"container_name": "Dockerity owns all container names",
	"ports":          "host port bindings are not allowed; use http to route through Traefik",
	"network_mode":   "Dockerity owns networking",
	"networks":       "Dockerity owns networking",
	"restart":        "Dockerity manages restarts",
	"deploy":         "use strategy and resources instead",
}
