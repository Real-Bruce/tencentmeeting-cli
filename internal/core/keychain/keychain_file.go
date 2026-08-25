// keychain_file.go is the unified file-based Keychain implementation used on every platform.
//
// Storage layout (per account / OpenId), achieving per-account master-key isolation:
//
//	<dataDir>/<account>/
//	    master.key   // 32-byte master key (Windows: DPAPI-encrypted; Linux/macOS: raw bytes)
//	    data.enc     // AES-256-GCM ciphertext (AAD = account) of the business JSON
//
// The master key is generated once per account and never rotates; token refreshes only
// rewrite data.enc. The OS-level protection of the master key bytes is delegated to the
// platform hooks protectMasterKey / unprotectMasterKey (see masterkey_*.go).

package keychain

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// fileKeychain is the cross-platform file-based Keychain implementation.
// Per-account master keys are cached in keys to avoid re-reading master.key on every op.
type fileKeychain struct {
	keys sync.Map // account -> []byte (cached raw master key)
}

func newPlatformKeychain() KeychainAccess {
	return &fileKeychain{}
}

// getMasterKey returns the (cached) raw master key for the account, loading or generating it on first use.
func (k *fileKeychain) getMasterKey(account string) ([]byte, error) {
	if v, ok := k.keys.Load(account); ok {
		return v.([]byte), nil
	}
	raw, err := k.loadOrCreateMasterKey(account)
	if err != nil {
		return nil, err
	}
	k.keys.Store(account, raw)
	return raw, nil
}

// loadOrCreateMasterKey reads the per-account master.key; generates and persists a new one if absent.
//
// The on-disk bytes are the platform-protected form (DPAPI ciphertext on Windows, raw on Unix);
// unprotectMasterKey/protectMasterKey handle the conversion. SEC-010 (filecheck) still applies to
// the master key file to reject symlinks / foreign-owned files.
func (k *fileKeychain) loadOrCreateMasterKey(account string) ([]byte, error) {
	dataDir := GetDataDir()
	mkPath := masterKeyPath(dataDir, account)

	protected, err := readEncFile(mkPath)
	if err == nil {
		raw, err := unprotectMasterKey(account, protected)
		if err != nil {
			return nil, fmt.Errorf("failed to unprotect master key: %w", err)
		}
		if len(raw) != masterKeySize {
			return nil, fmt.Errorf("invalid master key length: expected %d bytes, got %d bytes", masterKeySize, len(raw))
		}
		return raw, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return nil, fmt.Errorf("failed to read master key: %w", err)
	}

	// First use for this account: generate a fresh independent master key.
	raw, err := generateMasterKey()
	if err != nil {
		return nil, err
	}
	protected, err = protectMasterKey(account, raw)
	if err != nil {
		return nil, fmt.Errorf("failed to protect master key: %w", err)
	}
	if err = atomicWriteFile(mkPath, protected); err != nil {
		return nil, fmt.Errorf("failed to write master key: %w", err)
	}
	return raw, nil
}

func (k *fileKeychain) Get(service, account string) (string, error) {
	masterKey, err := k.getMasterKey(account)
	if err != nil {
		return "", fmt.Errorf("failed to load master key: %w", err)
	}

	filePath := encFilePath(GetDataDir(), account)
	cipherData, err := readEncFile(filePath)
	if err != nil {
		return "", err
	}

	// SEC-009: account as AAD binds ciphertext to account, preventing cross-account .enc replacement.
	aad := []byte(account)
	plaintext, needsMigration, err := decryptWithAADFallback(masterKey, cipherData, aad)
	if err != nil {
		return "", err
	}
	if needsMigration {
		if newCipher, encErr := encrypt(masterKey, plaintext, aad); encErr == nil {
			if wErr := atomicWriteFile(filePath, newCipher); wErr != nil {
				fmt.Fprintf(os.Stderr, "warning: AAD migration write failed (account=%s): %v\n", account, wErr)
			}
		}
	}
	defer zeroBytes(plaintext)

	return string(plaintext), nil
}

func (k *fileKeychain) Set(service, account, data string) error {
	masterKey, err := k.getMasterKey(account)
	if err != nil {
		return fmt.Errorf("failed to load master key: %w", err)
	}

	cipherData, err := encrypt(masterKey, []byte(data), []byte(account)) // SEC-009: account as AAD
	if err != nil {
		return err
	}

	filePath := encFilePath(GetDataDir(), account)
	return atomicWriteFile(filePath, cipherData)
}

func (k *fileKeychain) Remove(service, account string) error {
	dataDir := GetDataDir()
	mkPath := masterKeyPath(dataDir, account)
	encPath := encFilePath(dataDir, account)

	mkErr := os.Remove(mkPath)
	encErr := os.Remove(encPath)

	// Both missing → nothing to remove.
	if errors.Is(mkErr, os.ErrNotExist) && errors.Is(encErr, os.ErrNotExist) {
		return ErrNotFound
	}
	// Best-effort cleanup of the now-empty account directory (ignore errors; may still contain
	// leftover temp/lock files).
	_ = os.Remove(filepath.Join(dataDir, account))
	k.keys.Delete(account)
	return nil
}
