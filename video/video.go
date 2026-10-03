package video

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/yliu7949/KouShare-dl/internal/hls"
	"github.com/yliu7949/KouShare-dl/internal/koushare"
	"github.com/yliu7949/KouShare-dl/internal/progress"
	"github.com/yliu7949/KouShare-dl/internal/proxy"
	"github.com/yliu7949/KouShare-dl/internal/videoopts"
	"github.com/yliu7949/KouShare-dl/user"
)

type Video struct {
	Vid         string
	SaveDir     string
	VidPrefix   bool
	Concurrency int

	info   koushare.VideoInfo
	series koushare.VideoSeries
}

func apiClient() *koushare.Client {
	return koushare.NewClient(&proxy.Client, user.AccessToken())
}

func (v *Video) GetVideoInfo() bool {
	info, err := apiClient().VideoInfo(context.Background(), v.Vid)
	if err != nil {
		fmt.Println("获取视频信息失败：", err)
		return false
	}
	v.info = info
	return true
}

func (v *Video) DownloadSingleVideo(quality string) {
	if !v.GetVideoInfo() {
		return
	}
	stream, err := v.selectStream(quality)
	if err != nil {
		fmt.Printf("%s\tvid=%s\n下载地址不可用：%v\n", v.info.Title, v.Vid, err)
		return
	}
	if err := os.MkdirAll(v.SaveDir, 0755); err != nil {
		fmt.Println("创建下载文件夹失败：", err)
		return
	}

	qualityName := stream.Label
	if qualityName == "" {
		qualityName = qualityLabel(stream.Height)
	}
	baseName := videoopts.SanitizeFilename(v.info.Title)
	if baseName == "" {
		baseName = "video_" + v.Vid
	}
	if v.VidPrefix {
		baseName = v.Vid + "_" + baseName
	}
	baseName += "_" + qualityName

	if strings.Contains(strings.ToLower(stream.FileURL), ".m3u8") || strings.EqualFold(stream.DRMType, "SimpleAES") {
		v.downloadHLS(stream.FileURL, quality, baseName)
		return
	}
	v.downloadFile(stream.FileURL, baseName+".mp4")
}

func (v *Video) selectStream(quality string) (koushare.VideoStream, error) {
	groups, err := apiClient().VideoPlayAddress(context.Background(), v.Vid, "")
	if err != nil {
		return koushare.VideoStream{}, err
	}
	var streams []koushare.VideoStream
	for _, group := range groups {
		if strings.EqualFold(group.Type, "HLS") {
			streams = append(streams, group.List...)
		}
	}
	if len(streams) == 0 {
		for _, group := range groups {
			streams = append(streams, group.List...)
		}
	}
	if len(streams) == 0 {
		return koushare.VideoStream{}, fmt.Errorf("API 未返回可下载的视频流")
	}
	return videoopts.ChooseStream(streams, quality)
}

func (v *Video) downloadHLS(playlistURL, quality, baseName string) {
	finalName := filepath.Join(v.SaveDir, baseName+".ts")
	if _, err := os.Stat(finalName); err == nil {
		fmt.Printf("%s\tvid=%s\n已下载，自动跳过：%s\n", v.info.Title, v.Vid, finalName)
		return
	}
	temporaryName := finalName + ".tmp"
	dst, err := os.OpenFile(temporaryName, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		fmt.Println("创建临时文件失败：", err)
		return
	}
	display := progress.New(fmt.Sprintf("%s  vid=%s", v.info.Title, v.Vid), os.Stdout)
	lastProgress := hls.Progress{}
	refresh := func(ctx context.Context) (string, error) {
		stream, refreshErr := v.selectStream(quality)
		if refreshErr != nil {
			return "", refreshErr
		}
		return stream.FileURL, nil
	}
	err = (hls.Downloader{Client: &proxy.Client, Concurrency: v.Concurrency}).DownloadRefreshing(context.Background(), playlistURL, refresh, dst, func(current hls.Progress) {
		lastProgress = current
		display.Update(progress.Snapshot{
			Current: int64(current.Completed),
			Total:   int64(current.Total),
			Bytes:   current.DownloadedBytes,
			Unit:    "个片段",
		})
	})
	closeErr := dst.Close()
	if err != nil {
		display.Break()
		fmt.Println("下载失败（可重新运行以重试）：", err)
		return
	}
	if closeErr != nil {
		display.Break()
		fmt.Println("关闭临时文件失败：", closeErr)
		return
	}
	display.Finish(progress.Snapshot{
		Current: int64(lastProgress.Total),
		Total:   int64(lastProgress.Total),
		Bytes:   lastProgress.DownloadedBytes,
		Unit:    "个片段",
	})
	if err := os.Rename(temporaryName, finalName); err != nil {
		fmt.Println("完成临时文件重命名失败：", err)
		return
	}
	fmt.Println("下载完成：", finalName)
}

func (v *Video) downloadFile(sourceURL, filename string) {
	finalName := filepath.Join(v.SaveDir, filename)
	if _, err := os.Stat(finalName); err == nil {
		fmt.Println("已下载，自动跳过：", finalName)
		return
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, sourceURL, nil)
	if err != nil {
		fmt.Println("创建下载请求失败：", err)
		return
	}
	req.Header.Set("Referer", "https://www.koushare.com/")
	req.Header.Set("User-Agent", "KouShare-dl/1")
	resp, err := proxy.Client.Do(req)
	if err != nil {
		fmt.Println("下载失败：", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		fmt.Printf("下载失败：HTTP %d\n", resp.StatusCode)
		return
	}
	temporaryName := finalName + ".tmp"
	dst, err := os.OpenFile(temporaryName, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		fmt.Println("创建临时文件失败：", err)
		return
	}
	display := progress.New(fmt.Sprintf("%s  vid=%s", v.info.Title, v.Vid), os.Stdout)
	progressWriter := display.WrapWriter(dst, resp.ContentLength)
	_, copyErr := io.Copy(progressWriter, resp.Body)
	closeErr := dst.Close()
	if copyErr != nil || closeErr != nil {
		display.Break()
		fmt.Println("写入视频失败：", errorsJoin(copyErr, closeErr))
		return
	}
	progressWriter.Finish()
	if err := os.Rename(temporaryName, finalName); err != nil {
		fmt.Println("完成临时文件重命名失败：", err)
		return
	}
	fmt.Println("下载完成：", finalName)
}

func (v *Video) DownloadSeriesVideos(quality string) {
	series, err := apiClient().VideoSeries(context.Background(), v.Vid)
	if err != nil {
		fmt.Println("获取专题信息失败：", err)
		return
	}
	v.series = series
	if len(series.VideoList) <= 1 || series.SeriesID == nil {
		v.DownloadSingleVideo(quality)
		return
	}
	directory := videoopts.SanitizeFilename(series.SeriesName) + "_videos"
	if directory == "_videos" {
		directory = "series_" + v.Vid + "_videos"
	}
	v.SaveDir = filepath.Join(v.SaveDir, directory)
	for index, item := range series.VideoList {
		fmt.Printf("正在下载专题 %q (%d/%d)\n", series.SeriesName, index+1, len(series.VideoList))
		v.Vid = strconv.FormatInt(item.ID, 10)
		v.DownloadSingleVideo(quality)
	}
}

func (v *Video) ShowVideoInfo() {
	if !v.GetVideoInfo() {
		return
	}
	reporter := v.info.UserName
	affiliation := ""
	if len(v.info.VideoSysReporters) > 0 {
		reporter = v.info.VideoSysReporters[0].RealName
		affiliation = v.info.VideoSysReporters[0].Unit
	}
	if reporter == "" {
		reporter = "未知"
	}
	duration := timeText(v.info.VideoLength)
	date := v.info.ReportingTime
	if date == "" {
		date = v.info.ReleaseTime
	}
	abstract := v.info.Blurb
	if abstract == "" {
		abstract = "（无）"
	}
	fmt.Printf("%s (vid=%s):\n", v.info.Title, v.Vid)
	fmt.Printf("\n\t时长：%-22s讲者：%s\n", duration, reporter)
	fmt.Printf("\t日期：%-22s单位：%s\n", date, affiliation)
	fmt.Printf("\t地点：%-22s课件：%s\n", emptyAs(v.info.ReportingLocation, "未知"), yesNo(v.info.CoursewareURL != ""))
	fmt.Printf("\n\t视频简介：%s\n\n", abstract)
}

func qualityLabel(height int) string {
	switch {
	case height >= 1080:
		return "超清"
	case height >= 720:
		return "高清"
	default:
		return "标清"
	}
}

func timeText(seconds int64) string {
	if seconds <= 0 {
		return "未知"
	}
	return fmt.Sprintf("%d:%02d", seconds/60, seconds%60)
}

func emptyAs(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func yesNo(value bool) string {
	if value {
		return "有"
	}
	return "无"
}

func errorsJoin(values ...error) error {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}
