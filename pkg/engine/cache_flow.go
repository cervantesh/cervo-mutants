package engine

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/cervantesh/cervo-mutants/pkg/discover"
)

func (s *runSession) getCached(key string) (MutantResult, bool, error) {
	data, err := os.ReadFile(filepath.Join(s.engine.cfg.Cache.Path, key+".json"))
	if os.IsNotExist(err) {
		return MutantResult{}, false, nil
	}
	if err != nil {
		return MutantResult{}, false, err
	}
	var result MutantResult
	if err := json.Unmarshal(data, &result); err != nil {
		return MutantResult{}, false, err
	}
	return result, true, nil
}

func (s *runSession) putCached(key string, result MutantResult) error {
	if err := os.MkdirAll(s.engine.cfg.Cache.Path, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(s.engine.cfg.Cache.Path, key+".json"), data, 0o644)
}

func (e *Engine) discoverForTest(targets []string) (discover.Result, error) {
	return discover.Discover(targets)
}

func (s *runSession) cacheKey(mutant Mutant, plan TestPlan) (string, error) {
	parts := []string{
		"v2",
		mutant.Fingerprint,
		mutant.File,
		mutant.Package,
		s.engine.cfg.Mutators.Profile,
		s.engine.cfg.Selection.Mode,
		strings.Join(plan.Command, "\x00"),
		runtime.Version(),
	}
	for _, name := range []string{"go.mod", "go.sum"} {
		if digest, err := digestFile(filepath.Join(mutant.Module, name)); err == nil {
			parts = append(parts, name+"="+digest)
		}
	}
	sourceDigest, err := digestFile(mutant.File)
	if err != nil {
		return "", err
	}
	parts = append(parts, "source="+sourceDigest)
	testDigests, err := testDigests(mutant.Module, mutant.Package)
	if err != nil {
		return "", err
	}
	parts = append(parts, testDigests...)
	configData, _ := json.Marshal(s.engine.cfg)
	parts = append(parts, "config="+digestBytes(configData))
	return digestBytes([]byte(strings.Join(parts, "\x00"))), nil
}

func testDigests(moduleDir, pkg string) ([]string, error) {
	dir := moduleDir
	if pkg != "." && strings.HasPrefix(pkg, "./") {
		dir = filepath.Join(moduleDir, filepath.FromSlash(strings.TrimPrefix(pkg, "./")))
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var digests []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		digest, err := digestFile(path)
		if err != nil {
			return nil, err
		}
		digests = append(digests, "test:"+entry.Name()+"="+digest)
	}
	sort.Strings(digests)
	return digests, nil
}

func digestFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func digestBytes(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func (s *runSession) recordTiming(mutantID string, duration time.Duration) {
	if !s.engine.cfg.Selection.UseTimings || s.engine.cfg.Selection.TimingsPath == "" || mutantID == "" {
		return
	}
	s.timingMu.Lock()
	defer s.timingMu.Unlock()
	path := s.engine.cfg.Selection.TimingsPath
	if !filepath.IsAbs(path) {
		path = filepath.Join(".", path)
	}
	timings := map[string]int64{}
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &timings)
	}
	timings[mutantID] = duration.Milliseconds()
	if timings[mutantID] == 0 {
		timings[mutantID] = 1
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	data, err := json.MarshalIndent(timings, "", "  ")
	if err == nil {
		_ = os.WriteFile(path, data, 0o644)
	}
}
