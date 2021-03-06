package live

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tidwall/gjson"
)

type Live struct {
	lid      string
	RoomID   string
	isLive   string
	title    string
	date     string
	m3u8Url  string
	newTsUrl string
	SaveDir  string
}

func (l *Live) WaitAndRecordTheLive(liveTime string, chooseAutoMerge bool) {
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
				} else {
					fmt.Printf("\r 还有%d秒开始直播...", parsedTime.Unix()-time.Now().Unix())
					time.Sleep(time.Second)
				}
			}
		}()
		time.Sleep(deltaTime)
	}

	l.getLidByRoomID()
	l.checkLiveStatus()
	if l.isLive != "1" {
		fmt.Println("直播未按时开始或已结束。")
		return
	}
	l.getLiveByRoomID(true)
	fmt.Println("运行录制程序...")
	var url string
	for {
		l.getNewTsUrlBym3u8()
		if l.newTsUrl != url {
			url = l.newTsUrl
			fmt.Println(l.newTsUrl[28:], "...")
			if chooseAutoMerge {
				l.downloadAndMergeTsFile()
			} else {
				l.downloadTsFile()
			}
		}
		time.Sleep(100 * time.Millisecond)
		l.checkLiveStatus()
		if l.isLive != "1" {
			fmt.Println("录制结束.")
			return
		}
	}
}

func MyGetRequest(url string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
	req.Header.Set("Referer", "https://www.koushare.com/")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/84.0.4147.105 Safari/537.36")
	resp, _ := http.DefaultClient.Do(req)
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s", data), nil
}

func (l *Live) getLidByRoomID() {
	Url := "https://www.koushare.com/api/api-live/getLidByRoomid?roomid=" + l.RoomID
	if str, err := MyGetRequest(Url); err != nil {
		fmt.Println("Get请求出错：", err)
	} else {
		l.lid = gjson.Get(str, "data").String()
	}
}

func (l *Live) checkLiveStatus() {
	Url := "https://www.koushare.com/api/api-live/checkLiveStatus?lid=" + l.lid
	if str, err := MyGetRequest(Url); err != nil {
		fmt.Println("Get请求出错：", err)
	} else {
		l.isLive = gjson.Get(str, "data.islive").String()
	}
}

func (l *Live) getLiveByRoomID(chooseHighQuality bool) {
	Url := "https://www.koushare.com/api/api-live/getLiveByRoomid?roomid=" + l.RoomID
	str, err := MyGetRequest(Url)
	if err != nil {
		fmt.Println("Get请求出错：", err)
		return
	}
	l.title = gjson.Get(str, "data.ltitle").String()
	l.date = gjson.Get(str, "data.livedate").String()
	if chooseHighQuality {
		l.m3u8Url = gjson.Get(str, "data.hlsurl").String()
	} else {
		l.m3u8Url = gjson.Get(str, "data.bqhlsurl").String()
	}
}

func (l *Live) getNewTsUrlBym3u8() {
	str, err := MyGetRequest(l.m3u8Url)
	if err != nil {
		fmt.Println("Get请求出错：", err)
		return
	}
	lines := strings.Split(str, "\n")
	l.newTsUrl = "https://live.am-stc.cn/live/" + lines[len(lines)-2 : len(lines)-1][0]
}

func (l *Live) downloadTsFile() {
	req, err := http.NewRequest(http.MethodGet, l.newTsUrl, nil)
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
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/84.0.4147.105 Safari/537.36")
	resp, _ := http.DefaultClient.Do(req)
	defer resp.Body.Close()

	fileName := l.SaveDir + l.newTsUrl[28:] + ".tmp"
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
	if err := os.Rename(fileName, l.SaveDir+l.newTsUrl[28:]); err != nil {
		fmt.Println(err)
		return
	}
}

func (l *Live) downloadAndMergeTsFile() {
	req, err := http.NewRequest(http.MethodGet, l.newTsUrl, nil)
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
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/84.0.4147.105 Safari/537.36")
	resp, _ := http.DefaultClient.Do(req)
	defer resp.Body.Close()

	fileName := l.SaveDir + l.title + strings.Replace(l.date, ":", "_", -1) + ".ts"
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
