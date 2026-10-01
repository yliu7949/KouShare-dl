package ks

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/yliu7949/KouShare-dl/live"
	"github.com/yliu7949/KouShare-dl/slide"
	"github.com/yliu7949/KouShare-dl/user"
	"github.com/yliu7949/KouShare-dl/video"
)

var path string

// InfoCmd 获取视频或直播的基本信息
func InfoCmd() *cobra.Command {
	var v video.Video
	var l live.Live
	var isLive bool
	var cmdInfo = &cobra.Command{
		Use:   "info [vid]",
		Short: "获取视频或直播的基本信息",
		Long:  `获取视频的基本信息，如讲者、拍摄日期、视频摘要等内容；使用 --live 获取直播信息。`,
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if isLive {
				l.RoomID = args[0]
				l.ShowLiveInfo()
			} else {
				v.Vid = args[0]
				v.ShowVideoInfo()
			}
		},
	}
	cmdInfo.Flags().BoolVarP(&isLive, "live", "l", false, "将 ID 作为新版直播 ID 查询")

	return cmdInfo
}

var quality string
var isSeries bool
var vidPrefix bool
var segmentConcurrency int
var videoConcurrency int

// SaveCmd 保存指定vid的视频
func SaveCmd() *cobra.Command {
	var v video.Video
	var cmdSave = &cobra.Command{
		Use:   "save [vid]",
		Short: "保存指定vid的视频",
		Long:  `保存指定vid的视频到本地计算机，未登录时仅可下载标清视频，登录后可以下载更高清晰度的免费视频. 此外仅能下载已购买的付费视频.`,
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			v.Vid = args[0]
			v.SaveDir = normalizedDirectory(path)
			v.VidPrefix = vidPrefix
			v.Concurrency = segmentConcurrency
			if isSeries {
				v.DownloadSeriesVideos(quality)
			} else {
				v.DownloadSingleVideo(quality)
			}
		},
		Aliases: []string{"video"},
	}
	cmdSave.PersistentFlags().StringVarP(&path, "path", "p", `.`, "指定保存视频的路径")
	cmdSave.PersistentFlags().BoolVarP(&isSeries, "series", "s", false, "指定是否下载专题视频")
	cmdSave.PersistentFlags().StringVarP(&quality, "quality", "q", `high`, "指定下载视频的清晰度（high、standard或low）")
	cmdSave.PersistentFlags().BoolVarP(&vidPrefix, "vidPrefix", "v", false, "指定是否使用vid作为保存视频文件名的前缀")
	cmdSave.PersistentFlags().IntVarP(&segmentConcurrency, "concurrency", "c", 12, "单个视频同时下载的 HLS 片段数")
	cmdSave.PersistentFlags().IntVar(&videoConcurrency, "video-concurrency", 3, "批量模式同时下载的视频数")
	cmdSave.AddCommand(SaveBatchCmd())

	return cmdSave
}

// SaveBatchCmd 批量保存指定vid和清晰度的视频，是save命令的子命令
func SaveBatchCmd() *cobra.Command {
	var b video.Batch
	var cmdSaveBatch = &cobra.Command{
		Use:   "batch [vids]",
		Short: "批量保存指定vid的视频",
		Long:  `批量保存指定vid的视频到本地计算机，可以下载不同清晰度的免费视频，但仅能下载已购买的付费视频.`,
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			b.Vids = args[0]
			b.SaveDir = normalizedDirectory(path)
			b.Quality = quality
			b.IsSeries = isSeries
			b.VidPrefix = vidPrefix
			b.Concurrency = segmentConcurrency
			b.VideoConcurrency = videoConcurrency
			b.DownloadMultiVideos()
		},
	}

	return cmdSaveBatch
}

// RecordCmd 录制指定直播间ID的直播
func RecordCmd() *cobra.Command {
	var l live.Live
	var liveTime string //开播时间，格式应为"2006-01-02 15:04:05"
	var autoMerge bool
	var replay bool
	var password string

	var cmdRecord = &cobra.Command{
		Use:   "record [roomID]",
		Short: "录制指定直播间ID的直播",
		Long:  `录制指定直播间ID的直播.`,
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			l.RoomID = args[0]
			l.SaveDir = normalizedDirectory(path)
			l.Password = password
			if !replay {
				l.WaitAndRecordTheLive(liveTime, autoMerge)
			} else {
				l.DownloadReplayVideo()
			}
		},
		Aliases: []string{"live"},
	}
	cmdRecord.Flags().StringVarP(&path, "path", "p", `.`, "指定保存视频的路径")
	cmdRecord.Flags().StringVarP(&liveTime, "at", "@", "", `开播时间，格式为"2006-01-02 15:04:05"`)
	cmdRecord.Flags().BoolVarP(&autoMerge, "autoMerge", "a", false, "指定是否自动合并下载的视频片段文件")
	cmdRecord.Flags().BoolVarP(&replay, "replay", "r", false, "指定是否下载直播间快速回放视频")
	cmdRecord.Flags().StringVar(&password, "password", "", "指定直播间密码")

	return cmdRecord
}

// MergeCmd 合并下载的视频片段文件
func MergeCmd() *cobra.Command {
	var dstFileName string
	var cmdMerge = &cobra.Command{
		Use:   "merge [directory]",
		Short: "合并下载的视频片段文件",
		Long:  `合并下载的视频片段文件(.ts)为一个视频文件(.ts)，[directory]参数为存放视频片段文件的文件夹的路径，若为空则默认为当前路径.`,
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				path = "."
			} else {
				path = normalizedDirectory(args[0])
			}
			live.MergeTsFiles(path, dstFileName)
		},
	}
	cmdMerge.Flags().StringVarP(&dstFileName, "name", "n", `recorded Video File.ts`, "指定合并后视频文件的名字(xxx.ts)")

	return cmdMerge
}

// SlideCmd 下载指定vid的视频对应的课件
func SlideCmd() *cobra.Command {
	var s slide.Slide
	var isSeries bool
	var qpdfBinPath string
	var cmdSlide = &cobra.Command{
		Use:   "slide [vid]",
		Short: "下载指定vid的视频对应的课件",
		Long:  `下载指定vid的视频对应的课件.`,
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			s.Vid = args[0]
			s.SaveDir = normalizedDirectory(path)
			if qpdfBinPath != "" {
				qpdfBinPath = normalizedDirectory(qpdfBinPath) + string(os.PathSeparator)
			}
			s.QpdfPath = qpdfBinPath
			if isSeries {
				s.DownloadSeriesSlides()
			} else {
				s.DownloadSingleSlide()
			}
		},
	}
	cmdSlide.Flags().StringVarP(&path, "path", "p", `.`, "指定保存课件的路径")
	cmdSlide.Flags().BoolVarP(&isSeries, "series", "s", false, "指定是否下载整个专题的所有课件")
	cmdSlide.Flags().StringVar(&qpdfBinPath, "qpdf-bin", "", "指定qpdf的bin文件夹所在的路径")

	return cmdSlide
}

// LoginCmd 打开浏览器并保存用户正常登录后获得的 Cookie。
func LoginCmd() *cobra.Command {
	var u user.User
	var accessToken string
	var refreshToken string
	var harPath string
	var browserPath string
	var loginTimeout time.Duration
	var validFor time.Duration
	var cmdLogin = &cobra.Command{
		Use:   "login [phone number]",
		Short: "登录蔻享账户并保存凭证",
		Long: `不带参数时打开专用 Chrome 窗口进行网页登录，并自动保存登录 Cookie。
带手机号时保留旧版终端习惯：在正常 Chrome 中完成人机验证并发送短信，然后回到命令行
输入 6 位短信验证码。程序不会尝试绕过或代替网站要求的人机验证。`,
		Args: cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			phone := ""
			if len(args) == 1 {
				phone = args[0]
			}
			var err error
			if harPath != "" {
				err = u.ImportTokensFromHAR(harPath)
			} else if accessToken != "" || refreshToken != "" {
				err = u.ImportTokens(accessToken, refreshToken, validFor)
			} else if os.Getenv(user.AccessTokenEnv) != "" {
				err = u.ImportTokensFromEnvironment(validFor)
			} else {
				loginContext, cancel := context.WithTimeout(cmd.Context(), loginTimeout)
				defer cancel()
				if phone != "" {
					err = u.LoginWithPhone(loginContext, phone, browserPath, os.Stdin, cmd.OutOrStdout())
				} else {
					fmt.Fprintln(cmd.OutOrStdout(), "Chrome 已打开。请在网页中完成登录，命令行正在等待登录 Cookie……")
					err = u.LoginWithBrowser(loginContext, browserPath)
				}
			}
			if err != nil {
				fmt.Println("登录失败：", err)
				return
			}
			fmt.Println("登录成功，凭证已安全保存。")
		},
	}
	cmdLogin.Flags().StringVar(&accessToken, "access-token", "", "网页端 accessToken（建议改用环境变量）")
	cmdLogin.Flags().StringVar(&refreshToken, "refresh-token", "", "网页端 refreshToken（可选）")
	cmdLogin.Flags().StringVar(&harPath, "har", "", "兼容选项：从登录后的 HAR 导入令牌")
	cmdLogin.Flags().StringVar(&browserPath, "browser-path", "", "Chrome/Chromium 可执行文件路径（默认自动查找）")
	cmdLogin.Flags().DurationVar(&loginTimeout, "timeout", 10*time.Minute, "等待浏览器登录的最长时间")
	cmdLogin.Flags().DurationVar(&validFor, "expires-in", 30*24*time.Hour, "令牌剩余有效期")

	return cmdLogin
}

// LogoutCmd 退出登录并删除保存在本地的登录凭证文件
func LogoutCmd() *cobra.Command {
	var u user.User
	var cmdLogout = &cobra.Command{
		Use:   "logout",
		Short: "退出登录",
		Long:  `退出登录并删除保存在本地的登录凭证文件.`,
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			u.Logout()
		},
	}

	return cmdLogout
}

// CleanCmd 清理指定目录下的所有临时文件
func CleanCmd() *cobra.Command {
	var quiet bool
	var cmdClean = &cobra.Command{
		Use:   "clean",
		Short: "清理指定目录下的所有tmp临时文件",
		Long:  `清理指定目录下的所有tmp临时文件.`,
		Run: func(cmd *cobra.Command, args []string) {
			path = normalizedDirectory(path)
			files, err := os.ReadDir(path)
			if err != nil {
				fmt.Println("读取目录错误：", err.Error())
				return
			}

			for _, file := range files {
				if !file.IsDir() && strings.HasSuffix(file.Name(), ".tmp") {
					err := os.Remove(filepath.Join(path, file.Name()))
					if err != nil {
						fmt.Println("删除文件错误：", err.Error())
						continue
					}
					if !quiet {
						fmt.Println("已清理文件：", file.Name())
					}
				}
			}
		},
	}
	cmdClean.Flags().StringVarP(&path, "path", "p", `.`, "指定清理临时文件的路径")
	cmdClean.Flags().BoolVarP(&quiet, "quiet", "q", false, "指定是否不输出清理过程中的信息")
	return cmdClean
}

func normalizedDirectory(value string) string {
	if strings.TrimSpace(value) == "" {
		return "."
	}
	return filepath.Clean(value)
}
