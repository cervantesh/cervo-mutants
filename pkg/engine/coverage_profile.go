package engine

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	internalcoverage "github.com/cervantesh/cervo-mutants/pkg/internal/coverageprofile"
)

func (s *runSession) coverageMentions(mutant Mutant) bool {
	lineCovered, _ := s.coverageSignal(mutant)
	return lineCovered
}

func (s *runSession) coverageSignal(mutant Mutant) (lineCovered bool, fileCovered bool) {
	profile := s.coverageProfilePath(mutant.Module)
	data, err := os.ReadFile(profile)
	if err != nil {
		return false, false
	}
	rel, _ := filepath.Rel(mutant.Module, mutant.File)
	rel = filepath.ToSlash(rel)
	base := filepath.Base(mutant.File)
	return coverageDataSignal(string(data), rel, base, mutant.Line)
}

func coverageDataSignal(data, rel, base string, mutantLine int) (lineCovered bool, fileCovered bool) {
	return internalcoverage.DataSignal(data, rel, base, mutantLine)
}

func (s *runSession) coverageProfilePath(moduleDir string) string {
	profile := s.engine.cfg.Selection.CoverageProfile
	if filepath.IsAbs(profile) {
		return profile
	}
	baseDir := moduleDir
	if s.coverageBaseDir != "" {
		baseDir = s.coverageBaseDir
	}
	return filepath.Join(baseDir, profile)
}

func fallbackCoverageMentions(data, rel, base string) bool {
	return internalcoverage.FallbackMentions(data, rel, base)
}

func parseCoverageProfileLine(line string) (string, int, int, int, bool) {
	return internalcoverage.ParseLine(line)
}

func parseCoverageLineNumber(value string) (int, bool) {
	dot := strings.Index(value, ".")
	if dot < 0 {
		return 0, false
	}
	line, err := strconv.Atoi(value[:dot])
	return line, err == nil
}

func coverageFileMatches(profileFile, rel, base string) bool {
	return internalcoverage.FileMatches(profileFile, rel, base)
}
