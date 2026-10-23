package spec

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
