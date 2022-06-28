package live

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/tidwall/gjson"
	"github.com/yliu7949/KouShare-dl/user"
)

// Live 包含房间号、直播链接和直播状态等信息
type Live struct {
	lid            string
	RoomID         string
	isLive         string // 值为0表示直播未开始；值为1表示正在进行直播；值为2表示直播已结束；值为3表示录播视频已上线。
	title          string
	date           string
	m3u8URL        string
	newTsURL       string
	quickReplayURL string // 快速回放地址
	rtmpURL        string // 正式回放视频地址
	playback       string // 值为0表示无回放；值为1表示有回放。
	SaveDir        string
}

// WaitAndRecordTheLive 倒计时结束后开始录制直播
func (l *Live) WaitAndRecordTheLive(liveTime string, autoMerge bool) {
	if liveTime != "" {
		loc, _ := time.LoadLocation("Local")
		parsedTime, err := time.ParseInLocation("2006-01-02 15:04:05", liveTime, loc)
		if err != nil {
			fmt.Println("时间解析出错：", err)
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
	l.getLiveByRoomID(true)
	if l.isLive != "1" {
		var msg string
		switch l.isLive {
		case "0":
			msg = "直播尚未开始，距离开播还有一段时间。"
		case "2":
			msg = "直播已结束。"
			if l.quickReplayURL != "" {
				msg += fmt.Sprintf(`快速回放视频已上线，访问 %s 观看快速回放或使用“ks record %s --replay”命令下载快速回放视频。`,
					`https://www.koushare.com/lives/room/`+l.RoomID, l.RoomID)
			} else if l.playback == "0" {
				msg += fmt.Sprintf("本场直播无回放。")
			} else if l.playback == "1" {
				msg += fmt.Sprintf("快速回放暂未上线。")
			}
		case "3":
			msg = "正式回放视频已上线。"
			if l.rtmpURL != "" {
				vid := strings.Split(l.rtmpURL, "/")[len(strings.Split(l.rtmpURL, "/"))-1]
				msg += fmt.Sprintf(`访问 %s 观看录播视频或使用“ks save %s”命令下载正式回放视频。`, l.rtmpURL, vid)
			}
		default:
			msg = "直播未按时开始或已结束。"
		}
		fmt.Println(msg)
		return
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
	}
}

func (l *Live) getLiveByRoomID(chooseHighQuality bool) {
	URL := "https://api.koushare.com/api/api-live/getLiveByRoomid?roomid=" + l.RoomID + "&allData=1"
	str, err := user.MyGetRequest(URL)
	if err != nil {
		fmt.Println("Get请求出错：", err)
		return
	}
	l.title = gjson.Get(str, "data.ltitle").String()
	l.date = gjson.Get(str, "data.livedate").String()
	if chooseHighQuality {
		l.m3u8URL = gjson.Get(str, "data.hlsurl").String()
	} else {
		l.m3u8URL = gjson.Get(str, "data.bqhlsurl").String()
	}
	l.quickReplayURL = gjson.Get(str, "data.lnoticeurl").String()
	l.rtmpURL = gjson.Get(str, "data.rtmpurl").String()
	l.playback = gjson.Get(str, "data.playback").String()
}

func (l *Live) recordLive(autoMerge bool) {
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
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/103.0.0.0 Safari/537.36")
	resp, _ := http.DefaultClient.Do(req)
	defer resp.Body.Close()

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
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/103.0.0.0 Safari/537.36")
	resp, _ := http.DefaultClient.Do(req)
	defer resp.Body.Close()

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
		fileByte, err := ioutil.ReadFile(tsFile)
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
	dstFile.Close()
	fmt.Println("合并完成.")
}
