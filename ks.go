package main

import (
	"github.com/spf13/cobra"

	"github.com/yliu7949/KouShare-dl/live"
	"github.com/yliu7949/KouShare-dl/slide"
	"github.com/yliu7949/KouShare-dl/video"
)

func main() {
	var v video.Video
	var path string
	var isSeries bool

	var cmdInfo = &cobra.Command{
		Use:   "info [vid]",
		Short: "获取视频的基本信息",
		Long:  `获取视频的基本信息，如讲者、拍摄日期、视频大小、视频摘要等内容.`,
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			v.Vid = args[0]
			v.ShowVideoInfo()
		},
	}
	var cmdSave = &cobra.Command{
		Use:   "save [vid]",
		Short: "保存指定vid的视频",
		Long:  `保存指定vid的视频到本地计算机，付费视频除外.`,
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			v.Vid = args[0]
			if path[len(path)-1:] != "\\" && path[len(path)-1:] != "/" {
				path = path + "\\"
			}
			v.SaveDir = path
			if isSeries {
				v.DownloadSeriesVideos()
			} else {
				v.DownloadSingleVideo()
			}
		},
	}
	cmdSave.Flags().StringVarP(&path, "path", "p", `.\`, "指定保存视频的路径")
	cmdSave.Flags().BoolVarP(&isSeries, "series", "s", false, "指定是否下载系列视频")

	var l live.Live
	var liveTime string //开播时间，格式应为"2006-01-02 15:04:05"
	var chooseAutoMerge bool
	var dstFileName string

	var cmdRecord = &cobra.Command{
		Use:   "record [roomID]",
		Short: "录制指定直播间ID的直播",
		Long:  `录制指定直播间ID的直播.`,
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			l.RoomID = args[0]
			if path[len(path)-1:] != "\\" && path[len(path)-1:] != "/" {
				path = path + "\\"
			}
			l.SaveDir = path
			l.WaitAndRecordTheLive(liveTime, chooseAutoMerge)
		},
	}
	cmdRecord.Flags().StringVarP(&path, "path", "p", `.\`, "指定保存视频的路径")
	cmdRecord.Flags().StringVarP(&liveTime, "at", "@", "", `开播时间，格式为"2006-01-02 15:04:05"`)
	cmdRecord.Flags().BoolVarP(&chooseAutoMerge, "autoMerge", "a", false, "指定是否自动合并下载的视频片段文件")
	var cmdMerge = &cobra.Command{
		Use:   "merge [directory]",
		Short: "合并下载的视频片段文件",
		Long:  `合并下载的视频片段文件(.ts)为一个视频文件(.ts)，[directory]参数为存放视频片段文件的文件夹的路径，若为空则默认为当前路径.`,
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				path = "./"
			} else {
				path = args[0]
				if path[len(path)-1:] != "\\" && path[len(path)-1:] != "/" {
					path = path + "\\"
				}
			}
			live.MergeTsFiles(path, dstFileName)
		},
	}
	cmdMerge.Flags().StringVarP(&dstFileName, "name", "n", `recorded Video File.ts`, "指定合并后视频文件的名字(xxx.ts)")

	var s slide.Slide
	var qpdfBinPath string
	var cmdSlide = &cobra.Command{
		Use:   "slide [vid]",
		Short: "下载指定vid的视频对应的课件",
		Long:  `下载指定vid的视频对应的课件.`,
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			s.Vid = args[0]
			if path[len(path)-1:] != "\\" && path[len(path)-1:] != "/" {
				path = path + "\\"
			}
			s.SaveDir = path
			s.DownloadSlides(isSeries, qpdfBinPath)
		},
	}
	cmdSlide.Flags().StringVarP(&path, "path", "p", `.\`, "指定保存课件的路径")
	cmdSlide.Flags().BoolVarP(&isSeries, "series", "s", false, "指定是否下载整个系列的所有课件")
	cmdSlide.Flags().StringVar(&qpdfBinPath, "qpdf-bin", "", "指定qpdf的bin文件夹所在的路径")

	var rootCmd = &cobra.Command{Use: "ks"}
	rootCmd.AddCommand(cmdInfo, cmdSave, cmdRecord, cmdMerge, cmdSlide)
	rootCmd.Version = "v0.5"
	_ = rootCmd.Execute()
}
