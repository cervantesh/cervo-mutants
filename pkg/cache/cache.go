package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"gitea.cervbox.synology.me/CervoSoft/cervo-mutant/pkg/engine"
)

type Store struct {
	Path string
}

func Key(parts ...string) string {
	hash := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(hash[:])
}

func (s Store) Get(ctx context.Context, key string) (engine.CachedResult, bool, error) {
	data, err := os.ReadFile(filepath.Join(s.Path, key+".json"))
	if os.IsNotExist(err) {
		return engine.CachedResult{}, false, nil
	}
	if err != nil {
		return engine.CachedResult{}, false, err
	}
	var cached engine.CachedResult
	if err := json.Unmarshal(data, &cached); err != nil {
		return engine.CachedResult{}, false, err
	}
	return cached, true, nil
}

func (s Store) Put(ctx context.Context, result engine.MutantResult) error {
	if err := os.MkdirAll(s.Path, 0o755); err != nil {
		return err
	}
	key := Key(result.Mutant.Fingerprint, strings.Join(result.TestCommand, " "))
	data, err := json.MarshalIndent(engine.CachedResult{Key: key, Result: result}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(s.Path, key+".json"), data, 0o644)
}
