package live

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/yliu7949/KouShare-dl/internal/hls"
	"github.com/yliu7949/KouShare-dl/internal/koushare"
	"github.com/yliu7949/KouShare-dl/internal/proxy"
	"github.com/yliu7949/KouShare-dl/user"
)

type Live struct {
	RoomID   string
	Password string
	SaveDir  string

	info   koushare.LiveInfo
	hlsURL string
}

func liveAPIClient() *koushare.Client {
	return koushare.NewClient(&proxy.Client, user.AccessToken())
}

func (l *Live) loadInfo() error {
	info, err := liveAPIClient().LiveInfo(context.Background(), l.RoomID)
	if err != nil {
		return err
	}
	l.info = info
	return nil
}

func (l *Live) loadStream(chooseHighQuality bool) error {
	result, err := liveAPIClient().LiveStream(context.Background(), l.RoomID, l.Password)
	if err != nil {
		return err
	}
	if result.Authorization.Result != "" && result.Authorization.Result != "AUTHENTICATED" {
		return fmt.Errorf("直播访问校验失败：%s", result.Authorization.Result)
	}
	var streams []koushare.LiveStream
	for _, group := range result.StreamURLs {
		if strings.EqualFold(group.Type, "HLS") {
			streams = append(streams, group.List...)
		}
	}
	if len(streams) == 0 {
		return fmt.Errorf("API 未返回 HLS 直播流")
	}
	sort.SliceStable(streams, func(i, j int) bool {
		return streams[i].QualityValue > streams[j].QualityValue
	})
	for _, stream := range streams {
		if stream.URL == "" {
			continue
		}
		if !chooseHighQuality && stream.QualityValue > 720 {
			continue
		}
		if stream.IsLogin && user.GetLoginState() != 1 {
			continue
		}
		l.hlsURL = stream.URL
		return nil
	}
	return fmt.Errorf("没有当前登录状态可用的直播清晰度")
}

func (l *Live) WaitAndRecordTheLive(liveTime string, autoMerge bool) {
	if liveTime != "" {
		parsedTime, err := time.ParseInLocation("2006-01-02 15:04:05", liveTime, time.Local)
		if err != nil {
			fmt.Println("时间解析出错：", err)
			return
		}
		if delay := time.Until(parsedTime); delay > 0 {
			fmt.Println("设定的直播时间为：", parsedTime)
			time.Sleep(delay)
		}
	}
	if err := l.loadInfo(); err != nil {
		fmt.Println("获取直播信息失败：", err)
		return
	}
	if l.info.LiveStatus == 0 {
		start, err := time.ParseInLocation("2006-01-02 15:04:05", l.info.LiveStart, time.Local)
		if err == nil && time.Until(start) > 0 {
			fmt.Printf("直播尚未开始，将于 %s 自动开始录制。\n", l.info.LiveStart)
			time.Sleep(time.Until(start))
			if err := l.loadInfo(); err != nil {
				fmt.Println("刷新直播状态失败：", err)
				return
			}
		}
	}
	if l.info.LiveStatus != 1 {
		fmt.Println(liveStatusMessage(l.info))
		return
	}
	if err := l.loadStream(true); err != nil {
		fmt.Println("获取直播流失败：", err)
		return
	}
	if err := os.MkdirAll(l.SaveDir, 0755); err != nil {
		fmt.Println("创建保存目录失败：", err)
		return
	}

	name := sanitizeLiveFilename(l.info.Title)
	if name == "" {
		name = "live_" + l.RoomID
	}
	name += "_" + strings.NewReplacer(":", "_", " ", "_").Replace(l.info.LiveStart)
	finalName := filepath.Join(l.SaveDir, name+".ts")
	temporaryName := finalName + ".tmp"
	dst, err := os.OpenFile(temporaryName, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		fmt.Println("创建直播文件失败：", err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go l.stopWhenEnded(ctx, cancel)
	fmt.Println("开始录制直播；按 Ctrl+C 可停止。")
	err = (hls.Downloader{Client: &proxy.Client}).Record(ctx, l.hlsURL, dst, 2*time.Second, func(total int) {
		fmt.Printf("\r已录制 %d 个片段", total)
	})
	closeErr := dst.Close()
	fmt.Println()
	if err != nil {
		fmt.Println("录制失败：", err)
		return
	}
	if closeErr != nil {
		fmt.Println("关闭直播文件失败：", closeErr)
		return
	}
	if err := os.Rename(temporaryName, finalName); err != nil {
		fmt.Println("完成直播文件失败：", err)
		return
	}
	if !autoMerge {
		fmt.Println("录制完成：", finalName)
	} else {
		fmt.Println("录制完成（当前 API 已直接合并为单个 TS 文件）：", finalName)
	}
}

func (l *Live) stopWhenEnded(ctx context.Context, cancel context.CancelFunc) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			info, err := liveAPIClient().LiveInfo(ctx, l.RoomID)
			if err == nil && info.LiveStatus != 1 {
				cancel()
				return
			}
		}
	}
}

func (l *Live) ShowLiveInfo() {
	if err := l.loadInfo(); err != nil {
		fmt.Println("获取直播信息失败：", err)
		return
	}
	organizers := make([]string, 0, len(l.info.LiveMajorOrganizers))
	for _, organizer := range l.info.LiveMajorOrganizers {
		if organizer.Name != "" {
			organizers = append(organizers, organizer.Name)
		}
	}
	if len(organizers) == 0 {
		organizers = append(organizers, "未知")
	}
	notice := l.info.Notice
	if notice == "" {
		notice = "（无）"
	}
	fmt.Printf("%s (liveID=%s, roomNo=%s):\n", l.info.Title, l.RoomID, l.info.RoomNo)
	fmt.Printf("\n\t直播状态：%-17s主办单位：%s\n", liveStatusName(l.info.LiveStatus), strings.Join(organizers, "、"))
	fmt.Printf("\t开播时间：%-22s有无快速回放：%s\n", l.info.LiveStart, yesNo(l.info.HasFastPlayback == 1))
	fmt.Printf("\t观看次数：%-22d专题：%s\n", l.info.Views, empty(l.info.SubjectName, "（无）"))
	fmt.Printf("\n\t最新通知：%s\n", notice)
}

func liveStatusName(status int) string {
	switch status {
	case 0:
		return "直播未开始"
	case 1:
		return "正在直播中"
	case 2:
		return "直播已结束"
	case 3:
		return "录播已上线"
	default:
		return "未知状态"
	}
}

func liveStatusMessage(info koushare.LiveInfo) string {
	switch info.LiveStatus {
	case 0:
		return "直播尚未开始。"
	case 2:
		if info.HasFastPlayback == 1 {
			return "直播已结束，快速回放已上线。"
		}
		return "直播已结束，暂无快速回放。"
	case 3:
		if info.VideoID != nil {
			return fmt.Sprintf("正式回放已上线，可使用 ks save %d 下载。", *info.VideoID)
		}
		return "正式回放已上线。"
	default:
		return "直播当前不可录制。"
	}
}

func MergeTsFiles(dir string, dstFileName string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		fmt.Println("读取目录失败：", err)
		return
	}
	var files []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.EqualFold(filepath.Ext(entry.Name()), ".ts") && entry.Name() != dstFileName {
			files = append(files, filepath.Join(dir, entry.Name()))
		}
	}
	sort.Strings(files)
	if len(files) == 0 {
		fmt.Println("没有需要合并的视频片段。")
		return
	}
	destination := filepath.Join(dir, dstFileName)
	dst, err := os.OpenFile(destination, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		fmt.Println("创建合并文件失败：", err)
		return
	}
	writer := bufio.NewWriter(dst)
	for _, filename := range files {
		data, readErr := os.ReadFile(filename)
		if readErr != nil {
			_ = dst.Close()
			fmt.Println("读取片段失败：", readErr)
			return
		}
		if _, err := writer.Write(data); err != nil {
			_ = dst.Close()
			fmt.Println("写入合并文件失败：", err)
			return
		}
	}
	if err := writer.Flush(); err != nil {
		_ = dst.Close()
		fmt.Println("写入合并文件失败：", err)
		return
	}
	if err := dst.Close(); err != nil {
		fmt.Println("关闭合并文件失败：", err)
		return
	}
	fmt.Println("合并完成：", destination)
}

func sanitizeLiveFilename(name string) string {
	invalid := regexp.MustCompile(`[\\/:*?"<>|\x00-\x1f]`)
	return strings.TrimSpace(invalid.ReplaceAllString(name, ""))
}

func yesNo(value bool) string {
	if value {
		return "有"
	}
	return "无"
}

func empty(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
