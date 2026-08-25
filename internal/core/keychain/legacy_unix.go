//go:build !darwin && !windows

// legacy_unix.go reads accounts stored in the pre-multi-account Linux/Unix layout
// (raw shared master key at <dataDir>/master.key + flat <dataDir>/<account>.enc ciphertext files)
// so they can be migrated to the new per-account folder layout. Read-only.

package keychain

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func legacyListOldAccounts() []string {
	dataDir := GetDataDir()
	entries, err := os.ReadDir(dataDir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".enc") {
			continue
		}
		openId := strings.TrimSuffix(name, ".enc")
		if openId == "" || openId == MasterKeyAccount {
			continue
		}
		out = append(out, openId)
	}
	return out
}

func legacyReadAccount(openId string) (plaintext string, found bool, err error) {
	dataDir := GetDataDir()
	encPath := filepath.Join(dataDir, openId+".enc")

	cipherData, err := readEncFile(encPath)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("read legacy enc file: %w", err)
	}

	raw, err := os.ReadFile(filepath.Join(dataDir, MasterKeyAccount))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", false, nil // no old shared master key → nothing to migrate
		}
		return "", false, fmt.Errorf("read legacy master key: %w", err)
	}
	if len(raw) != masterKeySize {
		return "", false, fmt.Errorf("invalid legacy master key length: expected %d, got %d", masterKeySize, len(raw))
	}

	aad := []byte(openId)
	plain, _, err := decryptWithAADFallback(raw, cipherData, aad)
	if err != nil {
		return "", false, fmt.Errorf("decrypt legacy account %s: %w", openId, err)
	}
	return string(plain), true, nil
}

// legacyRemoveAccount deletes the old-layout flat <account>.enc file.
func legacyRemoveAccount(openId string) error {
	encPath := filepath.Join(GetDataDir(), openId+".enc")
	if err := os.Remove(encPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("delete legacy enc file: %w", err)
	}
	return nil
}
