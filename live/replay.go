package live

import (
	"fmt"
	"strings"

	"github.com/yliu7949/KouShare-dl/user"
)

// DownloadReplayVideo 下载指定直播间的快速回放视频
func (l *Live) DownloadReplayVideo() {
	if !l.getLidByRoomID() {
		return
	}
	l.checkLiveStatus()
	l.getLiveByRoomID(true)

	if l.isLive == "1" {
		fmt.Printf(`直播间正在直播中。可使用“ks record %s”命令录制该直播间。`, l.RoomID)
		return
	}

	// 直播回放有四种状态：直播结束不久回放尚未上线；已上线快速回放；已上线正式录播回放；本场直播无回放。
	switch l.isLive {
	case "0":
		fmt.Println("直播尚未开始，无快速回放。")
	case "2":
		if l.quickReplayURL != "" { //若有快速回放，则下载快速回放视频
			l.recordVOD()
		} else if l.playback == "0" {
			fmt.Println("本场直播无回放。")
		} else if l.playback == "1" {
			fmt.Println("快速回放暂未上线。")
		}
	case "3":
		fmt.Println("正式回放视频已上线。")
		if l.rtmpURL != "" {
			vid := strings.Split(l.rtmpURL, "/")[len(strings.Split(l.rtmpURL, "/"))-1]
			fmt.Printf("访问 %s 观看录播视频或使用“ks save %s”命令下载正式回放视频。\n", l.rtmpURL, vid)
		}
	default:
		fmt.Println("暂时无法下载回放视频。")
	}
}

// recordVOD 根据点播模式的m3u8文件下载快速回放视频
func (l *Live) recordVOD() {
	fmt.Println("开始下载快速回放视频...")
	str, err := user.MyGetRequest(l.quickReplayURL)
	if err != nil {
		fmt.Println("Get请求出错：", err)
		return
	}

	lines := strings.Split(str, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "#") { //忽略m3u8文件中的注释行
			continue
		}

		fmt.Println(strings.Split(line, "&")[0], "...")
		l.newTsURL = l.quickReplayURL[:strings.LastIndex(l.quickReplayURL, "/")+1] + line
		l.downloadAndMergeTsFile()
	}
	fmt.Println("快速回放视频下载完成。")
}
