package engine

import (
	"path/filepath"
	"strings"

	"github.com/cervantesh/cervo-mutants/pkg/config"
)

func (e *Engine) ownershipRoute(moduleDir, pkgPath, filePath string) *OwnershipRoute {
	normalizedPkg := normalizeOwnershipPackage(pkgPath)
	normalizedFile, absoluteFile := normalizeOwnershipFile(moduleDir, filePath)
	for _, rule := range e.cfg.Ownership.Rules {
		if ownershipRuleMatches(rule, normalizedPkg, normalizedFile, absoluteFile) {
			return &OwnershipRoute{
				Owner:   strings.TrimSpace(rule.Owner),
				Team:    strings.TrimSpace(rule.Team),
				Contact: strings.TrimSpace(rule.Contact),
				Rule:    strings.TrimSpace(rule.Name),
			}
		}
	}
	if target := e.cfg.Ownership.Default; ownershipTargetConfigured(target) {
		return &OwnershipRoute{
			Owner:   strings.TrimSpace(target.Owner),
			Team:    strings.TrimSpace(target.Team),
			Contact: strings.TrimSpace(target.Contact),
			Rule:    "default",
		}
	}
	return nil
}

func ownershipRuleMatches(rule config.OwnershipRule, pkgPath, filePath, absoluteFile string) bool {
	if selector := strings.TrimSpace(rule.Package); selector != "" && !ownershipPackageMatches(selector, pkgPath) {
		return false
	}
	if selector := strings.TrimSpace(rule.File); selector != "" && !suppressionFileMatches(selector, filePath) {
		if absoluteFile == "" || !suppressionFileMatches(selector, absoluteFile) {
			return false
		}
	}
	return true
}

func normalizeOwnershipFile(moduleDir, filePath string) (string, string) {
	filePath = strings.TrimSpace(filePath)
	if filePath == "" {
		return "", ""
	}
	absoluteFile := filepath.ToSlash(filepath.Clean(filePath))
	if strings.TrimSpace(moduleDir) != "" {
		if rel, err := filepath.Rel(moduleDir, filePath); err == nil && rel != "." && !strings.HasPrefix(rel, "..") {
			return filepath.ToSlash(rel), absoluteFile
		}
	}
	if strings.HasPrefix(absoluteFile, "./") {
		absoluteFile = strings.TrimPrefix(absoluteFile, "./")
	}
	return absoluteFile, absoluteFile
}

func ownershipPackageMatches(pattern, pkgPath string) bool {
	pattern = normalizeOwnershipPackage(pattern)
	if pattern == "" {
		return false
	}
	if pkgPath == pattern {
		return true
	}
	return globMatch(pattern, pkgPath)
}

func normalizeOwnershipPackage(value string) string {
	value = filepath.ToSlash(strings.TrimSpace(value))
	value = strings.TrimPrefix(value, "./")
	if value == "" {
		return "."
	}
	return value
}

func ownershipTargetConfigured(target config.OwnershipTarget) bool {
	return strings.TrimSpace(target.Owner) != "" || strings.TrimSpace(target.Team) != "" || strings.TrimSpace(target.Contact) != ""
}
