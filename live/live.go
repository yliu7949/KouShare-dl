package live

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/tidwall/gjson"
	"github.com/yliu7949/KouShare-dl/internal/color"
	"github.com/yliu7949/KouShare-dl/internal/proxy"
	"github.com/yliu7949/KouShare-dl/user"
)

// Live 包含房间号、直播链接和直播状态等信息
type Live struct {
	lid            string
	RoomID         string
	isLive         string // 值为0表示直播未开始；值为1表示正在进行直播；值为2表示直播已结束；值为3表示录播视频已上线。
	title          string
	date           string // 开播时间
	sponsor        string // 主办单位
	notice         string // 最新通知
	clicks         string // 点击量
	topicName      string // 专题/回放
	m3u8URL        string
	newTsURL       string
	quickReplayURL string // 快速回放地址
	rtmpURL        string // 正式回放视频地址
	playback       string // 值为0表示无回放；值为1表示有回放。
	needPassword   string // 值为0表示无需密码；值为1表示需要密码。
	Password       string // 观看直播间需要输入的密码
	statusCode     string // 获取直播信息时返回的状态码，301即需要密码或密码不正确；200即请求成功（无需密码或密码正确）。
	SaveDir        string
}

// WaitAndRecordTheLive 倒计时结束后开始录制直播
func (l *Live) WaitAndRecordTheLive(liveTime string, autoMerge bool) {
	if liveTime != "" {
		loc, _ := time.LoadLocation("Local")
		parsedTime, err := time.ParseInLocation("2006-01-02 15:04:05", liveTime, loc)
		if err != nil {
			fmt.Println("时间解析出错：", err)
			return
		}
		fmt.Println("设定的直播时间为：", parsedTime)
		deltaTime, _ := time.ParseDuration(fmt.Sprint(parsedTime.Unix()-time.Now().Unix()) + "s")
		go func() {
			for {
				if parsedTime.Unix()-time.Now().Unix() <= 0 {
					fmt.Println("\n直播时间到。")
					return
				}
				fmt.Printf("\r 还有%d秒开始直播...", parsedTime.Unix()-time.Now().Unix())
				time.Sleep(time.Second)
			}
		}()
		time.Sleep(deltaTime)
	}

	if !l.getLidByRoomID() {
		return
	}
	l.checkLiveStatus()
	if l.needPassword == "1" && l.Password == "" {
		fmt.Println(color.Highlight("该直播间需要密码，请使用 --password 参数指定密码。"))
		return
	}
	l.getLiveByRoomID(true)

	if l.statusCode == "301" {
		fmt.Println(color.Highlight("直播间密码不正确。"))
		return
	}
	if l.isLive != "1" {
		var msg string
		switch l.isLive {
		case "0":
			fmt.Printf("直播尚未开始，开播时间为 %s，倒计时结束后将自动开始录制。\n", l.date)
			loc, _ := time.LoadLocation("Local")
			parsedTime, err := time.ParseInLocation("2006-01-02 15:04:05", l.date, loc)
			if err != nil {
				fmt.Println("直播时间解析出错：", err)
				return
			}
			deltaTime, _ := time.ParseDuration(fmt.Sprint(parsedTime.Unix()-time.Now().Unix()) + "s")
			formatDuration := func(seconds int64) string {
				duration := time.Duration(seconds) * time.Second
				days := duration / (24 * time.Hour)
				duration -= days * (24 * time.Hour)
				hours := duration / time.Hour
				duration -= hours * time.Hour
				minutes := duration / time.Minute
				duration -= minutes * time.Minute
				seconds = int64(duration.Seconds())

				return fmt.Sprintf("距开播：%02d天%02d时%02d分%02d秒", days, hours, minutes, seconds)
			}
			go func() {
				for {
					if parsedTime.Unix()-time.Now().Unix() <= 0 {
						fmt.Println("\n直播时间到。")
						return
					}
					fmt.Printf("\r %s...", formatDuration(parsedTime.Unix()-time.Now().Unix()))
					time.Sleep(time.Second)
				}
			}()
			time.Sleep(deltaTime)
		case "2":
			msg = "直播已结束。"
			if l.quickReplayURL != "" {
				msg += fmt.Sprintf(`快速回放视频已上线，访问 %s 观看快速回放或使用“ks record %s --replay”命令下载快速回放视频。`,
					`https://www.koushare.com/lives/room/`+l.RoomID, l.RoomID)
			} else if l.playback == "0" {
				msg += "本场直播无回放。"
			} else if l.playback == "1" {
				msg += "快速回放暂未上线。"
			}
			fmt.Println(msg)
			return
		case "3":
			msg = "正式回放视频已上线。"
			if l.rtmpURL != "" {
				vid := strings.Split(l.rtmpURL, "/")[len(strings.Split(l.rtmpURL, "/"))-1]
				msg += fmt.Sprintf(`访问 %s 观看录播视频或使用“ks save %s”命令下载正式回放视频。`, l.rtmpURL, vid)
			}
			fmt.Println(msg)
			return
		default:
			fmt.Println("直播未按时开始或已结束。")
			return
		}
	}

	fmt.Println("运行录制程序...")
	l.recordLive(autoMerge)

	fmt.Println("录制结束.")
}

func (l *Live) getLidByRoomID() bool {
	URL := "https://api.koushare.com/api/api-live/getLidByRoomid?roomid=" + l.RoomID
	str, err := user.MyGetRequest(URL)
	if err != nil {
		fmt.Println("Get请求出错：", err)
		return false
	}
	if l.lid = gjson.Get(str, "data").String(); l.lid == "" {
		fmt.Println("直播间ID无效。")
		return false
	}
	return true
}

func (l *Live) checkLiveStatus() {
	URL := "https://api.koushare.com/api/api-live/checkLiveStatus?initial=1&lid=" + l.lid
	if str, err := user.MyGetRequest(URL); err != nil {
		fmt.Println("Get请求出错：", err)
	} else {
		l.isLive = gjson.Get(str, "data.islive").String()
		l.needPassword = gjson.Get(str, "data.lopen").String()
	}
}

func (l *Live) getLiveByRoomID(chooseHighQuality bool) {
	URL := "https://api.koushare.com/api/api-live/getLiveByRoomid?roomid=" + l.RoomID + "&allData=1"
	if l.needPassword == "1" {
		URL = fmt.Sprintf("https://api.koushare.com/api/api-live/getLiveByRoomid?roomid=%s&password=%s&allData=1",
			l.RoomID, l.Password)
	}

	str, err := user.MyGetRequest(URL)
	if err != nil {
		fmt.Println("Get请求出错：", err)
		return
	}

	l.statusCode = gjson.Get(str, "code").String()
	l.title = gjson.Get(str, "data.ltitle").String()
	l.date = gjson.Get(str, "data.livedate").String()
	l.sponsor = gjson.Get(str, "data.lsponsor").String()
	l.notice = gjson.Get(str, "data.lnotice").String()
	l.clicks = gjson.Get(str, "data.lsize").String()
	l.topicName = gjson.Get(str, "data.topicname").String()

	l.isLive = gjson.Get(str, "data.islive").String()
	if chooseHighQuality {
		l.m3u8URL = gjson.Get(str, "data.hlsurl").String()
	} else {
		l.m3u8URL = gjson.Get(str, "data.bqhlsurl").String()
	}
	l.quickReplayURL = gjson.Get(str, "data.lnoticeurl").String()
	l.rtmpURL = gjson.Get(str, "data.rtmpurl").String()
	l.playback = gjson.Get(str, "data.playback").String()
	l.needPassword = gjson.Get(str, "data.lopen").String()
}

// ShowLiveInfo 按照格式输出直播的基本信息
func (l *Live) ShowLiveInfo() {
	l.getLiveByRoomID(true)
	var liveStatus string
	switch l.isLive {
	case "0":
		liveStatus = "直播未开始"
	case "1":
		liveStatus = "正在直播中"
	case "2":
		liveStatus = "直播已结束"
	case "3":
		liveStatus = "录播已上线"
	default:
		liveStatus = "未知的状态"
	}
	if l.playback == "1" {
		l.playback = "[有回放]"
	} else {
		l.playback = "[无回放]"
	}
	if l.topicName == "" {
		l.topicName = "（无）"
	}
	if l.notice == "" {
		l.notice = "（无）"
	}

	fmt.Printf("%s (roomID=%s):\n", l.title, l.RoomID)
	fmt.Printf("\n\t直播状态：%-17s主办单位：%s\n", liveStatus, l.sponsor)
	fmt.Printf("\t开播时间：%-22s有无回放：%s\n", l.date, l.playback)
	fmt.Printf("\t浏览次数：%-22s专题：%s\n", l.clicks, l.topicName)
	if l.needPassword == "1" {
		fmt.Printf("\n\t※该直播间需要密码\n")
	}
	fmt.Printf("\n\t最新通知：%s\n", l.notice)
}

func (l *Live) recordLive(autoMerge bool) {
	if l.m3u8URL == "" {
		fmt.Println(color.Error("m3u8 URL 为空，无法录制。"))
		return
	}

	var url string
	for {
		l.getNewTsURLBym3u8()
		if l.newTsURL != url {
			url = l.newTsURL
			fmt.Println(strings.Split(l.newTsURL[29:], ".")[0], "...")
			if autoMerge {
				l.downloadAndMergeTsFile()
			} else {
				l.downloadTsFile()
			}
		}
		time.Sleep(100 * time.Millisecond)
		l.checkLiveStatus()
		if l.isLive != "1" {
			return
		}
	}
}

func (l *Live) getNewTsURLBym3u8() {
	str, err := user.MyGetRequest(l.m3u8URL)
	if err != nil {
		fmt.Println("Get请求出错：", err)
		return
	}
	lines := strings.Split(str, "\n")
	l.newTsURL = "https://live.am-hpc.com/live/" + lines[len(lines)-2 : len(lines)-1][0]
}

func (l *Live) downloadTsFile() {
	req, err := http.NewRequest(http.MethodGet, l.newTsURL, nil)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Accept-Language", "zh-CN")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Host", "live.am-stc.cn")
	req.Header.Set("Origin", "https://www.koushare.com")
	req.Header.Set("Referer", "https://www.koushare.com")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36")
	resp, _ := proxy.Client.Do(req)
	defer func() {
		err = resp.Body.Close()
		if err != nil {
			log.Fatal(err)
		}
	}()

	name := strings.Split(l.newTsURL[29:], ".")[0]
	fileName := l.SaveDir + name + ".tmp"
	dstFile, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	if _, err = io.Copy(dstFile, resp.Body); err != nil {
		fmt.Println(err.Error())
		return
	}
	_ = dstFile.Close()
	if err = os.Rename(fileName, l.SaveDir+name+".ts"); err != nil {
		fmt.Println(err.Error())
		return
	}
}

func (l *Live) downloadAndMergeTsFile() {
	req, err := http.NewRequest(http.MethodGet, l.newTsURL, nil)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Accept-Language", "zh-CN")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Host", "live.am-stc.cn")
	req.Header.Set("Origin", "https://www.koushare.com")
	req.Header.Set("Referer", "https://www.koushare.com")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36")
	resp, _ := proxy.Client.Do(req)
	defer func() {
		err = resp.Body.Close()
		if err != nil {
			log.Fatal(err)
		}
	}()

	// 过滤视频标题中的不合法字符
	reg, _ := regexp.Compile(`[\\/:*?"<>|]`)
	title := reg.ReplaceAllString(l.title, "")
	fileName := l.SaveDir + fmt.Sprintf("%s_%s.ts", title, strings.Replace(l.date, ":", "_", -1))
	dstFile, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	defer dstFile.Close()
	write := bufio.NewWriter(dstFile)
	_, err = write.ReadFrom(resp.Body)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	err = write.Flush()
	if err != nil {
		fmt.Println(err.Error())
		return
	}
}

// MergeTsFiles 将录制得到的众多.ts文件合并为一个.mp4文件
func MergeTsFiles(dir string, dstFileName string) {
	fmt.Println("开始合并视频文件...")
	var tsFiles []string
	_ = filepath.Walk(dir, func(path string, f os.FileInfo, err error) error {
		if f.IsDir() || len(f.Name()) < 2 || f.Name() == dstFileName {
			return nil
		}
		if f.Name()[len(f.Name())-3:] == ".ts" {
			tsFiles = append(tsFiles, dir+f.Name())
		}
		return nil
	})
	if len(tsFiles) == 0 {
		fmt.Println("没有需要合并的视频片段.")
		return
	}

	dstFile, err := os.OpenFile(dir+dstFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	write := bufio.NewWriter(dstFile)
	for _, tsFile := range tsFiles {
		fileByte, err := os.ReadFile(tsFile)
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		_, err = write.Write(fileByte)
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		_ = write.Flush()
		_ = os.Remove(tsFile)
	}
	_ = dstFile.Close()
	fmt.Println("合并完成.")
}
