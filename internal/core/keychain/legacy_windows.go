//go:build windows

// legacy_windows.go reads accounts stored in the pre-multi-account Windows layout
// (DPAPI-protected shared master key + per-account ciphertext in the registry) so they can be
// migrated to the new per-account folder layout. Read-only; never writes the registry.

package keychain

import (
	"encoding/base64"
	"errors"
	"fmt"

	"golang.org/x/sys/windows/registry"
)

// legacyRegistryKeyPath is the registry path used by the old single-account release.
const legacyRegistryKeyPath = `Software\TmeetCli\keychain`

func legacyListOldAccounts() []string {
	regKey, err := registry.OpenKey(registry.CURRENT_USER, legacyRegistryKeyPath, registry.READ)
	if err != nil {
		return nil
	}
	defer regKey.Close()

	names, err := regKey.ReadValueNames(-1)
	if err != nil {
		return nil
	}
	var out []string
	for _, n := range names {
		if n == MasterKeyAccount {
			continue
		}
		out = append(out, n)
	}
	return out
}

// legacyLoadMasterKey reads and DPAPI-decrypts the old shared master key from the registry.
// Returns found=false when the key does not exist (no legacy data on this machine).
func legacyLoadMasterKey() (masterKey []byte, found bool, err error) {
	regKey, err := registry.OpenKey(registry.CURRENT_USER, legacyRegistryKeyPath, registry.READ)
	if err != nil {
		if errors.Is(err, registry.ErrNotExist) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("open legacy registry key: %w", err)
	}
	defer regKey.Close()

	encoded, _, err := regKey.GetStringValue(MasterKeyAccount)
	if err != nil {
		if errors.Is(err, registry.ErrNotExist) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("read legacy master key: %w", err)
	}

	protected, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, false, fmt.Errorf("decode legacy master key base64: %w", err)
	}

	entropy := dpapiEntropy(ServiceName, MasterKeyAccount)
	raw, err := cryptUnprotectData(protected, entropy)
	if err != nil {
		// Old DPAPI ciphertexts had no entropy.
		raw, err = cryptUnprotectData(protected, nil)
		if err != nil {
			return nil, false, fmt.Errorf("DPAPI decrypt legacy master key: %w", err)
		}
	}
	if len(raw) != masterKeySize {
		return nil, false, fmt.Errorf("invalid legacy master key length: expected %d, got %d", masterKeySize, len(raw))
	}
	return raw, true, nil
}

func legacyReadAccount(openId string) (plaintext string, found bool, err error) {
	masterKey, mkFound, err := legacyLoadMasterKey()
	if err != nil {
		return "", false, err
	}
	if !mkFound {
		return "", false, nil
	}

	regKey, err := registry.OpenKey(registry.CURRENT_USER, legacyRegistryKeyPath, registry.READ)
	if err != nil {
		return "", false, nil // registry gone → nothing to migrate
	}
	defer regKey.Close()

	encoded, _, err := regKey.GetStringValue(openId)
	if err != nil {
		if errors.Is(err, registry.ErrNotExist) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("read legacy account %s: %w", openId, err)
	}

	cipherData, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", false, fmt.Errorf("decode legacy account %s base64: %w", openId, err)
	}

	aad := []byte(openId)
	plain, _, err := decryptWithAADFallback(masterKey, cipherData, aad)
	if err != nil {
		return "", false, fmt.Errorf("decrypt legacy account %s: %w", openId, err)
	}
	return string(plain), true, nil
}

// legacyRemoveAccount deletes the old-layout registry value for the account.
func legacyRemoveAccount(openId string) error {
	regKey, err := registry.OpenKey(registry.CURRENT_USER, legacyRegistryKeyPath, registry.ALL_ACCESS)
	if err != nil {
		if errors.Is(err, registry.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("open legacy registry key for cleanup: %w", err)
	}
	defer regKey.Close()
	if err := regKey.DeleteValue(openId); err != nil && !errors.Is(err, registry.ErrNotExist) {
		return fmt.Errorf("delete legacy registry value %s: %w", openId, err)
	}
	return nil
}
