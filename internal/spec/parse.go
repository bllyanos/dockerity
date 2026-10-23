package spec

import (
	"regexp"
	"strconv"

	"go.yaml.in/yaml/v3"
)

// Parse decodes an app spec. It reports every syntax, type and unknown-field
// problem with its position as Problems. Semantic rules are checked by Validate.
func Parse(src []byte) (Spec, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(src, &doc); err != nil {
		return Spec{}, syntaxProblem(err)
	}
	if len(doc.Content) == 0 {
		return Spec{}, Problems{{Pos: Pos{Line: 1, Column: 1}, Message: "spec is empty"}}
	}
	d := &decoder{}
	s := d.spec(doc.Content[0])
	return s, d.probs.sorted()
}

var yamlLine = regexp.MustCompile(`line (\d+): (.*)`)

func syntaxProblem(err error) Problems {
	if m := yamlLine.FindStringSubmatch(err.Error()); m != nil {
		line, _ := strconv.Atoi(m[1])
		return Problems{{Pos: Pos{Line: line, Column: 1}, Message: "invalid YAML: " + m[2]}}
	}
	return Problems{{Message: "invalid YAML: " + err.Error()}}
}

func (d *decoder) spec(root *yaml.Node) Spec {
	var s Spec
	fields, ok := d.mapping(root, "the spec")
	if !ok {
		return s
	}
	var hasServices bool
	for _, f := range fields {
		switch f.key {
		case "name":
			s.Name, s.NamePos = d.str(f.value, "name"), f.pos
		case "services":
			hasServices = true
			s.Services = d.services(f.value)
		default:
			d.unknown(f, "the spec")
		}
	}
	if !hasServices {
		d.probs.add(posOf(resolve(root)), "the spec must define services")
	}
	return s
}

func (d *decoder) services(n *yaml.Node) []Service {
	fields, ok := d.mapping(n, "services")
	if !ok {
		return nil
	}
	out := make([]Service, 0, len(fields))
	for _, f := range fields {
		out = append(out, d.service(f))
	}
	return out
}

func (d *decoder) service(entry field) Service {
	s := Service{Name: entry.key, Pos: entry.pos, FieldPos: map[string]Pos{}}
	where := "service " + entry.key
	fields, ok := d.mapping(entry.value, where)
	if !ok {
		return s
	}
	for _, f := range fields {
		s.FieldPos[f.key] = f.pos
		what := where + "." + f.key
		switch f.key {
		case "kind":
			s.Kind = Kind(d.str(f.value, what))
		case "phase":
			s.Phase = Phase(d.str(f.value, what))
		case "strategy":
			s.Strategy = Strategy(d.str(f.value, what))
		case "image":
			s.Image = d.str(f.value, what)
		case "build":
			s.Build = d.build(f.value, what)
		case "command":
			s.Command = d.strList(f.value, what)
		case "env":
			s.Env = d.envVars(f.value, what)
		case "secrets":
			s.Secrets = d.secrets(f.value, what)
		case "volumes":
			s.Volumes = d.volumes(f.value, what)
		case "depends_on":
			s.DependsOn = d.refs(f.value, what)
		case "http":
			s.HTTP = d.http(f.value, what)
		case "healthcheck":
			s.Healthcheck = d.healthcheck(f.value, what)
		case "resources":
			s.Resources = d.resources(f.value, what)
		default:
			delete(s.FieldPos, f.key)
			d.unknown(f, where)
		}
	}
	return s
}

// build accepts a context path shorthand (`build: .`) or a mapping.
func (d *decoder) build(n *yaml.Node, what string) *Build {
	if r := resolve(n); r.Kind == yaml.ScalarNode {
		return &Build{Context: d.str(r, what)}
	}
	b := &Build{}
	fields, ok := d.mapping(n, what)
	if !ok {
		return b
	}
	for _, f := range fields {
		switch f.key {
		case "context":
			b.Context = d.str(f.value, what+".context")
		case "dockerfile":
			b.Dockerfile = d.str(f.value, what+".dockerfile")
		case "target":
			b.Target = d.str(f.value, what+".target")
		case "args":
			b.Args = d.envVars(f.value, what+".args")
		default:
			d.unknown(f, what)
		}
	}
	return b
}

// secrets accepts names, or {name, file} mappings for file injection.
func (d *decoder) secrets(n *yaml.Node, what string) []SecretRef {
	n = resolve(n)
	if n.Kind != yaml.SequenceNode {
		d.probs.add(posOf(n), "%s must be a list, got %s", what, kindName(n))
		return nil
	}
	out := make([]SecretRef, 0, len(n.Content))
	for _, item := range n.Content {
		item = resolve(item)
		if item.Kind == yaml.ScalarNode {
			out = append(out, SecretRef{Name: d.str(item, what+" item"), Pos: posOf(item)})
			continue
		}
		ref := SecretRef{Pos: posOf(item)}
		fields, ok := d.mapping(item, what+" item")
		if !ok {
			continue
		}
		for _, f := range fields {
			switch f.key {
			case "name":
				ref.Name = d.str(f.value, what+".name")
			case "file":
				ref.File = d.str(f.value, what+".file")
			default:
				d.unknown(f, what+" item")
			}
		}
		out = append(out, ref)
	}
	return out
}

func (d *decoder) volumes(n *yaml.Node, what string) []Volume {
	fields, ok := d.mapping(n, what)
	if !ok {
		return nil
	}
	out := make([]Volume, 0, len(fields))
	for _, f := range fields {
		out = append(out, Volume{Name: f.key, Path: d.str(f.value, what+"."+f.key), Pos: f.pos})
	}
	return out
}

func (d *decoder) refs(n *yaml.Node, what string) []Ref {
	n = resolve(n)
	if n.Kind != yaml.SequenceNode {
		d.probs.add(posOf(n), "%s must be a list of service names, got %s", what, kindName(n))
		return nil
	}
	out := make([]Ref, 0, len(n.Content))
	for _, item := range n.Content {
		out = append(out, Ref{Name: d.str(item, what+" item"), Pos: posOf(resolve(item))})
	}
	return out
}

func (d *decoder) http(n *yaml.Node, what string) *HTTP {
	h := &HTTP{Pos: posOf(resolve(n))}
	fields, ok := d.mapping(n, what)
	if !ok {
		return h
	}
	for _, f := range fields {
		switch f.key {
		case "port":
			h.Port = d.integer(f.value, what+".port")
		case "path":
			h.Path = d.str(f.value, what+".path")
		default:
			d.unknown(f, what)
		}
	}
	return h
}

func (d *decoder) healthcheck(n *yaml.Node, what string) *Healthcheck {
	h := &Healthcheck{Pos: posOf(resolve(n))}
	fields, ok := d.mapping(n, what)
	if !ok {
		return h
	}
	for _, f := range fields {
		w := what + "." + f.key
		switch f.key {
		case "cmd":
			h.Cmd = d.strList(f.value, w)
		case "http":
			h.HTTP = d.str(f.value, w)
		case "interval":
			h.Interval = d.duration(f.value, w)
		case "timeout":
			h.Timeout = d.duration(f.value, w)
		case "start_period":
			h.StartPeriod = d.duration(f.value, w)
		case "retries":
			h.Retries = d.integer(f.value, w)
		default:
			d.unknown(f, what)
		}
	}
	return h
}

func (d *decoder) resources(n *yaml.Node, what string) *Resources {
	r := &Resources{}
	fields, ok := d.mapping(n, what)
	if !ok {
		return r
	}
	for _, f := range fields {
		switch f.key {
		case "memory":
			r.Memory, r.MemoryPos = d.str(f.value, what+".memory"), posOf(resolve(f.value))
		case "cpus":
			r.CPUs, r.CPUsPos = d.str(f.value, what+".cpus"), posOf(resolve(f.value))
		default:
			d.unknown(f, what)
		}
	}
	return r
}
