package user

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/yliu7949/KouShare-dl/internal/koushare"
)

func TestImportAndLoadTokens(t *testing.T) {
	originalPath := tokenFileName
	originalUser := u
	t.Cleanup(func() {
		tokenFileName = originalPath
		u = originalUser
	})
	tokenFileName = filepath.Join(t.TempDir(), ".ks.token")

	var imported User
	if err := imported.ImportTokens("access", "refresh", time.Hour); err != nil {
		t.Fatal(err)
	}
	var loaded User
	loaded.LoadToken()
	if loaded.LoginState != 1 || loaded.Token != "access" || loaded.RefreshToken != "refresh" {
		t.Fatalf("loaded = %#v", loaded)
	}
	info, err := os.Stat(tokenFileName)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0077 != 0 {
		t.Fatalf("credential permissions = %o, want no group/other access", info.Mode().Perm())
	}
}

func TestLoadExpiredLegacyToken(t *testing.T) {
	originalPath := tokenFileName
	t.Cleanup(func() { tokenFileName = originalPath })
	tokenFileName = filepath.Join(t.TempDir(), ".ks.token")
	if err := os.WriteFile(tokenFileName, []byte("legacy 1"), 0600); err != nil {
		t.Fatal(err)
	}
	var loaded User
	loaded.LoadToken()
	if loaded.LoginState != -1 {
		t.Fatalf("LoginState = %d, want -1", loaded.LoginState)
	}
}

func TestImportTokensFromHAR(t *testing.T) {
	originalPath := tokenFileName
	originalUser := u
	t.Cleanup(func() {
		tokenFileName = originalPath
		u = originalUser
	})
	directory := t.TempDir()
	tokenFileName = filepath.Join(directory, ".ks.token")
	harPath := filepath.Join(directory, "login.har")
	har := `{"log":{"entries":[{"request":{"method":"POST","url":"https://api-core.koushare.com/iam/userLogin/phoneLogin"},"response":{"content":{"text":"{\"code\":200000,\"data\":{\"data\":{\"access_token\":\"har-access\",\"refresh_token\":\"har-refresh\",\"expire_in\":3600000,\"refresh_expire_in\":7200000}}}"}}}]}}`
	if err := os.WriteFile(harPath, []byte(har), 0600); err != nil {
		t.Fatal(err)
	}
	var imported User
	if err := imported.ImportTokensFromHAR(harPath); err != nil {
		t.Fatal(err)
	}
	if imported.Token != "har-access" || imported.RefreshToken != "har-refresh" || imported.LoginState != 1 {
		t.Fatalf("imported = %#v", imported)
	}
}

func TestNormalizeBrowserCookie(t *testing.T) {
	tests := map[string]string{
		`%22header.payload.signature%22`:  "header.payload.signature",
		`"plain-token"`:                   "plain-token",
		`plain-token`:                     "plain-token",
		`%22token%2Bwith%2Fsymbols%3D%22`: "token+with/symbols=",
	}
	for input, want := range tests {
		if got := normalizeBrowserCookie(input); got != want {
			t.Errorf("normalizeBrowserCookie(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestCredentialsFromCookies(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	cookies := []*network.Cookie{
		{Name: "refreshToken", Value: `%22refresh%22`},
		{Name: "accessToken", Value: `%22access%22`, Expires: float64(now.Add(2 * time.Hour).Unix())},
	}
	access, refresh, validFor, found, err := credentialsFromCookies(cookies, now)
	if err != nil {
		t.Fatal(err)
	}
	if !found || access != "access" || refresh != "refresh" || validFor != 2*time.Hour {
		t.Fatalf("credentials = %q, %q, %v, %v", access, refresh, validFor, found)
	}
}

func TestCredentialsFromCookiesWithoutLogin(t *testing.T) {
	_, _, _, found, err := credentialsFromCookies(nil, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if found {
		t.Fatal("found credentials before login")
	}
}

func TestLoginPhonePattern(t *testing.T) {
	for _, phone := range []string{"13800138000", "13912345678"} {
		if !loginPhonePattern.MatchString(phone) {
			t.Errorf("valid phone %q was rejected", phone)
		}
	}
	for _, phone := range []string{"12345", "8613800138000", "+8613800138000", "138 0013 8000", "phone"} {
		if loginPhonePattern.MatchString(phone) {
			t.Errorf("invalid phone %q was accepted", phone)
		}
	}
}

func TestCaptchaPageHandler(t *testing.T) {
	handler := captchaPageHandler(koushare.CaptchaConfig{
		CaptchaAppID: "app-id", AidEncrypted: "encrypted-value",
	}, "/captcha/nonce", "/state/nonce")
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d", recorder.Code)
	}
	body := recorder.Body.String()
	for _, value := range []string{"app-id", "encrypted-value", "/captcha/nonce", "/state/nonce", "TCaptcha.js", "window.close()"} {
		if !strings.Contains(body, value) {
			t.Errorf("captcha page does not contain %q", value)
		}
	}
	for _, unwanted := range []string{"请完成人机验证", "开始验证", "验证完成", "请返回终端"} {
		if strings.Contains(body, unwanted) {
			t.Errorf("captcha page unexpectedly contains %q", unwanted)
		}
	}
}

func TestMacAppBundle(t *testing.T) {
	path := "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
	if got := macAppBundle(path); got != "/Applications/Google Chrome.app" {
		t.Fatalf("macAppBundle(%q) = %q", path, got)
	}
	if got := macAppBundle("/usr/local/bin/chromium"); got != "" {
		t.Fatalf("macAppBundle for non-app executable = %q", got)
	}
}

func TestCaptchaChromeArgsUseAppWindow(t *testing.T) {
	args := captchaChromeArgs("http://127.0.0.1:1234/", "/tmp/profile")
	joined := strings.Join(args, " ")
	for _, value := range []string{
		"--app=http://127.0.0.1:1234/",
		"--window-size=460,600",
		"--user-data-dir=/tmp/profile",
		"--disable-background-mode",
	} {
		if !strings.Contains(joined, value) {
			t.Errorf("Chrome arguments do not contain %q: %v", value, args)
		}
	}
	for _, arg := range args {
		if arg == "http://127.0.0.1:1234/" {
			t.Fatalf("captcha URL was passed as a normal tab argument: %v", args)
		}
	}
}

func TestCaptchaCallbackHandler(t *testing.T) {
	results := make(chan captchaResult, 1)
	handler := captchaCallbackHandler(results)
	request := httptest.NewRequest(http.MethodPost, "/captcha/nonce", strings.NewReader(`{"ticket":"ticket","randstr":"@rand"}`))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	result := <-results
	if result.Ticket != "ticket" || result.Randstr != "@rand" {
		t.Fatalf("result = %#v", result)
	}
}

func TestAllDigits(t *testing.T) {
	if !allDigits("123456") || allDigits("12345x") || allDigits("") {
		t.Fatal("allDigits validation is incorrect")
	}
}
