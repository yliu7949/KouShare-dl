package user

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	AccessTokenEnv  = "KOUSHARE_ACCESS_TOKEN"
	RefreshTokenEnv = "KOUSHARE_REFRESH_TOKEN"
)

// User represents locally imported credentials. The current website requires
// an interactive captcha before it sends an SMS. KouShare-dl therefore accepts
// tokens obtained after a normal browser login instead of trying to automate
// or bypass that challenge.
type User struct {
	LoginState   int
	Token        string
	RefreshToken string
	ExpiresAt    time.Time
}

type credentialFile struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	ExpiresAt    time.Time `json:"expires_at"`
}

type harFile struct {
	Log struct {
		Entries []struct {
			Request struct {
				Method string `json:"method"`
				URL    string `json:"url"`
			} `json:"request"`
			Response struct {
				Content struct {
					Text string `json:"text"`
				} `json:"content"`
			} `json:"response"`
		} `json:"entries"`
	} `json:"log"`
}

type phoneLoginResponse struct {
	Code int `json:"code"`
	Data struct {
		Data struct {
			AccessToken     string `json:"access_token"`
			RefreshToken    string `json:"refresh_token"`
			ExpireIn        int64  `json:"expire_in"`
			RefreshExpireIn int64  `json:"refresh_expire_in"`
		} `json:"data"`
	} `json:"data"`
}

var tokenFileName string
var u User

func init() {
	binaryFilePath, _ := os.Executable()
	ksFilePath := filepath.Dir(binaryFilePath) + string(os.PathSeparator)
	if runtime.GOOS == "windows" {
		tokenFileName = ksFilePath + "ks.token"
	} else {
		tokenFileName = ksFilePath + ".ks.token"
	}
	u.LoadToken()
}

// LoadToken loads the current JSON credential format and the legacy
// "token unix-time" format used by releases before v1.
func (u *User) LoadToken() {
	u.LoginState = 0
	u.Token = ""
	u.RefreshToken = ""
	u.ExpiresAt = time.Time{}

	data, err := os.ReadFile(tokenFileName)
	if err != nil {
		return
	}
	var stored credentialFile
	if json.Unmarshal(data, &stored) == nil && stored.AccessToken != "" {
		u.Token = stored.AccessToken
		u.RefreshToken = stored.RefreshToken
		u.ExpiresAt = stored.ExpiresAt
	} else {
		parts := strings.Fields(string(data))
		if len(parts) < 2 {
			return
		}
		expiresUnix, parseErr := strconv.ParseInt(parts[1], 10, 64)
		if parseErr != nil {
			return
		}
		u.Token = parts[0]
		u.ExpiresAt = time.Unix(expiresUnix, 0)
	}
	if u.Token == "" {
		return
	}
	if !u.ExpiresAt.IsZero() && time.Now().After(u.ExpiresAt) {
		u.LoginState = -1
		return
	}
	u.LoginState = 1
}

func (u *User) ImportTokens(accessToken, refreshToken string, validFor time.Duration) error {
	accessToken = strings.TrimSpace(accessToken)
	if accessToken == "" {
		return errors.New("访问令牌不能为空")
	}
	if validFor <= 0 {
		return errors.New("令牌有效期必须大于 0")
	}
	stored := credentialFile{
		AccessToken:  accessToken,
		RefreshToken: strings.TrimSpace(refreshToken),
		ExpiresAt:    time.Now().Add(validFor),
	}
	if err := saveCredentials(stored); err != nil {
		return err
	}
	u.Token = stored.AccessToken
	u.RefreshToken = stored.RefreshToken
	u.ExpiresAt = stored.ExpiresAt
	u.LoginState = 1
	globalSync(*u)
	return nil
}

func (u *User) ImportTokensFromEnvironment(validFor time.Duration) error {
	return u.ImportTokens(os.Getenv(AccessTokenEnv), os.Getenv(RefreshTokenEnv), validFor)
}

// ImportTokensFromHAR reuses a session that the user obtained by completing
// the website's normal interactive login. It never submits or solves a
// captcha. HAR files contain sensitive data and must not be shared publicly.
func (u *User) ImportTokensFromHAR(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	var archive harFile
	decoder := json.NewDecoder(io.LimitReader(file, 512<<20))
	if err := decoder.Decode(&archive); err != nil {
		return fmt.Errorf("解析 HAR: %w", err)
	}
	for index := len(archive.Log.Entries) - 1; index >= 0; index-- {
		entry := archive.Log.Entries[index]
		if entry.Request.Method != "POST" || !strings.Contains(entry.Request.URL, "/iam/userLogin/phoneLogin") {
			continue
		}
		var response phoneLoginResponse
		if err := json.Unmarshal([]byte(entry.Response.Content.Text), &response); err != nil {
			continue
		}
		if response.Code != 200000 || response.Data.Data.AccessToken == "" {
			continue
		}
		validFor := time.Duration(response.Data.Data.ExpireIn) * time.Millisecond
		if validFor <= 0 || validFor > 365*24*time.Hour {
			return fmt.Errorf("HAR 中的访问令牌有效期无效")
		}
		return u.ImportTokens(response.Data.Data.AccessToken, response.Data.Data.RefreshToken, validFor)
	}
	return errors.New("HAR 中没有找到成功的新版网页登录响应")
}

func (u *User) Logout() {
	if err := os.Remove(tokenFileName); err != nil && !os.IsNotExist(err) {
		fmt.Println("删除登录凭证失败：", err)
		return
	}
	u.LoginState = 0
	u.Token = ""
	u.RefreshToken = ""
	u.ExpiresAt = time.Time{}
	globalSync(*u)
	fmt.Println("已删除登录凭证")
}

func GetLoginState() int {
	return u.LoginState
}

func AccessToken() string {
	if u.LoginState != 1 {
		return ""
	}
	return u.Token
}

func Current() User {
	return u
}

func globalSync(imported User) {
	u = imported
}

func saveCredentials(stored credentialFile) error {
	data, err := json.MarshalIndent(stored, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(tokenFileName, data, 0600); err != nil {
		return err
	}
	if err := os.Chmod(tokenFileName, 0600); err != nil && runtime.GOOS != "windows" {
		return err
	}
	if err := hideFile(tokenFileName); err != nil {
		return err
	}
	return nil
}
