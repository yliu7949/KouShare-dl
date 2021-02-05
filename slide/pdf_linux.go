package slide

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

var qpdfBinPath = ""

func removeWM(fileName string) {
	_ = decompressPdfFile(fileName)
	offsetMap := getOffsetFromXref(fileName)
	objectSlice := getObjectSlice(fileName)

	pdfFile, err := os.OpenFile(fileName, os.O_RDWR, 0666)
	if err != nil {
		fmt.Println(err)
		return
	}
	for _, objID := range objectSlice {
		_, _ = pdfFile.Seek(offsetMap[objID], 0)
		id, _ := strconv.Atoi(objID)
		buff := make([]byte, offsetMap[strconv.Itoa(id+1)]-offsetMap[objID]-1)
		_, err = pdfFile.Read(buff)
		if err != nil {
			fmt.Println(err)
			return
		}

		var result string
		reg := regexp.MustCompile(` \d+ Tf`)
		if reg.MatchString(string(buff)) {
			result = reg.ReplaceAllString(string(buff), " 00 Tf")
		} else {
			reg = regexp.MustCompile(`/.+Do`)
			strSlice := reg.FindStringSubmatch(string(buff))
			if len(strSlice) != 0 {
				result = reg.ReplaceAllString(string(buff), strings.Repeat("\n", len(strSlice[0])))
			} else {
				fmt.Println("obj:", objID, "未匹配，跳过该对象。")
				continue
			}
		}
		_, _ = pdfFile.Seek(offsetMap[objID], 0)
		_, err = pdfFile.Write([]byte(result))
		if err != nil {
			fmt.Println(err)
			return
		}
	}
	_ = pdfFile.Close()
	_ = compressPdfFile(fileName)
	//如果在解压pdf文件时得到了warning，则会生成以~qpdf-orig结尾的备份文件；如果解压时没有警告和错误，则不会生成备份文件
	if _, err := os.Stat(fileName + ".~qpdf-orig"); err == nil {
		err = os.Remove(fileName + ".~qpdf-orig") //删除备份文件
		if err != nil {
			return
		}
	}
}

//获取解压后的pdf文件中的obj列表
func getObjectSlice(fileName string) (objectSlice []string) {
	cmd := exec.Command(qpdfBinPath+"qpdf", "--show-pages", fileName)
	output, err := cmd.Output()
	if err != nil {
		fmt.Println(err)
		return nil
	}
	showPages := string(output) + "p"

	//通过正则表达式获取pdf文件中每一页content中的最后一个对象的ID，并存储在objectSlice变量里
	reg := regexp.MustCompile(`content:\s+(?:.+\s+){2,5}\s+(?P<x>\d+).+\np`)
	if !reg.MatchString(showPages) {
		fmt.Println("Not match.")
		return nil
	}
	result := reg.FindAllStringSubmatch(showPages, -1)
	for i := 0; i < len(result); i++ {
		objectSlice = append(objectSlice, result[i][1])
	}
	return objectSlice
}

func getOffsetFromXref(fileName string) map[string]int64 {
	cmd := exec.Command(qpdfBinPath+"qpdf", "--show-xref", fileName)
	output, err := cmd.Output()
	if err != nil {
		fmt.Println(err)
		return nil
	}
	showXref := string(output)

	//通过正则表达式获取pdf文件中的每一个obj及其距文档开头的offset，并存储在offsetMap变量里
	reg := regexp.MustCompile(`(?P<object>\d+).+= (?P<offset>\d+)`)
	if !reg.MatchString(showXref) {
		fmt.Println("Not match.")
		return nil
	}
	result := reg.FindAllStringSubmatch(showXref, -1)
	offsetMap := make(map[string]int64)
	for i := 0; i < len(result); i++ {
		offset, _ := strconv.Atoi(result[i][2])
		offsetMap[result[i][1]] = int64(offset)
	}
	return offsetMap
}

func decompressPdfFile(fileName string) error {
	cmd := exec.Command(qpdfBinPath+"qpdf", "--qdf", "--replace-input", fileName)
	err := cmd.Run()
	if err != nil {
		return err
	}
	return nil
}

func compressPdfFile(fileName string) error {
	cmd := exec.Command(qpdfBinPath+"qpdf", "--replace-input", fileName)
	err := cmd.Run()
	if err != nil {
		return err
	}
	return nil
}
