package main

import (
	"fmt"
	"net"

	//"github.com/pkg/profile"
	"github.com/spf13/cobra"
	. "github.com/yliu7949/KouShare-dl/cmd/ks"
)

const version = "v0.8.5"

func main() {
	//defer profile.Start().Stop()
	var rootCmd = &cobra.Command{Use: "ks"}
	rootCmd.AddCommand(InfoCmd(), SaveCmd(), RecordCmd(), MergeCmd(), SlideCmd(), LoginCmd(), LogoutCmd(), VersionCmd())
	rootCmd.SetVersionTemplate(`{{printf "KouShare-dl %s\n" .Version}}`)
	rootCmd.Version = version
	_ = rootCmd.Execute()
}

func VersionCmd() *cobra.Command {
	var cmdVersion = &cobra.Command{
		Use:   "version",
		Short: "输出版本号，并检查最新版本",
		Long:  `输出KouSHare-dl的版本号，并检查最新版本`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("KouShare-dl", version)
			latestVersion, _ := net.LookupTXT("ks-version.gleamoe.com")
			if latestVersion[0] != version {
				fmt.Println("发现新版本：KouShare-dl", latestVersion[0])
				fmt.Println("请访问 https://github.com/yliu7949/KouShare-dl/releases/latest 下载最新版本。")
			} else {
				fmt.Println("当前已是最新版本。")
			}
		},
	}

	return cmdVersion
}
