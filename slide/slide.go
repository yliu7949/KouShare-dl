package slide

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/tidwall/gjson"
	"github.com/yliu7949/KouShare-dl/user"
)

// Slide 包含视频号、课件下载链接等基本信息
type Slide struct {
	Vid             string
	svid            string   //系列id
	seriesName      string   //系列名字
	svpid           string   //子系列id
	svpName         string   //子系列名字
	name            string   //单个课件的文件名
	url             string   //单个课件的下载链接
	coursewareNames []string //该系列视频中所有课件的文件名
	coursewareURLs  []string //该系列视频中所有课件的下载链接
	QpdfPath        string   //qpdf的bin路径
	SaveDir         string
}

// DownloadSingleSlide 下载指定vid的视频对应的课件
func (s *Slide) DownloadSingleSlide() {
	if ok := s.getSlideInfo(); !ok {
		fmt.Println("获取课件信息失败。")
		return
	}

	if s.url == "" {
		fmt.Println("vid为" + s.Vid + "的视频暂无课件。")
		return
	}
	if _, err := os.Stat(s.SaveDir); os.IsNotExist(err) {
		if err := os.Mkdir("ok", os.ModePerm); err != nil {
			fmt.Println("创建下载文件夹失败：", err)
			return
		}
	}
	s.saveFile()
}

// DownloadSeriesSlides 下载系列视频中的所有课件
func (s *Slide) DownloadSeriesSlides() {
	if ok := s.getSlideInfo(); !ok {
		fmt.Println("获取课件信息失败。")
		return
	}
	if s.svid == "0" || s.svid == "" { //判断是否是系列视频，若不是系列视频则仅下载该课件
		s.DownloadSingleSlide()
		return
	}

	s.findSeriesSlides()
	if s.svpName != "" {
		s.SaveDir += fmt.Sprintf("%s_%s_slides\\", s.seriesName, s.svpName)
	} else {
		s.SaveDir += fmt.Sprintf("%s_slides\\", s.seriesName)
	}
	if _, err := os.Stat(s.SaveDir); os.IsNotExist(err) {
		if err := os.Mkdir(s.SaveDir, os.ModePerm); err != nil {
			fmt.Println("创建下载文件夹失败：", err)
			return
		}
	}

	var tempName string //用来记录for循环中上一次下载课件的名字
	for i, coursewareURL := range s.coursewareURLs {
		if s.url = coursewareURL[1 : len(coursewareURL)-1]; len(s.url) == 0 {
			continue
		}
		fmt.Println("比较：", tempName+`"`, "和", s.coursewareNames[i][1:])
		if i >= 1 && tempName+`"` == s.coursewareNames[i][1:] { //若本次要下载的文件与上一次下载的文件相同，则跳过本次下载
			continue
		}
		fmt.Printf("正在下载 \"%s\"系列课件(%d/%d)\t", s.seriesName, i+1, len(s.coursewareURLs))
		s.name = s.coursewareNames[i]
		s.name = s.name[1 : len(s.name)-1]
		tempName = s.name
		s.saveFile()
	}
}

func (s *Slide) getSlideInfo() bool {
	URL := "https://api.koushare.com/api/api-video/getVideoById?vid=" + s.Vid + "&related=3&allData=1&password="
	str, err := user.MyGetRequest(URL)
	if err != nil {
		fmt.Println("Get请求出错：", err)
		return false
	}
	if gjson.Get(str, "code").String() == "500" { //状态为500时，get请求返回的的data为null
		fmt.Printf("%s ", gjson.Get(str, "msg").String())
		return false
	}

	s.svid = gjson.Get(str, "data.svid").String()
	s.seriesName = gjson.Get(str, "data.svname").String()
	s.svpid = gjson.Get(str, "data.svpid").String()
	s.svpName = gjson.Get(str, "data.svpname").String()
	s.name = gjson.Get(str, "data.vcourseware").String()
	s.url = gjson.Get(str, "data.vcoursewareurl").String()
	return true
}

func (s *Slide) findSeriesSlides() {
	if s.svid == "0" || s.svid == "" { //判断是否为系列视频
		return
	}

	var URL string
	if s.svpid != "0" { //判断是否存在子系列视频
		URL = "https://api.koushare.com/api/api-video/getAllVideoBySeriesSub?svpid=" + s.svpid
	} else {
		URL = "https://api.koushare.com/api/api-video/getSeriesVideo?svid=" + s.svid
	}

	if str, err := user.MyGetRequest(URL); err != nil {
		fmt.Println("Get请求出错：", err)
	} else {
		coursewareNames := gjson.Get(str, `data.#(svid=="`+s.svid+`")#.vcourseware`).String()
		s.coursewareNames = strings.Split(coursewareNames[1:len(coursewareNames)-1], ",")
		coursewareURLs := gjson.Get(str, `data.#(svid=="`+s.svid+`")#.vcoursewareurl`).String()
		s.coursewareURLs = strings.Split(coursewareURLs[1:len(coursewareURLs)-1], ",")
	}
}

func (s *Slide) saveFile() {
	resp, err := http.Get(s.url)
	if err != nil {
		fmt.Println("Get请求出错：", err.Error())
		return
	}
	defer resp.Body.Close()
	if len(s.name) >= 3 && s.name[len(s.name)-3:] != "pdf" {
		s.name += ".pdf"
	}
	fmt.Println(s.name)
	dstFile, err := os.OpenFile(s.SaveDir+s.name, os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	if _, err = io.Copy(dstFile, resp.Body); err != nil {
		fmt.Println(err.Error())
		return
	}
	_ = dstFile.Close()

	//优化pdf文件
	if s.QpdfPath != "" {
		qpdfBinPath = s.QpdfPath
		optimizePDF(s.SaveDir + s.name)
	}
}
