package upgrade

import (
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"runtime"

	"github.com/yliu7949/KouShare-dl/internal/color"
	"github.com/yliu7949/KouShare-dl/internal/proxy"
)

var (
	ksFileName string
	ksFilePath string
	ksOldFile  string
)

func init() {
	if runtime.GOOS == "windows" {
		ksFileName = "ks.exe"
	} else {
		ksFileName = "ks"
	}

	binaryFilePath, _ := os.Executable()
	ksFilePath = filepath.Dir(binaryFilePath) + string(os.PathSeparator)
	ksOldFile = ksFilePath + ksFileName + ".old"
}

// GetLatestVersion 获取最新的KouShare-dl版本号
func GetLatestVersion() string {
	_ = os.Remove(ksOldFile)
	latestVersion, _ := net.LookupTXT("ks-version.gleamoe.com")
	return latestVersion[0]
}

// Upgrade 查询并升级KouShare-dl至最新版本
func Upgrade() {
	_ = os.Remove(ksOldFile)

	fmt.Println("正在更新KouShare-dl ...")
	if downloadBinaryFile() != nil {
		_ = os.Remove(ksFilePath + ksFileName + ".new")
		fmt.Println(color.Error("无法完整下载新版本程序，请访问 https://github.com/yliu7949/KouShare-dl/releases/latest 手动下载最新版本。"))
		return
	}
	fmt.Print(color.Done("新版本程序下载完毕。"), "\n\n")
	fileReplace()
}

func downloadBinaryFile() error {
	URL := fmt.Sprintf("https://github.com/yliu7949/KouShare-dl/releases/download/%s/%s", GetLatestVersion(), ksFileName)
	resp, err := proxy.Client.Get(URL)
	if err != nil {
		fmt.Println(err)
		return err
	}
	defer func() {
		err = resp.Body.Close()
		if err != nil {
			fmt.Println(err)
		}
	}()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err.Error())
		return err
	}

	if err = os.WriteFile(ksFilePath+ksFileName+".new", data, 0664); err != nil {
		fmt.Println(err.Error())
		return err
	}

	return nil
}
