package config

import (
	"errors"
	"fmt"
	"os"

	"tmeet/internal/core/keychain"
)

// MigrateLegacyAccounts migrates accounts stored in the pre-multi-account storage layout
// (single shared master key + flat per-account ciphertext) into the new per-account folder
// layout with independent master keys, and registers them in the plaintext account registry.
//
// It is idempotent: already-migrated accounts (present in the new layout or the registry) are
// skipped. Failures are logged to stderr and do not block startup, since a partial migration
// still leaves the old data intact (migration never deletes legacy files).
func MigrateLegacyAccounts() {
	openIds := keychain.LegacyListAccounts()
	if len(openIds) == 0 {
		return
	}

	meta, err := loadMeta()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		fmt.Fprintf(os.Stderr, "warning: load meta for migration failed: %v\n", err)
		return
	}
	if meta == nil {
		meta = &AppMeta{}
	}

	for _, openId := range openIds {
		// Skip accounts already in the registry (a prior run migrated them).
		if findAccountByOpenId(meta.Accounts, openId) != nil {
			_ = keychain.MigrateLegacyAccount(openId) // ensure data layout migrated too
			continue
		}
		if err := keychain.MigrateLegacyAccount(openId); err != nil {
			fmt.Fprintf(os.Stderr, "warning: migrate account %s failed: %v\n", openId, err)
			continue
		}
		meta.Accounts = append(meta.Accounts, AccountEntry{Name: openId, OpenId: openId})
	}

	// Point the active account at the migrated account when the registry was empty.
	if meta.ActiveName == "" {
		if meta.ActiveOpenId != "" {
			if e := findAccountByOpenId(meta.Accounts, meta.ActiveOpenId); e != nil {
				meta.ActiveName = e.Name
			}
		}
		if meta.ActiveName == "" && len(meta.Accounts) > 0 {
			meta.ActiveName = meta.Accounts[0].Name
		}
	}

	if err := saveMeta(meta); err != nil {
		fmt.Fprintf(os.Stderr, "warning: save migrated meta failed: %v\n", err)
	}
	ResetCache()
}

// findAccountByOpenId returns the registry entry matching the given OpenId, or nil if not found.
func findAccountByOpenId(accounts []AccountEntry, openId string) *AccountEntry {
	for i := range accounts {
		if accounts[i].OpenId == openId {
			return &accounts[i]
		}
	}
	return nil
}
