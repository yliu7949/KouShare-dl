package test

import (
	"testing"

	"github.com/yliu7949/KouShare-dl/internal/koushare"
	"github.com/yliu7949/KouShare-dl/internal/videoopts"
)

func TestSelectQualityPolicy(t *testing.T) {
	streams := []koushare.VideoStream{{Height: 1080, FileURL: "1080"}, {Height: 720, FileURL: "720"}, {Height: 480, FileURL: "480"}}
	for _, test := range []struct {
		quality string
		want    int
	}{{"high", 1080}, {"standard", 720}, {"low", 480}} {
		got, err := videoopts.ChooseStream(append([]koushare.VideoStream(nil), streams...), test.quality)
		if err != nil {
			t.Fatal(err)
		}
		if got.Height != test.want {
			t.Fatalf("quality %s selected %d, want %d", test.quality, got.Height, test.want)
		}
	}
	if _, err := videoopts.ChooseStream(streams, "4k"); err == nil {
		t.Fatal("expected invalid quality error")
	}
}

func TestSanitizeFilename(t *testing.T) {
	if got := videoopts.SanitizeFilename(`a/b:c*?"<>|`); got != "abc" {
		t.Fatalf("sanitizeFilename = %q", got)
	}
}

func TestParseVids(t *testing.T) {
	got, err := videoopts.ParseIDs("[226845,123,226845]")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "226845" || got[1] != "123" {
		t.Fatalf("parseVids returned %#v", got)
	}
	for _, invalid := range []string{"", "226845", "[1, 2]", "[]"} {
		if _, err := videoopts.ParseIDs(invalid); err == nil {
			t.Errorf("parseVids(%q) unexpectedly succeeded", invalid)
		}
	}
}
