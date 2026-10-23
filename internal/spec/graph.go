package spec

import "slices"

// DependencyOrder returns service names so that every service comes after
// the services it depends on, keeping source order where there is a choice.
// If depends_on has a cycle, order is nil and cycle lists it, starting and
// ending with the same service. Unknown dependencies are ignored here; Validate
// reports them.
func DependencyOrder(s Spec) (order []string, cycle []string) {
	deps := map[string][]string{}
	for _, svc := range s.Services {
		for _, d := range svc.DependsOn {
			if _, ok := s.Service(d.Name); ok {
				deps[svc.Name] = append(deps[svc.Name], d.Name)
			}
		}
	}

	const (
		unvisited = iota
		visiting
		done
	)
	state := map[string]int{}
	var stack []string
	var visit func(name string) bool
	visit = func(name string) bool {
		switch state[name] {
		case done:
			return true
		case visiting:
			start := slices.Index(stack, name)
			cycle = append(slices.Clone(stack[start:]), name)
			return false
		}
		state[name] = visiting
		stack = append(stack, name)
		for _, d := range deps[name] {
			if !visit(d) {
				return false
			}
		}
		stack = stack[:len(stack)-1]
		state[name] = done
		order = append(order, name)
		return true
	}
	for _, svc := range s.Services {
		if !visit(svc.Name) {
			return nil, cycle
		}
	}
	return order, nil
}
