package mathx

func ClampPort(port int) int {
	if port < 1 {
		return 1
	}
	if port > 65535 {
		return 65535
	}
	return port
}

func RetryBudget(attempts int, includeWarmup bool) int {
	if attempts <= 0 {
		return 0
	}
	if includeWarmup {
		return attempts + 1
	}
	return attempts
}
