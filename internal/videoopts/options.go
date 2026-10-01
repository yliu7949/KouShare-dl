package videoopts

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/yliu7949/KouShare-dl/internal/koushare"
)

var invalidFilenameCharacters = regexp.MustCompile(`[\\/:*?"<>|\x00-\x1f]`)

// ChooseStream selects the best available stream for a CLI quality preset.
func ChooseStream(streams []koushare.VideoStream, quality string) (koushare.VideoStream, error) {
	sort.SliceStable(streams, func(i, j int) bool {
		return streams[i].Height > streams[j].Height
	})
	limit := 1 << 30
	switch strings.ToLower(quality) {
	case "low":
		limit = 480
	case "standard":
		limit = 720
	case "high", "":
	default:
		return koushare.VideoStream{}, fmt.Errorf("未知清晰度 %q（可选 high、standard、low）", quality)
	}
	for _, stream := range streams {
		if stream.FileURL != "" && stream.Height <= limit {
			return stream, nil
		}
	}
	for index := len(streams) - 1; index >= 0; index-- {
		if streams[index].FileURL != "" {
			return streams[index], nil
		}
	}
	return koushare.VideoStream{}, fmt.Errorf("所有视频流地址均为空")
}

// SanitizeFilename removes characters that are invalid on common filesystems.
func SanitizeFilename(name string) string {
	return strings.TrimSpace(invalidFilenameCharacters.ReplaceAllString(name, ""))
}

// ParseIDs parses the legacy bracketed list accepted by ks save batch.
func ParseIDs(value string) ([]string, error) {
	match, _ := regexp.MatchString(`^\[\d+(,\d+)*]$`, value)
	if !match {
		return nil, fmt.Errorf("vids 参数格式错误，应为 [vid1,vid2,...]，vid 之间用英文逗号分隔，且参数中不能包含空格")
	}
	ids := make([]string, 0)
	seen := make(map[string]bool)
	for _, id := range strings.Split(value[1:len(value)-1], ",") {
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
	}
	return ids, nil
}
