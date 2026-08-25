//go:build darwin

// keychain_darwin.go provides the macOS master-key protection hook for the unified file-based
// keychain. The per-account master key is stored as raw 32 bytes at <dataDir>/<account>/master.key
// (permission 0600 via the user-private Application Support directory). The previous release kept
// the master key in the system Keychain (go-keyring); multi-account per-OpenId folders require the
// master key to live in the per-account folder, so the new impl stores it as a 0600 file (same
// protection model as Linux). Migration of old Keychain-based accounts is handled in legacy_darwin.go.

package keychain

// protectMasterKey returns the raw master key bytes unchanged (no OS-level wrapping on macOS).
func protectMasterKey(_ string, raw []byte) ([]byte, error) {
	return raw, nil
}

// unprotectMasterKey returns the on-disk bytes unchanged.
func unprotectMasterKey(_ string, protected []byte) ([]byte, error) {
	return protected, nil
}
