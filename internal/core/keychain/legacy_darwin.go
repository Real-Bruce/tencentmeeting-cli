//go:build darwin

// legacy_darwin.go reads accounts stored in the pre-multi-account macOS layout
// (base64-encoded shared master key in the system Keychain via go-keyring + flat
// <dataDir>/<account>.enc ciphertext files) so they can be migrated to the new per-account
// folder layout. Read-only.

package keychain

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zalando/go-keyring"
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

	encoded, kerr := keyring.Get(ServiceName, MasterKeyAccount)
	if kerr != nil {
		if errors.Is(kerr, keyring.ErrNotFound) {
			return "", false, nil // no old master key in Keychain → nothing to migrate
		}
		return "", false, fmt.Errorf("read legacy master key from keychain: %w", kerr)
	}
	raw, derr := base64.StdEncoding.DecodeString(encoded)
	if derr != nil {
		return "", false, fmt.Errorf("decode legacy master key: %w", derr)
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
