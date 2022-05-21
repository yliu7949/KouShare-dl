package user

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/tidwall/gjson"
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
	if runtime.GOOS == "windows" {
		tokenFileName = "ks.token"
	} else {
		tokenFileName = ".ks.token"
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
		text := strings.Split(fmt.Sprintf("%s", f), " ")

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

// Login 使用短信验证码的方式登陆“蔻享学术”平台，登陆成功后获得token，并将token保存在当前路径下的token文件中
func (u *User) Login() error {
	URL := "https://login.koushare.com/api/api-user/"
	res1, err := http.PostForm(URL+"sendSms", url.Values{"phone": {u.PhoneNumber}, "scope": {"LOGIN"}})
	if err != nil {
		return err
	}
	body, err := ioutil.ReadAll(res1.Body)
	res1.Body.Close()
	if err != nil {
		return err
	}
	if res1.StatusCode == 200 && gjson.Get(fmt.Sprintf("%s", body), "code").String() == "200" {
		fmt.Printf("短信验证码发送成功，请输入6位验证码：")
		var verifyCode string
		_, err = fmt.Scan(&verifyCode)
		if err != nil {
			return err
		}

		res2, err := http.PostForm(URL+"smsLogin", url.Values{"phone": {u.PhoneNumber}, "key": {verifyCode}, "rm": {"1"}})
		if err != nil {
			return err
		}
		body, err = ioutil.ReadAll(res2.Body)
		res2.Body.Close()
		if err != nil {
			return err
		}
		if res2.StatusCode == 200 && gjson.Get(fmt.Sprintf("%s", body), "code").String() == "200" {
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
		fmt.Println(gjson.Get(fmt.Sprintf("%s", body), "msg").String())
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

	f, err := os.OpenFile("ks.token", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
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
	if err = hideFile("ks.token"); err != nil {
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
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/84.0.4147.105 Safari/537.36")
	if u.LoginState == 1 { //如果token有效，则添加cookie请求头
		req.Header.Set("Cookie", "Token="+u.Token)
	}
	if len(headers) != 0 {
		for key, value := range headers[0] {
			req.Header.Set(key, value)
		}
	}

	resp, _ := http.DefaultClient.Do(req)
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s", data), nil
}