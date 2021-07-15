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
	"github.com/yliu7949/KouShare-dl/user"
)

// Video 包含视频号、标题、作者、日期等基本信息
type Video struct {
	Vid          string
	svid         string //系列id
	svpid        string //子系列id
	svpName      string //子系列名字
	title        string
	author       string
	affiliation  string
	abstract     string
	date         string
	seriesName   string //系列名字
	seriesVids   []string
	videoTime    string //视频时长
	size         int64  //视频体积
	easyURL      string //标清播放链接
	standardURL  string //高清播放链接
	url          string //超清播放链接
	statusCode   string //获取视频信息时返回的状态码，401即需要登陆；301即需要付费；200即请求成功（免费视频或已购买视频）；500即视频不存在
	vrName       string //视频类别，分为“付费视频”和“免费视频”两类，若为空则视为“免费视频”
	SaveDir      string
	videoQuality string //实际下载视频时的清晰度，分为“标清”、“高清”和“超清”三类
	wg           sync.WaitGroup
}

// DownloadSingleVideo 下载指定清晰度的视频，若指定的视频清晰度不存在，则尝试下载稍低的清晰度的视频
func (v *Video) DownloadSingleVideo(quality string) {
	if ok := v.GetVideoInfo(); !ok {
		fmt.Println("\n获取视频信息失败。")
		return
	}

	if v.statusCode == "401" {
		fmt.Printf("%s\tvid=%s\n", v.title, v.Vid)
		fmt.Print(" [>>>>>>>>>>>>该视频需登陆，自动取消下载>>>>>>>>>>>>]\n\n")
		return
	} else if v.statusCode == "301" {
		fmt.Printf("%s\tvid=%s\n", v.title, v.Vid)
		fmt.Print(" [>>>>>>>>>>>>该视频需付费，自动取消下载>>>>>>>>>>>>]\n\n")
		return
	}

	var URL string
	if v.vrName != "付费视频" && user.GetLoginState() != 1 {
		URL = v.easyURL
		v.videoQuality = "标清"
	} else {
		switch quality {
		case "high":
			if v.url != "" {
				URL = v.url
				v.videoQuality = "超清"
				break
			}
			fallthrough
		case "standard":
			if v.standardURL != "" {
				URL = v.standardURL
				v.videoQuality = "高清"
				break
			}
			fallthrough
		default:
			URL = v.easyURL
			v.videoQuality = "标清"
		}
	}
	v.getVideoSize(URL)

	//若mp4文件已存在，说明该视频已下载完成。自动跳过该视频的下载。
	if _, err := os.Stat(v.SaveDir + v.title + "_" + v.videoQuality + ".mp4"); err == nil {
		fmt.Printf("%s\tvid=%s\t%s\n", v.title, v.Vid, v.videoQuality)
		fmt.Print(" [>>>>>>>>>>>>该视频已下载，自动跳过下载>>>>>>>>>>>>]\n\n")
		return
	}

	//若tmp文件已存在，说明该视频处于下载中断状态。为视频文件追加未下载的内容。
	var firstByte = 0
	if tmpFileSize := v.checkTmpFileSize(); tmpFileSize != 0 {
		if tmpFileSize == v.size {
			err := os.Rename(v.SaveDir+v.title+"_"+v.videoQuality+".tmp", v.SaveDir+v.title+"_"+v.videoQuality+".mp4")
			if err != nil {
				fmt.Println(err)
			}
			fmt.Printf("%s\tvid=%s\t%s\n", v.title, v.Vid, v.videoQuality)
			fmt.Print(" [>>>>>>>>>>>>该视频已下载，自动跳过下载>>>>>>>>>>>>]\n\n")
			return
		}
		firstByte = int(tmpFileSize)
	}
	req, err := http.NewRequest(http.MethodGet, URL, nil)
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
	fileName := v.SaveDir + v.title + "_" + v.videoQuality + ".tmp"
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

// DownloadSeriesVideos 下载指定清晰度的系列视频
func (v *Video) DownloadSeriesVideos(quality string) {
	if ok := v.GetVideoInfo(); !ok {
		fmt.Println("获取视频信息失败。")
		return
	}
	if v.svid == "0" || v.svid == "" { //判断是否是系列视频，若不是系列视频则仅下载该视频
		v.DownloadSingleVideo(quality)
		return
	}

	if v.svpName != "" {
		v.SaveDir += fmt.Sprintf("%s_%s_videos\\", v.seriesName, v.svpName)
	} else {
		v.SaveDir += fmt.Sprintf("%s_videos\\", v.seriesName)
	}
	if _, err := os.Stat(v.SaveDir); os.IsNotExist(err) {
		if err := os.Mkdir(v.SaveDir, os.ModePerm); err != nil {
			fmt.Println("创建下载文件夹失败：", err)
			return
		}
	}

	v.findSeriesVideos()
	seriesVids := v.seriesVids //此行须保留
	for i, vid := range seriesVids {
		fmt.Printf("正在下载 \"%s\"系列视频(%d/%d)\t", v.seriesName, i+1, len(seriesVids))
		v.Vid = vid
		v.DownloadSingleVideo(quality)
	}
}

// GetVideoInfo 获取视频的基本信息
func (v *Video) GetVideoInfo() bool {
	URL := "https://api.koushare.com/api/api-video/getVideoById?vid=" + v.Vid + "&related=3&allData=1&password="
	str, err := user.MyGetRequest(URL)
	if err != nil {
		fmt.Println("Get请求出错：", err)
		return false
	}

	if v.statusCode = gjson.Get(str, "code").String(); v.statusCode == "500" { //状态为500时，get请求返回的的data为null
		fmt.Printf("%s ", gjson.Get(str, "msg").String())
		return false
	}

	v.svid = gjson.Get(str, "data.svid").String()
	v.svpid = gjson.Get(str, "data.svpid").String()
	v.svpName = gjson.Get(str, "data.svpname").String()
	v.title = gjson.Get(str, "data.vtitle").String()
	v.author = gjson.Get(str, "data.details_name").String()
	v.affiliation = gjson.Get(str, "data.details_affiliation").String()
	v.abstract = gjson.Get(str, "data.videoabstract").String()
	v.date = gjson.Get(str, "data.details_date").String()
	v.easyURL = gjson.Get(str, "data.easyurl").String()
	v.standardURL = gjson.Get(str, "data.standardurl").String()
	v.url = gjson.Get(str, "data.url").String()
	v.vrName = gjson.Get(str, "data.vrname").String()
	v.seriesName = gjson.Get(str, "data.svname").String()
	v.videoTime = gjson.Get(str, "data.vtime").String()
	return true
}

func (v *Video) checkTmpFileSize() (size int64) {
	fileName := v.SaveDir + v.title + "_" + v.videoQuality + ".tmp"
	if _, err := os.Stat(fileName); os.IsNotExist(err) {
		return 0
	}
	_ = filepath.Walk(fileName, func(path string, f os.FileInfo, err error) error {
		size = f.Size()
		return nil
	})
	return size
}

func (v *Video) getVideoSize(URL string) {
	// URL参数为视频的真实下载地址
	req, err := http.NewRequest(http.MethodGet, URL, nil)
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
	fmt.Printf("%s\tvid=%s\t%s\n", v.title, v.Vid, v.videoQuality)
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
			err := os.Rename(v.SaveDir+v.title+"_"+v.videoQuality+".tmp", v.SaveDir+v.title+"_"+v.videoQuality+".mp4")
			if err != nil {
				fmt.Println(err)
			}
			v.wg.Done()
			return
		}
	}
}

// ShowVideoInfo 按照格式输出视频的基本信息
func (v *Video) ShowVideoInfo() {
	if ok := v.GetVideoInfo(); !ok {
		fmt.Println("\n获取视频信息失败。")
		return
	}
	if v.statusCode == "200" {
		if user.GetLoginState() == 1 {
			if v.url != "" {
				v.getVideoSize(v.url)
				v.videoQuality = " [超清]"
			} else if v.standardURL != "" {
				v.getVideoSize(v.standardURL)
				v.videoQuality = " [高清]"
			} else {
				v.getVideoSize(v.easyURL)
				v.videoQuality = " [标清]"
			}
		} else {
			v.getVideoSize(v.easyURL)
			v.videoQuality = " [标清]"
		}
	} else {
		v.videoQuality = " [未知]"
	}

	if v.videoTime == "" {
		v.videoTime = "Unknown"
	}
	if v.abstract == "" {
		v.abstract = "(无)"
	}
	if v.vrName == "" && v.size != 0 {
		v.vrName = "免费视频"
	}
	fmt.Printf("%s (vid=%s):\n", v.title, v.Vid)
	fmt.Printf("\n\t时长：%-22s讲者：%s\n", v.videoTime+"min", v.author)
	fmt.Printf("\t体积：%-20s单位：%s\n", strconv.Itoa(int(v.size/1024/1024))+"MB"+v.videoQuality, v.affiliation)
	fmt.Printf("\t日期：%-22s系列：%s\n", v.date, v.seriesName)
	fmt.Printf("\t类别：%-18s分组：%s\n", v.vrName, v.svpName)
	fmt.Printf("\n\t视频简介：%s\n\n", v.abstract)
}

func (v *Video) findSeriesVideos() {
	if v.svid == "0" || v.svid == "" { //判断是否为系列视频
		return
	}

	var URL string
	if v.svpid != "0" { //判断是否存在子系列视频
		URL = "https://api.koushare.com/api/api-video/getAllVideoBySeriesSub?svpid=" + v.svpid
	} else {
		URL = "https://api.koushare.com/api/api-video/getSeriesVideo?svid=" + v.svid
	}

	if str, err := user.MyGetRequest(URL); err != nil {
		fmt.Println("Get请求出错：", err)
	} else {
		seriesVids := gjson.Get(str, `data.#(svid=="`+v.svid+`")#.vid`).String()
		v.seriesVids = strings.Split(seriesVids[1:len(seriesVids)-1], ",")
	}
}
