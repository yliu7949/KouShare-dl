package video

import (
	"fmt"
	"regexp"
	"strings"
)

// Batch 包含多个 Video 的信息
type Batch struct {
	Vids      string
	VideoList []Video
	SaveDir   string
	Quality   string
	IsSeries  bool
}

// DownloadMultiVideos 下载多个视频
func (b *Batch) DownloadMultiVideos() {
	b.inspectVids()
	for _, video := range b.VideoList {
		video.SaveDir = b.SaveDir
		if b.IsSeries {
			video.DownloadSeriesVideos(b.Quality)
		} else {
			video.DownloadSingleVideo(b.Quality)
		}
	}
}

// inspectVids 检查用户输入的视频 vid 列表，若无错误则将视频信息解析到 VideoList 中
func (b *Batch) inspectVids() {
	match, _ := regexp.MatchString(`^\[\d+(,\d+)*]$`, b.Vids)
	if !match {
		fmt.Println("\nvids 参数格式错误，应为 [vid1,vid2,...]，vid 之间用英文逗号分隔，且参数中不能包含空格。")
		return
	}
	for _, vid := range strings.Split(b.Vids[1:len(b.Vids)-1], ",") {
		if vid != "" {
			var v Video
			v.Vid = vid
			if ok := v.GetVideoInfo(); !ok {
				fmt.Printf("\n获取 vid=%s 的视频信息失败。\n", vid)
				continue
			}
			b.VideoList = append(b.VideoList, v)
		}
	}
}
