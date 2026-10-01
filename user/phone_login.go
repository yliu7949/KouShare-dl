package user

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/yliu7949/KouShare-dl/internal/koushare"
	"github.com/yliu7949/KouShare-dl/internal/proxy"
)

type captchaResult struct {
	Ticket  string `json:"ticket"`
	Randstr string `json:"randstr"`
}

// LoginWithPhone keeps the legacy terminal login experience. Tencent's
// mandatory anti-abuse challenge is completed by the user in their normal
// browser; the SMS code itself is read from the terminal.
func (u *User) LoginWithPhone(ctx context.Context, phone, browserPath string, input io.Reader, output io.Writer) error {
	phone = strings.TrimSpace(phone)
	if !loginPhonePattern.MatchString(phone) {
		return errors.New("手机号码格式不正确")
	}
	if input == nil {
		input = strings.NewReader("")
	}
	if output == nil {
		output = io.Discard
	}

	client := koushare.NewClient(&proxy.Client, "")
	captcha, err := client.LoginCaptchaConfig(ctx)
	if err != nil {
		return fmt.Errorf("获取腾讯验证码配置: %w", err)
	}
	fmt.Fprintln(output, "请在打开的 Chrome 小窗口中完成人机验证，以发送短信验证码。")
	verification, err := waitForCaptcha(ctx, captcha, browserPath)
	if err != nil {
		return err
	}
	if err := client.SendLoginSMS(ctx, phone, captcha, verification.Ticket, verification.Randstr); err != nil {
		return fmt.Errorf("发送短信验证码: %w", err)
	}

	fmt.Fprint(output, "短信验证码发送成功，请输入 6 位验证码：")
	line, err := bufio.NewReader(input).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("读取短信验证码: %w", err)
	}
	code := strings.TrimSpace(line)
	if len(code) != 6 || !allDigits(code) {
		return errors.New("短信验证码应为 6 位数字")
	}
	tokens, err := client.PhoneLogin(ctx, phone, code)
	if err != nil {
		return fmt.Errorf("短信验证码登录: %w", err)
	}
	validFor := time.Duration(tokens.ExpireIn) * time.Millisecond
	if validFor <= 0 || validFor > 365*24*time.Hour {
		return errors.New("登录接口返回的访问令牌有效期无效")
	}
	return u.ImportTokens(tokens.AccessToken, tokens.RefreshToken, validFor)
}

func waitForCaptcha(ctx context.Context, config koushare.CaptchaConfig, browserPath string) (captchaResult, error) {
	var result captchaResult
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return result, fmt.Errorf("启动本机验证码回调: %w", err)
	}
	defer listener.Close()

	nonceBytes := make([]byte, 16)
	if _, err := rand.Read(nonceBytes); err != nil {
		return result, fmt.Errorf("生成验证码回调地址: %w", err)
	}
	callbackPath := "/captcha/" + hex.EncodeToString(nonceBytes)
	statePath := "/state/" + hex.EncodeToString(nonceBytes)
	var closeWindow atomic.Bool
	resultChannel := make(chan captchaResult, 1)
	mux := http.NewServeMux()
	mux.HandleFunc("/", captchaPageHandler(config, callbackPath, statePath))
	mux.HandleFunc(callbackPath, captchaCallbackHandler(resultChannel))
	mux.HandleFunc(statePath, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		if closeWindow.Load() {
			w.WriteHeader(http.StatusGone)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	server := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       15 * time.Second,
	}
	serverErrors := make(chan error, 1)
	go func() {
		if serveErr := server.Serve(listener); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			serverErrors <- serveErr
		}
	}()
	defer func() {
		shutdownContext, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownContext)
	}()

	pageURL := "http://" + listener.Addr().String() + "/"
	window, err := openCaptchaWindow(pageURL, browserPath)
	if err != nil {
		return result, fmt.Errorf("打开 Chrome 验证页面: %w", err)
	}
	defer func() {
		closeWindow.Store(true)
		window.Close()
	}()
	select {
	case <-ctx.Done():
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return result, errors.New("等待人机验证超时；请重新运行登录命令")
		}
		return result, errors.New("短信登录已取消")
	case err := <-serverErrors:
		return result, fmt.Errorf("本机验证码回调: %w", err)
	case result = <-resultChannel:
		return result, nil
	case <-window.done:
		// The browser can finish just after posting a successful result. Give
		// the loopback callback a brief chance to win that race.
		select {
		case result = <-resultChannel:
			return result, nil
		case <-time.After(300 * time.Millisecond):
			return result, errors.New("人机验证窗口已关闭；请重新运行登录命令")
		}
	}
}

func captchaPageHandler(config koushare.CaptchaConfig, callbackPath, statePath string) http.HandlerFunc {
	appID, _ := json.Marshal(config.CaptchaAppID)
	aidEncrypted, _ := json.Marshal(config.AidEncrypted)
	callback, _ := json.Marshal(callbackPath)
	stateURL, _ := json.Marshal(statePath)
	page := fmt.Sprintf(`<!doctype html>
<html lang="zh-CN"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>KouShare-dl 登录验证</title><style>html,body{width:100%%;height:100%%;margin:0;overflow:hidden;background:#fff}</style>
<script src="https://turing.captcha.qcloud.com/TCaptcha.js"></script></head>
<body><script>
const appID=%s, aidEncrypted=%s, callbackURL=%s, stateURL=%s;
setInterval(async()=>{
  try{const response=await fetch(stateURL,{cache:'no-store'});if(response.status===410)window.close();}catch(_error){}
},250);
async function showCaptcha(){
  if(typeof TencentCaptcha!=='function'){setTimeout(showCaptcha,250);return;}
  new TencentCaptcha(appID, async result => {
    if(result.ret!==0){window.close();return;}
    const response=await fetch(callbackURL,{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({ticket:result.ticket,randstr:result.randstr})});
	if(response.ok){
	  setTimeout(()=>window.close(),100);
	}else{
	  window.close();
	}
  },{aidEncrypted}).show();
}
showCaptcha();
</script></body></html>`, appID, aidEncrypted, callback, stateURL)
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" || r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = io.WriteString(w, page)
	}
}

func captchaCallbackHandler(results chan<- captchaResult) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var result captchaResult
		decoder := json.NewDecoder(io.LimitReader(r.Body, 64<<10))
		if err := decoder.Decode(&result); err != nil || strings.TrimSpace(result.Ticket) == "" || strings.TrimSpace(result.Randstr) == "" {
			http.Error(w, "invalid captcha result", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusNoContent)
		select {
		case results <- result:
		default:
		}
	}
}

type captchaWindow struct {
	command    *exec.Cmd
	done       chan struct{}
	profileDir string
	closeOnce  sync.Once
}

func (w *captchaWindow) Close() {
	w.closeOnce.Do(func() {
		select {
		case <-w.done:
		case <-time.After(750 * time.Millisecond):
		}
		w.terminateProcesses()
		select {
		case <-w.done:
		case <-time.After(500 * time.Millisecond):
		}
		_ = os.RemoveAll(w.profileDir)
	})
}

func (w *captchaWindow) terminateProcesses() {
	if runtime.GOOS == "windows" {
		if w.command.Process != nil {
			_ = exec.Command("taskkill", "/PID", strconv.Itoa(w.command.Process.Pid), "/T", "/F").Run()
		}
		return
	}
	// On macOS, `open -n` owns only the launcher. Find the isolated Chrome
	// instance by the random profile directory so the user's normal browser is
	// never targeted. This also removes Chromium child processes on Linux.
	if output, err := exec.Command("pgrep", "-f", w.profileDir).Output(); err == nil {
		for _, field := range strings.Fields(string(output)) {
			pid, parseErr := strconv.Atoi(field)
			if parseErr != nil || pid == os.Getpid() {
				continue
			}
			if process, findErr := os.FindProcess(pid); findErr == nil {
				_ = process.Kill()
			}
		}
	}
	if w.command.Process != nil {
		_ = w.command.Process.Kill()
	}
}

func openCaptchaWindow(pageURL, browserPath string) (*captchaWindow, error) {
	executable, err := chromeExecutable(browserPath)
	if err != nil {
		return nil, err
	}
	profileDir, err := os.MkdirTemp("", "koushare-captcha-")
	if err != nil {
		return nil, fmt.Errorf("创建临时浏览器配置: %w", err)
	}
	command := captchaChromeCommand(executable, pageURL, profileDir)
	command.Stdout = io.Discard
	command.Stderr = io.Discard
	if err := command.Start(); err != nil {
		_ = os.RemoveAll(profileDir)
		return nil, err
	}
	window := &captchaWindow{command: command, done: make(chan struct{}), profileDir: profileDir}
	go func() {
		_ = command.Wait()
		close(window.done)
	}()
	return window, nil
}

func captchaChromeCommand(executable, pageURL, profileDir string) *exec.Cmd {
	args := captchaChromeArgs(pageURL, profileDir)
	if runtime.GOOS == "darwin" {
		if appBundle := macAppBundle(executable); appBundle != "" {
			openArgs := []string{"-W", "-n", "-a", appBundle, "--args"}
			return exec.Command("open", append(openArgs, args...)...)
		}
	}
	return exec.Command(executable, args...)
}

func macAppBundle(executable string) string {
	const suffix = ".app/"
	index := strings.Index(executable, suffix)
	if index < 0 {
		return ""
	}
	return executable[:index+len(".app")]
}

func captchaChromeArgs(pageURL, profileDir string) []string {
	return []string{
		"--app=" + pageURL,
		"--window-size=460,600",
		"--window-position=120,120",
		"--user-data-dir=" + profileDir,
		"--no-first-run",
		"--no-default-browser-check",
		"--disable-background-mode",
	}
}

func chromeExecutable(browserPath string) (string, error) {
	if path := strings.TrimSpace(browserPath); path != "" {
		return path, nil
	}
	candidates := []string{"google-chrome", "google-chrome-stable", "chromium", "chromium-browser"}
	switch runtime.GOOS {
	case "darwin":
		candidates = append([]string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
		}, candidates...)
		if home, err := os.UserHomeDir(); err == nil {
			candidates = append([]string{
				filepath.Join(home, "Applications/Google Chrome.app/Contents/MacOS/Google Chrome"),
				filepath.Join(home, "Applications/Chromium.app/Contents/MacOS/Chromium"),
			}, candidates...)
		}
	case "windows":
		for _, base := range []string{os.Getenv("PROGRAMFILES"), os.Getenv("PROGRAMFILES(X86)"), os.Getenv("LOCALAPPDATA")} {
			if base != "" {
				candidates = append([]string{filepath.Join(base, "Google", "Chrome", "Application", "chrome.exe")}, candidates...)
			}
		}
	}
	for _, candidate := range candidates {
		if filepath.IsAbs(candidate) {
			if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
				return candidate, nil
			}
			continue
		}
		if path, err := exec.LookPath(candidate); err == nil {
			return path, nil
		}
	}
	return "", errors.New("没有找到 Chrome/Chromium；请用 --browser-path 指定浏览器可执行文件")
}

func allDigits(value string) bool {
	for _, character := range value {
		if character < '0' || character > '9' {
			return false
		}
	}
	return value != ""
}
