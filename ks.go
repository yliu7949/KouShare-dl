package main

import (
	"fmt"

	//"github.com/pkg/profile"
	"github.com/spf13/cobra"
	ks "github.com/yliu7949/KouShare-dl/cmd/ks"
	"github.com/yliu7949/KouShare-dl/internal/color"
	"github.com/yliu7949/KouShare-dl/internal/proxy"
	"github.com/yliu7949/KouShare-dl/internal/upgrade"
)

const version = "v0.9.0"

func main() {
	//defer profile.Start().Stop()
	var noColor bool
	var proxyURL string
	var rootCmd = &cobra.Command{
		Use: "ks",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			color.DisableColor(noColor)
			proxy.EnableProxy(proxyURL)
		},
	}
	rootCmd.AddCommand(ks.InfoCmd(), ks.SaveCmd(), ks.RecordCmd(), ks.MergeCmd(), ks.SlideCmd(),
		ks.LoginCmd(), ks.LogoutCmd(), VersionCmd(), UpgradeCmd())
	rootCmd.SetVersionTemplate(`{{printf "KouShare-dl %s\n" .Version}}`)
	rootCmd.Version = version

	rootCmd.PersistentFlags().BoolVar(&noColor, "nocolor", false, "指定是否不使用彩色输出")
	rootCmd.PersistentFlags().StringVarP(&proxyURL, "proxy", "P", "", "指定使用的http/https/socks5代理服务地址")
	_ = rootCmd.Execute()
}

// VersionCmd 输出KouSHare-dl的版本号，并检查最新版本
func VersionCmd() *cobra.Command {
	var cmdVersion = &cobra.Command{
		Use:   "version",
		Short: "输出版本号，并检查最新版本",
		Long:  `输出KouSHare-dl的版本号，并检查最新版本`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(color.Emphasize("KouShare-dl " + version))
			latestVersion := upgrade.GetLatestVersion()
			if latestVersion != version {
				fmt.Println("发现新版本：KouShare-dl", latestVersion)
				fmt.Println("使用ks upgrade命令升级至最新版本，或访问 https://github.com/yliu7949/KouShare-dl/releases/latest 手动下载更新。")
			} else {
				fmt.Println("当前已是最新版本。")
			}
		},
	}

	return cmdVersion
}

// UpgradeCmd 查询并升级KouShare-dl至最新版本
func UpgradeCmd() *cobra.Command {
	var cmdUpgrade = &cobra.Command{
		Use:   "upgrade",
		Short: "升级为最新版本",
		Long:  `查询并升级至最新版本.`,
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			if upgrade.GetLatestVersion() != version {
				upgrade.Upgrade()
			} else {
				fmt.Println("当前已是最新版本。")
			}
		},
	}

	return cmdUpgrade
}
