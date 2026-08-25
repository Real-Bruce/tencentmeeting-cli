package keychain

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// 使用当前平台真实实现（darwin Keychain / Linux 文件 master key / Windows 注册表+DPAPI），
// 数据目录隔离到 t.TempDir()，避免污染用户环境。
func TestIntegrationSetGetRemove(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("TMEET_CLI_DATA_DIR", dataDir)

	kc := New()
	account := "integration-cli-sdk"
	payload := `{"secret":"x","access_token":"y"}`

	t.Cleanup(func() {
		_ = kc.Remove(ServiceName, account)
	})

	if err := kc.Set(ServiceName, account, payload); err != nil {
		t.Fatalf("Set: %v", err)
	}

	got, err := kc.Get(ServiceName, account)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != payload {
		t.Fatalf("Get() = %q, want %q", got, payload)
	}

	if err := kc.Remove(ServiceName, account); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	_, err = kc.Get(ServiceName, account)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("after Remove, Get err = %v, want ErrNotFound", err)
	}
}

func TestIntegrationConcurrentSetSameAccount(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("TMEET_CLI_DATA_DIR", dataDir)

	kc := New()
	account := "integration-concurrent"
	t.Cleanup(func() { _ = kc.Remove(ServiceName, account) })

	const n = 8
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		go func(v int) {
			errs <- kc.Set(ServiceName, account, fmt.Sprintf(`{"i":%d}`, v))
		}(i)
	}
	for i := 0; i < n; i++ {
		if err := <-errs; err != nil {
			t.Errorf("Set goroutine: %v", err)
		}
	}

	_, err := kc.Get(ServiceName, account)
	if err != nil {
		t.Fatalf("Get after concurrent Set: %v", err)
	}
}

func TestIntegrationEncFilesUnderDataDirUnixLike(t *testing.T) {
	if testing.Short() {
		t.Skip("short mode: skip filesystem layout check")
	}

	dataDir := t.TempDir()
	t.Setenv("TMEET_CLI_DATA_DIR", dataDir)

	kc := New()
	account := "integration-fs-layout"
	t.Cleanup(func() { _ = kc.Remove(ServiceName, account) })

	if err := kc.Set(ServiceName, account, `{"k":"v"}`); err != nil {
		t.Fatal(err)
	}

	// New per-account folder layout: <dataDir>/<account>/{master.key, data.enc}
	encPath := filepath.Join(dataDir, account, "data.enc")
	if _, err := os.Stat(encPath); err != nil {
		t.Fatalf("expected data.enc at %s: %v", encPath, err)
	}
	mkPath := filepath.Join(dataDir, account, "master.key")
	if _, err := os.Stat(mkPath); err != nil {
		t.Fatalf("expected master.key at %s: %v", mkPath, err)
	}

	raw, err := os.ReadFile(encPath)
	if err != nil {
		t.Fatalf("ReadFile data.enc: %v", err)
	}
	if bytes.Contains(raw, []byte(`{"k":"v"}`)) {
		t.Fatal("data.enc file must not contain plaintext; data should be encrypted")
	}
}

// TestIntegrationPerAccountMasterKeyIsolation verifies that each account gets its own master
// key folder and that one account's ciphertext cannot be swapped into another account's folder.
func TestIntegrationPerAccountMasterKeyIsolation(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("TMEET_CLI_DATA_DIR", dataDir)

	kc := New()
	accA := "iso-account-a"
	accB := "iso-account-b"
	t.Cleanup(func() {
		_ = kc.Remove(ServiceName, accA)
		_ = kc.Remove(ServiceName, accB)
	})

	if err := kc.Set(ServiceName, accA, `{"who":"A"}`); err != nil {
		t.Fatal(err)
	}
	if err := kc.Set(ServiceName, accB, `{"who":"B"}`); err != nil {
		t.Fatal(err)
	}

	// Each account has its own master.key file.
	mkA := filepath.Join(dataDir, accA, "master.key")
	mkB := filepath.Join(dataDir, accB, "master.key")
	aBytes, _ := os.ReadFile(mkA)
	bBytes, _ := os.ReadFile(mkB)
	if bytes.Equal(aBytes, bBytes) {
		t.Fatal("per-account master keys should differ between accounts")
	}

	// Swap account A's data.enc into account B's folder (cross-account replacement attack).
	encA := filepath.Join(dataDir, accA, "data.enc")
	encB := filepath.Join(dataDir, accB, "data.enc")
	aEnc, _ := os.ReadFile(encA)
	_ = atomicWriteFile(encB, aEnc)

	_, err := kc.Get(ServiceName, accB)
	if err == nil {
		t.Fatal("Get should fail when account A's ciphertext is placed in account B's folder (master key + AAD mismatch)")
	}
}

func TestIntegrationLoadExistingMasterKey(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("TMEET_CLI_DATA_DIR", dataDir)

	account := "reload-test"
	payload := `{"token":"persist-across-instances"}`

	kc1 := New()
	if err := kc1.Set(ServiceName, account, payload); err != nil {
		t.Fatalf("Set with kc1: %v", err)
	}

	// A fresh instance must reload the per-account master key from disk and decrypt the data.
	kc2 := New()
	got, err := kc2.Get(ServiceName, account)
	if err != nil {
		t.Fatalf("Get with kc2 (reload): %v", err)
	}
	if got != payload {
		t.Fatalf("kc2.Get() = %q, want %q", got, payload)
	}

	t.Cleanup(func() { _ = kc2.Remove(ServiceName, account) })
}

func TestIntegrationMasterKeyWrongSize(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows master key is DPAPI-protected; raw wrong-size bytes fail via DPAPI, tested elsewhere")
	}

	dataDir := t.TempDir()
	t.Setenv("TMEET_CLI_DATA_DIR", dataDir)

	account := "bad-masterkey"
	mkPath := masterKeyPath(dataDir, account)
	if err := os.MkdirAll(filepath.Dir(mkPath), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mkPath, []byte("too-short-key"), 0600); err != nil {
		t.Fatal(err)
	}

	kc := New()
	_, err := kc.Get(ServiceName, account)
	if err == nil {
		t.Fatal("Get should fail when master key file has wrong size")
	}
}

func TestIntegrationMasterKeyFilePermission(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix permission check not applicable on Windows")
	}

	dataDir := t.TempDir()
	t.Setenv("TMEET_CLI_DATA_DIR", dataDir)

	kc := New()
	account := "perm-check"
	t.Cleanup(func() { _ = kc.Remove(ServiceName, account) })

	if err := kc.Set(ServiceName, account, `{"k":"v"}`); err != nil {
		t.Fatal(err)
	}

	mkPath := masterKeyPath(dataDir, account)
	info, err := os.Stat(mkPath)
	if err != nil {
		t.Fatalf("Stat master key: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("master key permission = %o, want 0600", perm)
	}
}
