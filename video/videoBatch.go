package video

import (
	"fmt"
	"sync"

	"github.com/yliu7949/KouShare-dl/internal/videoopts"
)

// Batch 包含多个 Video 的信息
type Batch struct {
	Vids             string
	VideoList        []Video
	SaveDir          string
	Quality          string
	IsSeries         bool
	VidPrefix        bool
	Concurrency      int
	VideoConcurrency int
}

// DownloadMultiVideos 下载多个视频
func (b *Batch) DownloadMultiVideos() {
	b.inspectVids()
	concurrency := b.VideoConcurrency
	if concurrency <= 0 {
		concurrency = 3
	}
	concurrency = min(concurrency, len(b.VideoList))
	if concurrency == 0 {
		return
	}
	jobs := make(chan Video, len(b.VideoList))
	var workers sync.WaitGroup
	for worker := 0; worker < concurrency; worker++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for item := range jobs {
				item.SaveDir = b.SaveDir
				item.VidPrefix = b.VidPrefix
				item.Concurrency = b.Concurrency
				if b.IsSeries {
					item.DownloadSeriesVideos(b.Quality)
				} else {
					item.DownloadSingleVideo(b.Quality)
				}
			}
		}()
	}
	for _, item := range b.VideoList {
		jobs <- item
	}
	close(jobs)
	workers.Wait()
}

// inspectVids 检查用户输入的视频 vid 列表，若无错误则将视频信息解析到 VideoList 中
func (b *Batch) inspectVids() {
	b.VideoList = nil
	vids, err := videoopts.ParseIDs(b.Vids)
	if err != nil {
		fmt.Println("\n" + err.Error())
		return
	}
	for _, vid := range vids {
		b.VideoList = append(b.VideoList, Video{Vid: vid})
	}
}
