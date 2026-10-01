package slide

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/yliu7949/KouShare-dl/internal/koushare"
	"github.com/yliu7949/KouShare-dl/internal/proxy"
	"github.com/yliu7949/KouShare-dl/user"
)

type Slide struct {
	Vid      string
	QpdfPath string
	SaveDir  string

	info koushare.VideoInfo
}

func (s *Slide) getSlideInfo() bool {
	info, err := koushare.NewClient(&proxy.Client, user.AccessToken()).VideoInfo(context.Background(), s.Vid)
	if err != nil {
		fmt.Println("获取课件信息失败：", err)
		return false
	}
	s.info = info
	return true
}

func (s *Slide) DownloadSingleSlide() {
	if !s.getSlideInfo() {
		return
	}
	if s.info.CoursewareURL == "" {
		fmt.Println("vid 为 " + s.Vid + " 的视频暂无课件。")
		return
	}
	if err := os.MkdirAll(s.SaveDir, 0755); err != nil {
		fmt.Println("创建下载文件夹失败：", err)
		return
	}
	s.saveFile(s.info.CoursewareName, s.info.CoursewareURL)
}

func (s *Slide) DownloadSeriesSlides() {
	client := koushare.NewClient(&proxy.Client, user.AccessToken())
	series, err := client.VideoSeries(context.Background(), s.Vid)
	if err != nil {
		fmt.Println("获取专题课件信息失败：", err)
		return
	}
	if series.SeriesID == nil || len(series.VideoList) <= 1 {
		s.DownloadSingleSlide()
		return
	}
	directory := sanitize(series.SeriesName)
	if directory == "" {
		directory = "series_" + s.Vid
	}
	s.SaveDir = filepath.Join(s.SaveDir, directory+"_slides")
	if err := os.MkdirAll(s.SaveDir, 0755); err != nil {
		fmt.Println("创建下载文件夹失败：", err)
		return
	}
	seen := make(map[string]bool)
	for index, item := range series.VideoList {
		if item.CoursewareURL == "" || seen[item.CoursewareURL] {
			continue
		}
		seen[item.CoursewareURL] = true
		fmt.Printf("正在下载专题 %q 的课件 (%d/%d)\n", series.SeriesName, index+1, len(series.VideoList))
		s.saveFile(item.CoursewareName, item.CoursewareURL)
	}
}

func (s *Slide) saveFile(name, sourceURL string) {
	if name == "" {
		name = "slides_" + s.Vid + ".pdf"
	}
	name = sanitize(name)
	if !strings.HasSuffix(strings.ToLower(name), ".pdf") {
		name += ".pdf"
	}
	finalName := filepath.Join(s.SaveDir, name)
	if _, err := os.Stat(finalName); err == nil {
		fmt.Println("课件已存在，自动跳过：", finalName)
		return
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, sourceURL, nil)
	if err != nil {
		fmt.Println("创建课件请求失败：", err)
		return
	}
	req.Header.Set("Referer", "https://www.koushare.com/")
	req.Header.Set("User-Agent", "KouShare-dl/1")
	resp, err := proxy.Client.Do(req)
	if err != nil {
		fmt.Println("下载课件失败：", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		fmt.Printf("下载课件失败：HTTP %d\n", resp.StatusCode)
		return
	}
	dst, err := os.OpenFile(finalName+".tmp", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		fmt.Println("创建课件文件失败：", err)
		return
	}
	_, copyErr := io.Copy(dst, resp.Body)
	closeErr := dst.Close()
	if copyErr != nil || closeErr != nil {
		fmt.Println("写入课件失败")
		return
	}
	if err := os.Rename(finalName+".tmp", finalName); err != nil {
		fmt.Println("完成课件文件失败：", err)
		return
	}
	fmt.Println("下载完成：", finalName)
	if s.QpdfPath != "" {
		qpdfBinPath = s.QpdfPath
		optimizePDF(finalName)
	}
}

func sanitize(name string) string {
	invalid := regexp.MustCompile(`[\\/:*?"<>|\x00-\x1f]`)
	return strings.TrimSpace(invalid.ReplaceAllString(name, ""))
}
