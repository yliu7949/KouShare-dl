package user

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/tidwall/gjson"
	"github.com/yliu7949/KouShare-dl/internal/proxy"
)

// User 用户，包含手机号码、依据token文件判断的登陆状态和token的值
type User struct {
	PhoneNumber string
	LoginState  int //无token文件则为0；token过期则为-1；token有效则为1
	Token       string
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

// LoadToken 检查token文件并更新LoginState和Token
func (u *User) LoadToken() {
	// 判断token文件是否存在
	if _, err := os.Stat(tokenFileName); err == nil {
		f, err := os.ReadFile(tokenFileName)
		if err != nil {
			fmt.Println(err)
			return
		}
		text := strings.Split(string(f), " ")

		// 若token过期，则需要重新登陆获取token
		if t, _ := strconv.Atoi(text[1]); time.Now().Unix()-int64(t) > 604800 {
			u.LoginState = -1
			fmt.Printf("凭证过期，需要重新登陆。\n\n")
		} else {
			u.LoginState = 1
			u.Token = text[0]
			fmt.Printf("登陆凭证有效。\n\n")
		}
	} else {
		u.LoginState = 0
	}
}

// Login 使用短信验证码的方式登陆“蔻享学术”平台，登陆成功后获得token，并将token保存在可执行文件所在路径下的token文件中
func (u *User) Login() error {
	URL := "https://login.koushare.com/api/api-user/"
	res1, err := proxy.Client.PostForm(URL+"sendSms", url.Values{"phone": {u.PhoneNumber}, "scope": {"LOGIN"}})
	if err != nil {
		return err
	}
	body, err := io.ReadAll(res1.Body)
	res1.Body.Close()
	if err != nil {
		return err
	}
	if res1.StatusCode == 200 && gjson.Get(string(body), "code").String() == "200" {
		fmt.Printf("短信验证码发送成功，请输入6位验证码：")
		var verifyCode string
		_, err = fmt.Scan(&verifyCode)
		if err != nil {
			return err
		}

		res2, err := proxy.Client.PostForm(URL+"smsLogin", url.Values{"phone": {u.PhoneNumber}, "key": {verifyCode}, "rm": {"1"}})
		if err != nil {
			return err
		}
		body, err = io.ReadAll(res2.Body)
		res2.Body.Close()
		if err != nil {
			return err
		}
		if res2.StatusCode == 200 && gjson.Get(string(body), "code").String() == "200" {
			fmt.Println("登陆成功。")
			if len(res2.Cookies()) == 1 {
				cookie := *(res2.Cookies()[0])
				u.Token = cookie.Value
				u.LoginState = 1
				if err = saveToken(cookie); err != nil {
					fmt.Println("警告！保存token文件时遇到了问题：", err)
				} else {
					fmt.Println("token文件保存成功。")
				}
			}
		}
	} else {
		fmt.Println(gjson.Get(string(body), "msg").String())
	}
	return nil
}

// Logout 删除token文件，并更新LoginState为0
func (u *User) Logout() {
	if _, err := os.Stat(tokenFileName); err == nil {
		_ = os.Remove(tokenFileName)
	}
	u.LoginState = 0
	fmt.Println("已删除登陆凭证")
}

// GetLoginState 返回LoginState的值；有效登陆则为1，否则为0或-1
func GetLoginState() int {
	return u.LoginState
}

func saveToken(cookie http.Cookie) error {
	// 若token文件存在，则删除该文件
	if _, err := os.Stat(tokenFileName); err == nil {
		err = os.Remove(tokenFileName)
		if err != nil {
			return err
		}
	}

	f, err := os.OpenFile(tokenFileName, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	if _, err = f.WriteString(fmt.Sprintf("%s %d", cookie.Value, cookie.Expires.Unix())); err != nil {
		_ = f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	// 设置“ks.token”文件为隐藏文件
	if err = hideFile(tokenFileName); err != nil {
		return err
	}
	return nil
}

// MyGetRequest 这是一个自定义的Get请求，约定：可变参数headers仅允许传入一个设置header的map。
func MyGetRequest(url string, headers ...map[string]string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
	req.Header.Set("Referer", "https://www.koushare.com/")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/106.0.0.0 Safari/537.36")
	if u.LoginState == 1 { //如果token有效，则添加cookie请求头
		req.Header.Set("Cookie", "Token="+u.Token)
	}
	if len(headers) != 0 {
		for key, value := range headers[0] {
			req.Header.Set(key, value)
		}
	}

	resp, _ := proxy.Client.Do(req)
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
