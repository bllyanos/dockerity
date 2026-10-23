package spec

import "time"

// EffectivePhase is the job phase, defaulting to pre-deploy.
func (s Service) EffectivePhase() Phase {
	if s.Phase == "" {
		return PhasePreDeploy
	}
	return s.Phase
}

// EffectiveStrategy is the update strategy actually used: deployments default
// to blue-green, stateful services always recreate, jobs have none.
func (s Service) EffectiveStrategy() Strategy {
	switch s.Kind {
	case KindDeployment:
		if s.Strategy == "" {
			return StrategyBlueGreen
		}
		return s.Strategy
	case KindStateful:
		return StrategyRecreate
	default:
		return ""
	}
}

// RoutePath is the routed path prefix, defaulting to "/".
func (h HTTP) RoutePath() string {
	if h.Path == "" {
		return "/"
	}
	return h.Path
}

// Default timings for replacing containers.
const (
	DefaultDrain           = 5 * time.Second
	DefaultStopGracePeriod = 30 * time.Second
	// MaxTiming bounds drain and stop_grace_period.
	MaxTiming = 10 * time.Minute
)

// EffectiveDrain is the drain period, defaulting to DefaultDrain.
func (s Service) EffectiveDrain() time.Duration {
	if s.Drain == 0 {
		return DefaultDrain
	}
	return s.Drain
}

// EffectiveStopGracePeriod is the SIGTERM grace period, defaulting to DefaultStopGracePeriod.
func (s Service) EffectiveStopGracePeriod() time.Duration {
	if s.StopGracePeriod == 0 {
		return DefaultStopGracePeriod
	}
	return s.StopGracePeriod
}
