package test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/yliu7949/KouShare-dl/user"
)

func TestImportAndLoadTokens(t *testing.T) {
	credentialPath := isolatedCredentialPath(t)

	var imported user.User
	if err := imported.ImportTokens("access", "refresh", time.Hour); err != nil {
		t.Fatal(err)
	}
	var loaded user.User
	loaded.LoadToken()
	if loaded.LoginState != 1 || loaded.Token != "access" || loaded.RefreshToken != "refresh" {
		t.Fatalf("loaded = %#v", loaded)
	}
	info, err := os.Stat(credentialPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0077 != 0 {
		t.Fatalf("credential permissions = %o, want no group/other access", info.Mode().Perm())
	}
}

func TestLoadExpiredLegacyToken(t *testing.T) {
	credentialPath := isolatedCredentialPath(t)
	if err := os.WriteFile(credentialPath, []byte("legacy 1"), 0600); err != nil {
		t.Fatal(err)
	}
	var loaded user.User
	loaded.LoadToken()
	if loaded.LoginState != -1 {
		t.Fatalf("LoginState = %d, want -1", loaded.LoginState)
	}
}

func TestImportTokensFromHAR(t *testing.T) {
	isolatedCredentialPath(t)
	directory := t.TempDir()
	harPath := filepath.Join(directory, "login.har")
	har := `{"log":{"entries":[{"request":{"method":"POST","url":"https://api-core.koushare.com/iam/userLogin/phoneLogin"},"response":{"content":{"text":"{\"code\":200000,\"data\":{\"data\":{\"access_token\":\"har-access\",\"refresh_token\":\"har-refresh\",\"expire_in\":3600000,\"refresh_expire_in\":7200000}}}"}}}]}}`
	if err := os.WriteFile(harPath, []byte(har), 0600); err != nil {
		t.Fatal(err)
	}
	var imported user.User
	if err := imported.ImportTokensFromHAR(harPath); err != nil {
		t.Fatal(err)
	}
	if imported.Token != "har-access" || imported.RefreshToken != "har-refresh" || imported.LoginState != 1 {
		t.Fatalf("imported = %#v", imported)
	}
}

func TestLoginWithPhoneRejectsInvalidNumbersBeforeNetworkAccess(t *testing.T) {
	for _, phone := range []string{"12345", "8613800138000", "+8613800138000", "138 0013 8000", "phone"} {
		var account user.User
		err := account.LoginWithPhone(context.Background(), phone, "", nil, nil)
		if err == nil || !strings.Contains(err.Error(), "手机号码格式不正确") {
			t.Errorf("LoginWithPhone(%q) error = %v", phone, err)
		}
	}
	for _, phone := range []string{"13800138000", "13912345678"} {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		var account user.User
		err := account.LoginWithPhone(ctx, phone, "", nil, nil)
		if err == nil || strings.Contains(err.Error(), "手机号码格式不正确") {
			t.Errorf("valid phone %q was rejected as malformed: %v", phone, err)
		}
	}
}

func isolatedCredentialPath(t *testing.T) string {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	filename := ".ks.token"
	if runtime.GOOS == "windows" {
		filename = "ks.token"
	}
	path := filepath.Join(filepath.Dir(executable), filename)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			t.Errorf("remove test credential: %v", err)
		}
	})
	return path
}
