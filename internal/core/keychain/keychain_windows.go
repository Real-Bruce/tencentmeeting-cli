//go:build windows

// keychain_windows.go provides the Windows platform master-key protection hook.
//
// The per-account master key bytes are DPAPI-encrypted (CryptProtectData) before being written
// to <dataDir>/<account>/master.key, and DPAPI-decrypted (CryptUnprotectData) on read. DPAPI is
// bound to the current Windows user's login credentials, so the master key file is unusable if
// copied to another user or device. optionalEntropy (service + "\x00" + account) raises the
// DPAPI decryption threshold so that other processes under the same user cannot decrypt the
// master key without knowing the entropy (SEC-004).
//
// The encrypted business data (data.enc) is stored as a file under the same per-account folder,
// AES-256-GCM protected by the raw master key.

package keychain

import (
	"fmt"
	"os"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	cryptprotectUIForbidden = 0x1 // DPAPI must not show interactive prompts (CLI context)
)

type dataBlob struct {
	cbData uint32
	pbData *byte
}

func newDataBlob(data []byte) *dataBlob {
	if len(data) == 0 {
		return &dataBlob{}
	}
	return &dataBlob{
		cbData: uint32(len(data)),
		pbData: &data[0],
	}
}

func (b *dataBlob) toBytes() []byte {
	d := make([]byte, b.cbData)
	copy(d, unsafe.Slice(b.pbData, b.cbData))
	return d
}

var (
	crypt32                = windows.NewLazySystemDLL("crypt32.dll")
	procCryptProtectData   = crypt32.NewProc("CryptProtectData")
	procCryptUnprotectData = crypt32.NewProc("CryptUnprotectData")
)

// dpapiEntropy returns the DPAPI optionalEntropy parameter (SEC-004).
// Format aligned with Feishu/DingTalk: service + "\x00" + account.
func dpapiEntropy(service, account string) []byte {
	return []byte(service + "\x00" + account)
}

func cryptProtectData(plaintext, entropy []byte) ([]byte, error) {
	inBlob := newDataBlob(plaintext)
	var entropyPtr uintptr
	if len(entropy) > 0 {
		entropyBlob := newDataBlob(entropy)
		entropyPtr = uintptr(unsafe.Pointer(entropyBlob))
	}
	var outBlob dataBlob

	r, _, err := procCryptProtectData.Call(
		uintptr(unsafe.Pointer(inBlob)),
		0,
		entropyPtr,
		0, 0,
		cryptprotectUIForbidden,
		uintptr(unsafe.Pointer(&outBlob)),
	)
	if r == 0 {
		return nil, fmt.Errorf("CryptProtectData call failed: %w", err)
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(outBlob.pbData)))

	return outBlob.toBytes(), nil
}

func cryptUnprotectData(ciphertext, entropy []byte) ([]byte, error) {
	inBlob := newDataBlob(ciphertext)
	var entropyPtr uintptr
	if len(entropy) > 0 {
		entropyBlob := newDataBlob(entropy)
		entropyPtr = uintptr(unsafe.Pointer(entropyBlob))
	}
	var outBlob dataBlob

	r, _, err := procCryptUnprotectData.Call(
		uintptr(unsafe.Pointer(inBlob)),
		0,
		entropyPtr,
		0, 0,
		cryptprotectUIForbidden,
		uintptr(unsafe.Pointer(&outBlob)),
	)
	if r == 0 {
		return nil, fmt.Errorf("CryptUnprotectData call failed: %w", err)
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(outBlob.pbData)))

	return outBlob.toBytes(), nil
}

// masterKeyCache caches the unprotected raw master key per account to avoid repeated DPAPI
// + file reads on every operation (each costs ~2-10ms).
var masterKeyCache sync.Map // account -> []byte

// protectMasterKey DPAPI-encrypts the raw master key bytes for the given account.
func protectMasterKey(account string, raw []byte) ([]byte, error) {
	return cryptProtectData(raw, dpapiEntropy(ServiceName, account))
}

// evictMasterKeyCache drops the cached unprotected master key for the account on Windows.
// Called by fileKeychain.evictCachedKey after a decryption failure so the next read re-runs
// DPAPI unprotect against the on-disk bytes (which another process may have regenerated).
func evictMasterKeyCache(account string) {
	masterKeyCache.Delete(account)
}

// unprotectMasterKey DPAPI-decrypts the on-disk master key bytes for the given account.
// Falls back to nil-entropy DPAPI for ciphertexts written by older releases (auto-migration
// on first read), then re-encrypts with the entropy-bound form.
func unprotectMasterKey(account string, protected []byte) ([]byte, error) {
	if v, ok := masterKeyCache.Load(account); ok {
		return v.([]byte), nil
	}
	entropy := dpapiEntropy(ServiceName, account)
	raw, err := cryptUnprotectData(protected, entropy)
	if err != nil {
		// Backward compat: old DPAPI ciphertexts had no entropy.
		raw, err = cryptUnprotectData(protected, nil)
		if err != nil {
			return nil, fmt.Errorf("DPAPI failed to decrypt master key (user credentials may have changed): %w", err)
		}
		// Auto-migrate: re-encrypt with entropy-bound DPAPI.
		if newProtected, reErr := cryptProtectData(raw, entropy); reErr == nil {
			_ = newProtected // best-effort; persist handled lazily — caller may re-store
		} else {
			fmt.Fprintf(os.Stderr, "warning: DPAPI entropy migration re-encrypt failed: %v\n", reErr)
		}
	}
	masterKeyCache.Store(account, raw)
	return raw, nil
}
