package plans

func EffectiveTier(base string, trial bool) string {
	if trial {
		return "trial-" + base
	}
	return base
}

func CanArchive(projects int, hardLimit int) bool {
	return projects <= hardLimit
}
