// legacy.go migrates accounts stored in the pre-multi-account layout into the new per-account
// folder layout.
//
// Old layout (single-account releases):
//   - Linux/macOS: one shared master key at <dataDir>/master.key (raw bytes on Linux, base64 in
//     system Keychain on macOS) + flat <dataDir>/<account>.enc ciphertext files.
//   - Windows: DPAPI-protected shared master key + per-account ciphertext stored as registry
//     values under HKCU\Software\TmeetCli\keychain.
//
// New layout: <dataDir>/<account>/{master.key, data.enc} with an independent master key per
// account. Migration decrypts each old account with the old shared master key, then re-encrypts
// it via the new keychain (which generates a fresh per-account master key). Old files are left
// in place so a failed migration never destroys data.
//
// This file holds platform-agnostic orchestration; platform-specific discovery and reading live
// in legacy_windows.go / legacy_linux.go / legacy_darwin.go.

package keychain

import (
	"fmt"
	"os"
)

// LegacyListAccounts returns the OpenIds of accounts that still exist in the old layout.
// Used by the config layer to know which accounts need registry entries after migration.
func LegacyListAccounts() []string {
	return legacyListOldAccounts()
}

// MigrateLegacyAccount migrates a single account from the old layout to the new per-account
// folder layout. It is a no-op (returns nil) when the account is already in the new layout or
// has no old-layout data. Errors are returned for real I/O/crypto failures.
func MigrateLegacyAccount(openId string) error {
	dataDir := GetDataDir()

	// Already in new layout: clean up any lingering old-layout ciphertext for this account
	// (left behind by an earlier migration that ran before cleanup existed) so a future logout
	// is not resurrected by re-migration, then return.
	if _, err := os.Stat(encFilePath(dataDir, openId)); err == nil {
		if err := legacyRemoveAccount(openId); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to clean up legacy data for %s: %v\n", openId, err)
		}
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat new-layout data file: %w", err)
	}

	plaintext, found, err := legacyReadAccount(openId)
	if err != nil {
		return fmt.Errorf("read legacy account %s: %w", openId, err)
	}
	if !found {
		return nil
	}

	kc := New()
	if err := kc.Set(ServiceName, openId, plaintext); err != nil {
		return fmt.Errorf("write migrated account %s: %w", openId, err)
	}

	// Remove the old-layout per-account ciphertext so a subsequent logout is not resurrected
	// by re-migration on the next startup. The old shared master key is intentionally left in
	// place (harmless without any ciphertext, and shared across accounts).
	if err := legacyRemoveAccount(openId); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to clean up legacy data for %s: %v\n", openId, err)
	}
	return nil
}
