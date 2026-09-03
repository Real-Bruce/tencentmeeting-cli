//go:build !darwin && !windows

// keychain_other.go provides the Linux/Unix master-key protection hook for the unified file-based
// keychain. The per-account master key is stored as raw 32 bytes at <dataDir>/<account>/master.key
// (permission 0600; the data directory is 0700 via atomicWriteFile, SEC-001/SEC-002). On Linux a
// unified Keychain mechanism has low coverage, so a 0600 file is the industry-standard fallback
// (Feishu lark-cli, DingTalk dws, GitHub CLI all do this).

package keychain

// protectMasterKey returns the raw master key bytes unchanged on Linux/Unix.
func protectMasterKey(_ string, raw []byte) ([]byte, error) {
	return raw, nil
}

// unprotectMasterKey returns the on-disk bytes unchanged on Linux/Unix.
func unprotectMasterKey(_ string, protected []byte) ([]byte, error) {
	return protected, nil
}

// evictMasterKeyCache is a no-op on Linux/Unix: the only master-key cache is the
// fileKeychain.keys sync.Map, which fileKeychain.evictCachedKey clears directly.
func evictMasterKeyCache(_ string) {}
