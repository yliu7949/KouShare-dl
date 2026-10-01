package user

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

const browserLoginURL = "https://www.koushare.com/account/login"

var loginPhonePattern = regexp.MustCompile(`^1[3-9]\d{9}$`)

// LoginWithBrowser opens a visible browser where the user can complete
// KouShare's normal login. The dedicated profile is deliberately persistent:
// EdgeOne's user-completed verification cookie can be reused instead of
// presenting a fresh bot challenge on every invocation.
func (u *User) LoginWithBrowser(ctx context.Context, executablePath string) error {
	profileDir, err := browserProfileDirectory()
	if err != nil {
		return err
	}
	// Do not use DefaultExecAllocatorOptions here. They include
	// --enable-automation and a collection of testing-oriented feature flags,
	// which make Chrome show the automation banner and can cause unnecessary
	// anti-bot challenges. CDP is used only to observe the final login cookies.
	allocatorOptions := []chromedp.ExecAllocatorOption{
		chromedp.Flag("headless", false),
		chromedp.Flag("start-maximized", true),
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
		chromedp.UserDataDir(profileDir),
	}
	if strings.TrimSpace(executablePath) != "" {
		allocatorOptions = append(allocatorOptions, chromedp.ExecPath(executablePath))
	}

	allocatorContext, cancelAllocator := chromedp.NewExecAllocator(ctx, allocatorOptions...)
	defer cancelAllocator()
	browserContext, cancelBrowser := chromedp.NewContext(allocatorContext)
	defer cancelBrowser()

	if err := chromedp.Run(browserContext, network.Enable(), chromedp.Navigate(browserLoginURL)); err != nil {
		return fmt.Errorf("启动登录浏览器: %w", err)
	}

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		accessToken, refreshToken, validFor, found, err := readBrowserCredentials(browserContext)
		if err != nil {
			if ctx.Err() != nil {
				return browserLoginContextError(ctx.Err())
			}
			return fmt.Errorf("读取网页登录 Cookie: %w", err)
		}
		if found {
			return u.ImportTokens(accessToken, refreshToken, validFor)
		}

		select {
		case <-ctx.Done():
			return browserLoginContextError(ctx.Err())
		case <-ticker.C:
		}
	}
}

func browserProfileDirectory() (string, error) {
	configDirectory, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("查找用户配置目录: %w", err)
	}
	profileDirectory := filepath.Join(configDirectory, "KouShare-dl", "browser-profile")
	if err := os.MkdirAll(profileDirectory, 0700); err != nil {
		return "", fmt.Errorf("创建登录浏览器配置: %w", err)
	}
	return profileDirectory, nil
}

func readBrowserCredentials(ctx context.Context) (string, string, time.Duration, bool, error) {
	var cookies []*network.Cookie
	err := chromedp.Run(ctx, chromedp.ActionFunc(func(actionContext context.Context) error {
		var getErr error
		cookies, getErr = network.GetCookies().WithURLs([]string{
			"https://www.koushare.com/",
			"https://api-core.koushare.com/",
		}).Do(actionContext)
		return getErr
	}))
	if err != nil {
		return "", "", 0, false, err
	}
	return credentialsFromCookies(cookies, time.Now())
}

func credentialsFromCookies(cookies []*network.Cookie, now time.Time) (string, string, time.Duration, bool, error) {
	var accessCookie *network.Cookie
	var refreshToken string
	for _, cookie := range cookies {
		switch cookie.Name {
		case "accessToken":
			accessCookie = cookie
		case "refreshToken":
			refreshToken = normalizeBrowserCookie(cookie.Value)
		}
	}
	if accessCookie == nil {
		return "", "", 0, false, nil
	}
	accessToken := normalizeBrowserCookie(accessCookie.Value)
	if accessToken == "" {
		return "", "", 0, false, errors.New("网站写入了空的 accessToken Cookie")
	}

	validFor := 30 * 24 * time.Hour
	if accessCookie.Expires > 0 {
		validFor = time.Unix(int64(accessCookie.Expires), 0).Sub(now)
		if validFor <= 0 {
			return "", "", 0, false, errors.New("网站写入的 accessToken Cookie 已过期")
		}
	}
	return accessToken, refreshToken, validFor, true, nil
}

func normalizeBrowserCookie(value string) string {
	value = strings.TrimSpace(value)
	if decoded, err := url.PathUnescape(value); err == nil {
		value = decoded
	}
	if strings.HasPrefix(value, `"`) {
		var unquoted string
		if json.Unmarshal([]byte(value), &unquoted) == nil {
			return strings.TrimSpace(unquoted)
		}
	}
	return strings.TrimSpace(value)
}

func browserLoginContextError(err error) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return errors.New("等待网页登录超时；请重试并在浏览器中完成登录")
	}
	return errors.New("网页登录已取消")
}
