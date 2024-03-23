package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/tidwall/gjson"
	"github.com/yliu7949/KouShare-dl/user"
)

func TestMyGetRequest(t *testing.T) {
	URL := "https://core.api.koushare.com/iam/userLogin/checkLogin"
	result, err := user.MyGetRequest(URL)
	if err != nil {
		t.Error(err)
		return
	}
	output, _ := formatJSONString(result)
	fmt.Println(output)
}

func TestMyPostRequest(t *testing.T) {
	URL := "https://core.api.koushare.com/video/v1/video/checkVideoAuth"
	data := map[string]string{
		"id": "83047",
	}
	result, err := user.MyPostRequest(URL, data)
	if err != nil {
		t.Error(err)
		return
	}
	if gjson.Get(result, "success").String() != "true" {
		t.Error(result)
	} else {
		// 若测试成功，则格式化输出返回的 JSON
		output, _ := formatJSONString(result)
		fmt.Println(output)
	}
}

// formatJSONString 格式化JSON字符串
func formatJSONString(inputJSON string) (string, error) {
	var out bytes.Buffer
	err := json.Indent(&out, []byte(inputJSON), "", "    ")
	if err != nil {
		return "", err
	}
	return out.String(), nil
}
