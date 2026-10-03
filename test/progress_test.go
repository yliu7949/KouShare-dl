package test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/yliu7949/KouShare-dl/internal/progress"
)

func TestDownloadProgressShowsLegacyBarSpeedAndETA(t *testing.T) {
	var output bytes.Buffer
	display := progress.New("示例视频  vid=123", &output)
	display.Update(progress.Snapshot{
		Current: 50,
		Total:   100,
		Bytes:   5 * 1024 * 1024,
		Unit:    "个片段",
	})
	display.Finish(progress.Snapshot{
		Current: 100,
		Total:   100,
		Bytes:   10 * 1024 * 1024,
		Unit:    "个片段",
	})
	rendered := output.String()
	for _, want := range []string{"[>>>>>>>>", " 50%", "5.00MiB", "/s", "ETA", "50/100 个片段", "示例视频  vid=123", "100%"} {
		if !strings.Contains(rendered, want) {
			t.Errorf("progress output does not contain %q: %q", want, rendered)
		}
	}
	if !strings.HasSuffix(rendered, "\n") {
		t.Fatalf("finished progress did not end the line: %q", rendered)
	}
}
