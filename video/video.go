package video

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/tidwall/gjson"
	"github.com/yliu7949/KouShare-dl/live"
)

type Video struct {
	Vid         string
	Svid        string
	title       string
	author      string
	affiliation string
	abstract    string
	date        string
	seriesName  string
	seriesVids  []string
	videoTime   string
	size        int64
	url         string
	SaveDir     string
	wg          sync.WaitGroup
}

func (v *Video) DownloadSingleVideo() {
	if ok := v.GetVideoInfo(); !ok {
		fmt.Println("\n获取视频信息失败。")
		return
	}

	if v.url == "pay" {
		fmt.Println("\n该视频为付费视频，无法下载。")
		return
	}

	//若mp4文件已存在，说明该视频已下载完成。自动跳过该视频的下载。
	if _, err := os.Stat(v.SaveDir + v.title + ".mp4"); err == nil {
		fmt.Printf("%s\tvid=%s\n", v.title, v.Vid)
		fmt.Print(" [>>>>>>>>>>>>该视频已下载，自动跳过下载>>>>>>>>>>>>]\n\n")
		return
	}

	//若tmp文件已存在，说明该视频处于下载中断状态。为视频文件追加未下载的内容。
	var firstByte = 0
	if tmpFileSize := v.checkTmpFileSize(); tmpFileSize != 0 {
		if tmpFileSize == v.size {
			err := os.Rename(v.SaveDir+v.title+".tmp", v.SaveDir+v.title+".mp4")
			if err != nil {
				fmt.Println(err)
			}
			fmt.Printf("%s\tvid=%s\n", v.title, v.Vid)
			fmt.Print(" [>>>>>>>>>>>>该视频已下载，自动跳过下载>>>>>>>>>>>>]\n\n")
			return
		}
		firstByte = int(tmpFileSize)
	}

	req, err := http.NewRequest(http.MethodGet, v.url, nil)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Accept-Language", "zh-CN")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "Keep-Alive")
	req.Header.Set("GetContentFeatures.DLNA.ORG", "1")
	req.Header.Set("Host", "1254321318.vod2.myqcloud.com")
	req.Header.Set("Range", "bytes="+strconv.Itoa(firstByte)+"-")
	req.Header.Set("Referer", v.url)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/84.0.4147.105 Safari/537.36")
	resp, _ := http.DefaultClient.Do(req)
	defer resp.Body.Close()

	if _, err := os.Stat(v.SaveDir); os.IsNotExist(err) {
		if err := os.Mkdir(v.SaveDir, os.ModePerm); err != nil {
			fmt.Println("创建下载文件夹失败：", err)
			return
		}
	}
	fileName := v.SaveDir + v.title + ".tmp"
	dstFile, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	//启动进度条监听器
	v.wg.Add(1)
	go v.showBar()

	if _, err = io.Copy(dstFile, resp.Body); err != nil {
		fmt.Println(err.Error())
		return
	}
	_ = dstFile.Close()
	v.wg.Wait()
}

func (v *Video) DownloadSeriesVideos() {
	if ok := v.GetVideoInfo(); !ok {
		fmt.Println("获取视频信息失败。")
		return
	}

	if len(v.seriesVids) == 0 { //判断是否是系列视频，若不是系列视频则仅下载该视频
		v.DownloadSingleVideo()
		return
	}

	v.SaveDir += fmt.Sprintf("%s_videos\\", v.seriesName)
	if _, err := os.Stat(v.SaveDir); os.IsNotExist(err) {
		if err := os.Mkdir(v.SaveDir, os.ModePerm); err != nil {
			fmt.Println("创建下载文件夹失败：", err)
			return
		}
	}
	seriesVids := v.seriesVids
	for i, vid := range seriesVids {
		fmt.Printf("正在下载 \"%s\"系列视频(%d/%d)\t", v.seriesName, i+1, len(seriesVids))
		v.Vid = vid
		v.DownloadSingleVideo()
	}
}

func (v *Video) GetVideoInfo() bool {
	Url := "https://www.koushare.com/api/api-video/getVideoById?vid=" + v.Vid + "&related=3"
	if str, err := live.MyGetRequest(Url); err != nil {
		fmt.Println("Get请求出错：", err)
	} else {
		v.Svid = gjson.Get(str, "data.svid").String()
		v.title = gjson.Get(str, "data.vtitle").String()
		v.author = gjson.Get(str, "data.details_name").String()
		v.affiliation = gjson.Get(str, "data.details_affiliation").String()
		v.abstract = gjson.Get(str, "data.videoabstract").String()
		v.date = gjson.Get(str, "data.details_date").String()
		v.url = gjson.Get(str, "data.easyurl").String()
		v.videoTime = gjson.Get(str, "data.vtime").String()
	}
	if v.url == "" || v.title == "" {
		return false
	}
	v.getVideoSize()
	v.findSeriesVideos()
	return true
}

func (v *Video) checkTmpFileSize() (size int64) {
	fileName := v.SaveDir + v.title + ".tmp"
	if _, err := os.Stat(fileName); os.IsNotExist(err) {
		return 0
	}
	_ = filepath.Walk(fileName, func(path string, f os.FileInfo, err error) error {
		size = f.Size()
		return nil
	})
	return size
}

func (v *Video) getVideoSize() {
	req, err := http.NewRequest(http.MethodGet, v.url, nil)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	req.Header.Set("Accept", `text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9`)
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
	req.Header.Set("Cache-Control", "max-age=0")
	req.Header.Set("Connection", "Keep-Alive")
	req.Header.Set("Host", "1254321318.vod2.myqcloud.com")
	req.Header.Set("Range", "bytes=0-104857")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/84.0.4147.105 Safari/537.36")
	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp == nil {
		return
	}
	str := resp.Header.Get("Content-Range")
	array := strings.Split(str, "/")
	if len(array) >= 2 {
		i, _ := strconv.Atoi(array[1])
		v.size = int64(i)
	}
}

func (v *Video) showBar() {
	fmt.Printf("%s\tvid=%s\n", v.title, v.Vid)
	var saveRateGraph string
	for {
		if v.checkTmpFileSize() < v.size && v.size != 0 { //若相等则意味着该视频已下载完毕
			saveRateGraph = ""
			rate := v.checkTmpFileSize() * 100 / v.size
			for i := 0; i < int(rate/2); i++ {
				saveRateGraph += ">"
			}
			fmt.Printf("\r [%-50s]%3d%%  %8d/%d  ", saveRateGraph, rate, v.checkTmpFileSize(), v.size)
			time.Sleep(100 * time.Millisecond)
		} else {
			fmt.Printf("\r [%-50s]%3d%%  %8d/%d\n\n", saveRateGraph+">", 100, v.checkTmpFileSize(), v.size)
			//将下载完成的tmp文件重命名为mp4文件
			err := os.Rename(v.SaveDir+v.title+".tmp", v.SaveDir+v.title+".mp4")
			if err != nil {
				fmt.Println(err)
			}
			v.wg.Done()
			return
		}
	}
}

func (v *Video) ShowVideoInfo() {
	if ok := v.GetVideoInfo(); !ok {
		fmt.Println("\n获取视频信息失败。")
		return
	}

	if v.videoTime == "" {
		v.videoTime = "Unknown"
	}
	if v.abstract == "" {
		v.abstract = "(无)"
	}
	fmt.Printf("%s (vid=%s):\n", v.title, v.Vid)
	fmt.Printf("\n\t时长：%-20s讲者：%s\n", v.videoTime+"min", v.author)
	fmt.Printf("\t体积：%-20s单位：%s\n", strconv.Itoa(int(v.size/1024/1024))+"MB", v.affiliation)
	fmt.Printf("\t日期：%-20s系列：%s\n", v.date, v.seriesName)
	if v.url == "pay" {
		fmt.Println("\t※付费视频")
	}
	fmt.Printf("\n\t视频简介：%s\n\n", v.abstract)
}

func (v *Video) findSeriesVideos() {
	if v.Svid == "0" || v.Svid == "" { //判断是否为系列视频
		return
	}
	Url := "https://www.koushare.com/api/api-video/getSeriesVideo?svid=" + v.Svid
	if str, err := live.MyGetRequest(Url); err != nil {
		fmt.Println("Get请求出错：", err)
	} else {
		v.seriesName = gjson.Get(str, `data.1.svname`).String()
		seriesVids := gjson.Get(str, `data.#(details_name=="`+v.author+`")#.vid`).String()
		v.seriesVids = strings.Split(seriesVids[1:len(seriesVids)-1], ",")
	}
}
