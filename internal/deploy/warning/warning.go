package warning

import (
	"time"

	"github.com/bllyanos/dockerity/internal/deploy/planner"
	"github.com/bllyanos/dockerity/internal/domain"
	"github.com/bllyanos/dockerity/internal/spec"
)

// Service describes how one service is affected by a deployment.
type Service struct {
	Name         string
	Strategy     spec.Strategy
	Downtime     bool
	OldImage     string
	NewImage     string
	ImageChanged bool
	MajorChange  bool
	// Backup is true when Dockerity backs up the service's volumes first.
	Backup bool
	// LastBackup maps each volume to its latest backup time (nil when never).
	LastBackup map[string]*time.Time
	// Dependents are services that depend on this one and may misbehave while it is down.
	Dependents []string
}

// Warning summarizes a plan for the confirmation shown before deploying.
type Warning struct {
	Downtime bool
	Services []Service
}

// Input is a previewed plan plus what is known about backups.
type Input struct {
	App     string
	Env     string
	Spec    spec.Spec
	Plan    planner.Plan
	Current map[string]planner.Running
	// LastBackups maps volume names to their latest backup time.
	LastBackups map[string]time.Time
}

// Build lists every replaced service with its downtime, image change, backup
// and dependents.
func Build(in Input) Warning {
	w := Warning{Downtime: in.Plan.Downtime()}
	for _, st := range in.Plan.Steps {
		if st.Kind != planner.StepStateful && st.Kind != planner.StepDeploy {
			continue
		}
		svc, _ := in.Spec.Service(st.Service)
		old := in.Current[st.Service].Image
		s := Service{
			Name: st.Service, Strategy: st.Strategy, Downtime: st.Downtime,
			OldImage: old, NewImage: st.Image, ImageChanged: st.ImageChanged,
			MajorChange: st.ImageChanged && MajorChange(old, st.Image),
			Backup:      st.Kind == planner.StepStateful && st.ImageChanged,
		}
		if st.Kind == planner.StepStateful {
			s.LastBackup = map[string]*time.Time{}
			for _, v := range svc.Volumes {
				name := domain.VolumeName(in.App, in.Env, svc.Name, v.Name)
				if t, ok := in.LastBackups[name]; ok {
					s.LastBackup[name] = &t
				} else {
					s.LastBackup[name] = nil
				}
			}
		}
		if st.Downtime {
			s.Dependents = spec.Dependents(in.Spec, st.Service)
		}
		w.Services = append(w.Services, s)
	}
	return w
}
