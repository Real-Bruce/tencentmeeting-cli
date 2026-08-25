package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"sync"

	"tmeet/internal/core/keychain"
	"tmeet/internal/exception"
)

// kc is the Keychain access instance, supports injection for testing.
var (
	kc   keychain.KeychainAccess
	kcMu sync.Mutex
)

// SetKeychain injects a custom Keychain implementation.
//
// When not called, GetUserConfig and other functions will automatically use the default platform implementation (keychain.New()).
// Primary use: inject keychain.NewMockKeychain() in unit tests to avoid real system calls.
//
// Usage:
//
//	config.SetKeychain(keychain.NewMockKeychain())
func SetKeychain(k keychain.KeychainAccess) {
	kcMu.Lock()
	defer kcMu.Unlock()
	kc = k
}

// getKeychain returns the Keychain instance (lazy initialization, thread-safe).
func getKeychain() keychain.KeychainAccess {
	kcMu.Lock()
	defer kcMu.Unlock()
	if kc == nil {
		kc = keychain.New()
	}
	return kc
}

// Only letters, digits, and underscores are allowed.
var openIdPattern = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

// validateOpenId validates the OpenId format (whitelist mode) to prevent path traversal and filesystem anomalies.
// Also rejects values that conflict with keychain internal reserved names to avoid overwriting the master key.
func validateOpenId(openId string) error {
	if openId == "" {
		return fmt.Errorf("OpenId cannot be empty")
	}
	if !openIdPattern.MatchString(openId) {
		return fmt.Errorf("invalid OpenId format (only letters, digits, underscores allowed): %q", openId)
	}
	if openId == keychain.MasterKeyAccount {
		return fmt.Errorf("OpenId cannot use reserved name: %q", openId)
	}
	return nil
}

// accountNamePattern allows letters, digits, underscores and hyphens (e.g. token-qtj).
var accountNamePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// validateAccountName validates a user-chosen account alias.
// Aliases live in plaintext config.json (never used as a filesystem path), so hyphens are permitted.
func validateAccountName(name string) error {
	if name == "" {
		return fmt.Errorf("account name cannot be empty")
	}
	if len(name) > 64 {
		return fmt.Errorf("account name too long (max 64 chars): %q", name)
	}
	if !accountNamePattern.MatchString(name) {
		return fmt.Errorf("invalid account name (only letters, digits, underscores, hyphens allowed): %q", name)
	}
	return nil
}

// findAccount returns the registry entry matching the given name, or nil if not found.
func findAccount(accounts []AccountEntry, name string) *AccountEntry {
	for i := range accounts {
		if accounts[i].Name == name {
			return &accounts[i]
		}
	}
	return nil
}

// activeOpenId resolves the OpenId of the currently active account.
//
// Resolution order: ActiveName (looked up against the Accounts registry) → legacy ActiveOpenId.
// This keeps older single-account config.json files working after the multi-account upgrade.
func activeOpenId(meta *AppMeta) string {
	if meta == nil {
		return ""
	}
	if meta.ActiveName != "" {
		if e := findAccount(meta.Accounts, meta.ActiveName); e != nil {
			return e.OpenId
		}
		// ActiveName not in registry (stale meta); fall back to legacy field.
	}
	return meta.ActiveOpenId
}

// GetUserConfig retrieves the user configuration for the currently active account.
//
// Internally decrypts and reads from the encrypted <open_id>/data.enc file; callers do not need
// to know the encryption details. Returns (nil, nil) if not configured (config.json does not exist)
// or not logged in (encrypted entry does not exist). Results are cached in memory; multiple calls
// will not repeat decryption.
//
// Note: userConfig cache is not concurrency-safe; this is fine for CLI single-threaded scenarios.
// If concurrent use is needed, additional locking is required to protect userConfig.
func GetUserConfig() (*UserConfig, error) {
	if userConfig != nil {
		return userConfig, nil
	}

	meta, err := loadMeta()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, exception.GetUserConfigUnknownError.With("failed to load app metadata: %v", err)
	}
	openId := activeOpenId(meta)
	if openId == "" {
		return nil, nil
	}

	if err = validateOpenId(openId); err != nil {
		return nil, exception.GetUserConfigUnknownError.With("invalid OpenId in app metadata: %v", err)
	}

	data, err := getKeychain().Get(keychain.ServiceName, openId)
	if err != nil {
		if errors.Is(err, keychain.ErrNotFound) {
			return nil, nil
		}
		return nil, exception.GetUserConfigUnknownError.With("failed to read encrypted config: %v", err)
	}
	if data == "" {
		return nil, nil
	}

	cfg := &UserConfig{}
	if err = json.Unmarshal([]byte(data), cfg); err != nil {
		return nil, exception.ParseUserConfigError.With("failed to parse user config: %v", err)
	}

	// Authoritative alias comes from the plaintext registry, not the encrypted blob.
	if meta.ActiveName != "" {
		cfg.Name = meta.ActiveName
	} else if cfg.Name == "" {
		cfg.Name = openId
	}

	userConfig = cfg
	return userConfig, nil
}

// SaveUserConfig saves the user configuration (encrypted write to <open_id>/data.enc).
//
// cfg.Name is the user-chosen account alias; when empty it defaults to cfg.OpenId (backward
// compatible with single-account callers). The alias is registered in plaintext config.json
// (AccountEntry) alongside the active-account pointer, which is set to this account.
//
// cfg.OpenId must not be empty. An alias already bound to a different OpenId is rejected.
func SaveUserConfig(config *UserConfig) error {
	if config == nil {
		return exception.InvalidArgsError.With("config cannot be nil")
	}
	if err := validateOpenId(config.OpenId); err != nil {
		return exception.InvalidArgsError.With("invalid OpenId: %v", err)
	}

	name := config.Name
	if name == "" {
		name = config.OpenId
	}
	if err := validateAccountName(name); err != nil {
		return exception.InvalidArgsError.With("invalid account name: %v", err)
	}

	data, err := json.Marshal(config)
	if err != nil {
		return exception.InitializeFailedError.With("failed to serialize user config: %v", err)
	}

	if err = getKeychain().Set(keychain.ServiceName, config.OpenId, string(data)); err != nil {
		return exception.InitializeFailedError.With("failed to save encrypted config: %v", err)
	}

	// Load existing meta ( tolerate missing file = first account).
	meta, err := loadMeta()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return exception.InitializeFailedError.With("failed to load app metadata: %v", err)
	}
	if meta == nil {
		meta = &AppMeta{}
	}

	// Reject alias collision: same name bound to a different OpenId.
	for _, e := range meta.Accounts {
		if e.Name == name && e.OpenId != config.OpenId {
			return exception.AccountNameConflictError
		}
	}

	// Upsert by OpenId (a re-login of the same account updates its alias/sdk).
	upserted := false
	for i := range meta.Accounts {
		if meta.Accounts[i].OpenId == config.OpenId {
			meta.Accounts[i].Name = name
			meta.Accounts[i].SdkId = config.SdkId
			upserted = true
			break
		}
	}
	if !upserted {
		meta.Accounts = append(meta.Accounts, AccountEntry{
			Name:   name,
			OpenId: config.OpenId,
			SdkId:  config.SdkId,
		})
	}

	meta.ActiveName = name
	meta.ActiveOpenId = config.OpenId // legacy compat

	if err = saveMeta(meta); err != nil {
		return exception.InitializeFailedError.With("failed to update app metadata (encrypted data saved, please retry): %v", err)
	}

	config.Name = name
	userConfig = config
	return nil
}

// ClearUserConfig clears the configuration for the currently active account.
//
// Deletes the corresponding <open_id>/ encrypted entry and removes the account from the registry.
// If the active account was removed, the active pointer moves to the first remaining account
// (or is cleared when none remain). Other accounts' data is untouched.
func ClearUserConfig() (retErr error) {
	meta, err := loadMeta()
	if err != nil || meta == nil {
		ResetCache()
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return exception.LogoutFailedError.With("failed to load app metadata: %v", err)
		}
		return nil
	}

	openId := activeOpenId(meta)
	if openId == "" {
		ResetCache()
		return nil
	}

	activeName := meta.ActiveName
	if activeName == "" {
		activeName = openId
	}

	defer func() {
		ResetCache()
	}()

	if err := clearAccountEntry(meta, activeName); err != nil {
		return err
	}

	if len(meta.Accounts) > 0 {
		// Reassign active to the first remaining account.
		meta.ActiveName = meta.Accounts[0].Name
		meta.ActiveOpenId = meta.Accounts[0].OpenId
		if err := saveMeta(meta); err != nil {
			return exception.LogoutFailedError.With("failed to update app metadata: %v", err)
		}
	} else {
		// No accounts left: remove config.json.
		if err := clearMeta(); err != nil {
			return exception.LogoutFailedError.With("failed to clear app metadata: %v", err)
		}
	}

	// Best-effort removal of the encrypted entry; tolerate already-missing.
	if err := getKeychain().Remove(keychain.ServiceName, openId); err != nil && !errors.Is(err, keychain.ErrNotFound) {
		return exception.LogoutFailedError.With("failed to delete encrypted config: %v", err)
	}
	return nil
}

// ClearAccount clears the configuration for a named account.
//
// Removes the account from the registry and deletes its encrypted entry. If the named account
// was active, the active pointer moves to the first remaining account (or is cleared).
// Returns AccountNotFoundError when the name is not registered.
func ClearAccount(name string) error {
	if err := validateAccountName(name); err != nil {
		return exception.InvalidArgsError.With("invalid account name: %v", err)
	}

	meta, err := loadMeta()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return exception.AccountNotFoundError
		}
		return exception.LogoutFailedError.With("failed to load app metadata: %v", err)
	}
	if meta == nil {
		return exception.AccountNotFoundError
	}

	entry := findAccount(meta.Accounts, name)
	if entry == nil {
		return exception.AccountNotFoundError
	}
	openId := entry.OpenId

	wasActive := meta.ActiveName == name

	if err := clearAccountEntry(meta, name); err != nil {
		return err
	}

	if len(meta.Accounts) > 0 {
		if wasActive {
			meta.ActiveName = meta.Accounts[0].Name
			meta.ActiveOpenId = meta.Accounts[0].OpenId
		}
		if err := saveMeta(meta); err != nil {
			return exception.LogoutFailedError.With("failed to update app metadata: %v", err)
		}
	} else {
		if err := clearMeta(); err != nil {
			return exception.LogoutFailedError.With("failed to clear app metadata: %v", err)
		}
	}

	if err := getKeychain().Remove(keychain.ServiceName, openId); err != nil && !errors.Is(err, keychain.ErrNotFound) {
		return exception.LogoutFailedError.With("failed to delete encrypted config: %v", err)
	}
	ResetCache()
	return nil
}

// clearAccountEntry removes the named account from meta.Accounts in place.
// It does NOT touch the encrypted entry or persist meta; callers handle those.
func clearAccountEntry(meta *AppMeta, name string) error {
	idx := -1
	for i, e := range meta.Accounts {
		if e.Name == name {
			idx = i
			break
		}
	}
	if idx < 0 {
		// Legacy single-account meta without a registry entry: nothing to slice.
		return nil
	}
	meta.Accounts = append(meta.Accounts[:idx], meta.Accounts[idx+1:]...)
	return nil
}

// GetAccountConfig returns the user configuration for a named account without touching the
// active-account pointer or the in-memory cache. Returns (nil, nil) when the account is not
// registered or its encrypted entry is missing. Primarily used by the login guard to decide
// whether a named account is already authenticated.
func GetAccountConfig(name string) (*UserConfig, error) {
	if err := validateAccountName(name); err != nil {
		return nil, nil
	}
	meta, err := loadMeta()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, exception.GetUserConfigUnknownError.With("failed to load app metadata: %v", err)
	}
	if meta == nil {
		return nil, nil
	}
	entry := findAccount(meta.Accounts, name)
	if entry == nil {
		return nil, nil
	}
	if err = validateOpenId(entry.OpenId); err != nil {
		return nil, nil
	}
	data, err := getKeychain().Get(keychain.ServiceName, entry.OpenId)
	if err != nil {
		if errors.Is(err, keychain.ErrNotFound) {
			return nil, nil
		}
		return nil, exception.GetUserConfigUnknownError.With("failed to read encrypted config: %v", err)
	}
	if data == "" {
		return nil, nil
	}
	cfg := &UserConfig{}
	if err = json.Unmarshal([]byte(data), cfg); err != nil {
		return nil, exception.ParseUserConfigError.With("failed to parse user config: %v", err)
	}
	cfg.Name = name
	return cfg, nil
}

// ListAccounts returns the registered accounts (alias, openId, sdkId).
// Returns nil when no config.json exists or no accounts are registered.
func ListAccounts() ([]AccountEntry, error) {
	meta, err := loadMeta()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, exception.GetUserConfigUnknownError.With("failed to load app metadata: %v", err)
	}
	if meta == nil {
		return nil, nil
	}
	return meta.Accounts, nil
}

// GetActiveAccountName returns the alias of the currently active account, or "" if none.
// Falls back to the legacy ActiveOpenId for older config.json files.
func GetActiveAccountName() string {
	meta, err := loadMeta()
	if err != nil || meta == nil {
		return ""
	}
	if meta.ActiveName != "" {
		return meta.ActiveName
	}
	return meta.ActiveOpenId
}

// SwitchAccount sets the active account by alias.
// The named account must exist in the registry. The in-memory cache is invalidated so the
// next GetUserConfig reads the newly active account's encrypted entry.
func SwitchAccount(name string) error {
	if err := validateAccountName(name); err != nil {
		return exception.InvalidArgsError.With("invalid account name: %v", err)
	}

	meta, err := loadMeta()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return exception.AccountNotFoundError
		}
		return exception.GetUserConfigUnknownError.With("failed to load app metadata: %v", err)
	}
	if meta == nil {
		return exception.AccountNotFoundError
	}

	entry := findAccount(meta.Accounts, name)
	if entry == nil {
		return exception.AccountNotFoundError
	}

	meta.ActiveName = entry.Name
	meta.ActiveOpenId = entry.OpenId
	if err := saveMeta(meta); err != nil {
		return exception.InitializeFailedError.With("failed to save app metadata: %v", err)
	}
	ResetCache()
	return nil
}

// ResetCache 清除内存中的配置缓存，下次 GetUserConfig 会重新从加密文件读取。
// 主要用于测试场景或需要强制刷新配置时。
func ResetCache() {
	userConfig = nil
}
