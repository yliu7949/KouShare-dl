package live

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/yliu7949/KouShare-dl/internal/hls"
	"github.com/yliu7949/KouShare-dl/internal/proxy"
	"github.com/yliu7949/KouShare-dl/user"
	"github.com/yliu7949/KouShare-dl/video"
)

func (l *Live) DownloadReplayVideo() {
	if err := l.loadInfo(); err != nil {
		fmt.Println("获取直播信息失败：", err)
		return
	}
	if l.info.LiveStatus == 1 {
		fmt.Printf("直播正在进行中，可使用 ks record %s 录制。\n", l.RoomID)
		return
	}
	if l.info.VideoID != nil {
		v := video.Video{Vid: strconv.FormatInt(*l.info.VideoID, 10), SaveDir: l.SaveDir}
		v.DownloadSingleVideo("high")
		return
	}
	if l.info.PlaybackURL != "" {
		l.downloadReplayURL(l.info.PlaybackURL, "回放")
		return
	}
	if l.info.HasFastPlayback == 1 {
		l.downloadFastbacks()
		return
	}
	fmt.Println("该直播暂无可下载的回放。")
}

func (l *Live) downloadFastbacks() {
	if user.GetLoginState() != 1 {
		fmt.Println("新版快速回放播放接口要求登录，请先使用 ks login 导入网页登录凭证。")
		return
	}
	items, err := liveAPIClient().LiveFastbackList(context.Background(), l.RoomID)
	if err != nil {
		fmt.Println("获取快速回放列表失败：", err)
		return
	}
	if len(items) == 0 {
		fmt.Println("快速回放列表为空。")
		return
	}
	liveID, err := strconv.ParseInt(l.RoomID, 10, 64)
	if err != nil {
		fmt.Println("无效的直播 ID：", l.RoomID)
		return
	}
	for index, item := range items {
		fmt.Printf("正在下载快速回放 (%d/%d)：%s\n", index+1, len(items), item.Name)
		play, playErr := liveAPIClient().LiveFastbackPlay(context.Background(), liveID, item.ID, l.Password)
		if playErr != nil {
			fmt.Println("获取快速回放地址失败：", playErr)
			continue
		}
		if play.FastbackURL == "" {
			fmt.Println("该段快速回放没有可用地址。")
			continue
		}
		l.downloadReplayURL(play.FastbackURL, item.Name)
	}
}

func (l *Live) downloadReplayURL(sourceURL, suffix string) {
	if err := os.MkdirAll(l.SaveDir, 0755); err != nil {
		fmt.Println("创建保存目录失败：", err)
		return
	}
	name := sanitizeLiveFilename(l.info.Title)
	if name == "" {
		name = "replay_" + l.RoomID
	}
	suffix = sanitizeLiveFilename(suffix)
	if suffix == "" {
		suffix = "回放"
	}
	finalName := filepath.Join(l.SaveDir, name+"_"+suffix+".ts")
	if _, err := os.Stat(finalName); err == nil {
		fmt.Println("回放已下载，自动跳过：", finalName)
		return
	}
	dst, err := os.OpenFile(finalName+".tmp", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		fmt.Println("创建回放文件失败：", err)
		return
	}
	if strings.Contains(strings.ToLower(sourceURL), ".m3u8") {
		err = (hls.Downloader{Client: &proxy.Client}).Download(context.Background(), sourceURL, dst, nil)
	} else {
		err = fmt.Errorf("回放地址不是受支持的 HLS 清单")
	}
	closeErr := dst.Close()
	if err != nil || closeErr != nil {
		fmt.Println("回放下载失败：", err)
		return
	}
	if err := os.Rename(finalName+".tmp", finalName); err != nil {
		fmt.Println("完成回放文件失败：", err)
		return
	}
	fmt.Println("回放下载完成：", finalName)
}
