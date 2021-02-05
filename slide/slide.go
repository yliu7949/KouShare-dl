package slide

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/tidwall/gjson"
	"github.com/yliu7949/KouShare-dl/live"
)

var (
	vids         []string
	noteTitles   []string
	downloadUrls []string
)

type Slide struct {
	Vid         string
	Svid        string
	seriesName  string
	SaveDir     string
	noteTitle   string
	downloadUrl string
}

func (s *Slide) DownloadSlides(hasSeriesFlag bool, qpdfPath string) {
	Url := "https://www.koushare.com/api/api-video/getVideoById?vid=" + s.Vid + "&related=3"
	if str, err := live.MyGetRequest(Url); err != nil {
		fmt.Println("Get请求出错：", err)
		return
	} else {
		s.Svid = gjson.Get(str, "data.svid").String()
	}
	if s.Svid == "0" || s.Svid == "" { //判断是否为系列视频，非系列视频一般无课件
		fmt.Println("vid为" + s.Vid + "的视频暂无课件。")
		return
	}
	Url = "https://www.koushare.com/api/api-video/getSeriesVideo?svid=" + s.Svid
	str, err := live.MyGetRequest(Url)
	if err != nil {
		fmt.Println("Get请求出错：", err)
		return
	}
	s.seriesName = gjson.Get(str, `data.1.svname`).String()

	if _, err := os.Stat(s.SaveDir); os.IsNotExist(err) {
		if err := os.Mkdir("ok", os.ModePerm); err != nil {
			fmt.Println("创建下载文件夹失败：", err)
			return
		}
	}
	if !hasSeriesFlag {
		s.noteTitle = gjson.Get(str, `data.#(vid==`+s.Vid+`).vcourseware`).String()
		s.downloadUrl = gjson.Get(str, `data.#(vid==`+s.Vid+`).vcoursewareurl`).String()
		if s.downloadUrl == "" || s.noteTitle == "" {
			fmt.Println("vid为" + s.Vid + "的视频暂无课件。")
			return
		}
		s.saveFile()
		if qpdfPath != "" {
			if qpdfPath[len(qpdfPath)-1:] != "\\" && qpdfPath[len(qpdfPath)-1:] != "/" {
				qpdfPath = qpdfPath + "\\"
			}
			qpdfBinPath = qpdfPath
			removeWM(s.SaveDir + s.noteTitle)
		}
	} else {
		seriesName := gjson.Get(str, `data.#(vid==`+s.Vid+`).svname`).String()
		s.SaveDir += fmt.Sprintf("%s_slides\\", seriesName)
		if _, err := os.Stat(s.SaveDir); os.IsNotExist(err) {
			if err := os.Mkdir(s.SaveDir, os.ModePerm); err != nil {
				fmt.Println("创建下载文件夹失败：", err)
				return
			}
		}
		tmpStr := gjson.Get(str, `data.#.vid`).String()
		vids = strings.Split(tmpStr[1:len(tmpStr)-1], ",")
		tmpStr = gjson.Get(str, `data.#.vcourseware`).String()
		noteTitles = strings.Split(tmpStr[1:len(tmpStr)-1], ",")
		tmpStr = gjson.Get(str, `data.#.vcoursewareurl`).String()
		downloadUrls = strings.Split(tmpStr[1:len(tmpStr)-1], ",")

		for num, notetitle := range noteTitles {
			s.Vid = vids[num]
			s.noteTitle = notetitle
			s.downloadUrl = downloadUrls[num]
			if s.downloadUrl == `""` || s.noteTitle == `""` {
				continue
			}
			if num >= 1 && s.noteTitle == noteTitles[num-1] {
				continue
			}
			s.noteTitle = s.noteTitle[1 : len(s.noteTitle)-1]
			s.downloadUrl = s.downloadUrl[1 : len(s.downloadUrl)-1]
			fmt.Printf("正在下载 \"%s\"系列课件(%d/%d)\t", s.seriesName, num+1, len(noteTitles))
			s.saveFile()
			if qpdfPath != "" {
				if qpdfPath[len(qpdfPath)-1:] != "\\" && qpdfPath[len(qpdfPath)-1:] != "/" {
					qpdfPath = qpdfPath + "\\"
				}
				qpdfBinPath = qpdfPath
				removeWM(s.SaveDir + s.noteTitle)
			}
		}
	}
}

func (s *Slide) saveFile() {
	resp, err := http.Get(s.downloadUrl)
	if err != nil {
		fmt.Println("vid=", s.Vid, err)
		return
	}
	defer resp.Body.Close()
	if len(s.noteTitle) >= 3 && s.noteTitle[len(s.noteTitle)-3:] != "pdf" {
		s.noteTitle += ".pdf"
	}
	fmt.Printf("%s\tvid=%s\n", s.noteTitle, s.Vid)
	dstFile, err := os.OpenFile(s.SaveDir+s.noteTitle, os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		fmt.Println("vid=", s.Vid, err.Error())
		return
	}
	if _, err = io.Copy(dstFile, resp.Body); err != nil {
		fmt.Println("vid=", s.Vid, err.Error())
		return
	}
	_ = dstFile.Close()
}
