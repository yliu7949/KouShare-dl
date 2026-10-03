package test

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/yliu7949/KouShare-dl/internal/hls"
)

func TestDownloadAES128HLS(t *testing.T) {
	key := []byte("0123456789abcdef")
	iv, _ := hex.DecodeString("434425b6273a8028613fb55f616914db")
	parts := [][]byte{[]byte("first transport stream part"), []byte("second transport stream part")}

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	mux.HandleFunc("/master.m3u8", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "#EXTM3U\n#EXT-X-STREAM-INF:BANDWIDTH=1000\nmedia/index.m3u8\n")
	})
	mux.HandleFunc("/media/index.m3u8", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, "#EXTM3U\n#EXT-X-MEDIA-SEQUENCE:0\n#EXT-X-KEY:METHOD=AES-128,URI=%q,IV=0x%s\n#EXTINF:1,\n0.ts\n#EXTINF:1,\n1.ts\n#EXT-X-ENDLIST\n", server.URL+"/key", hex.EncodeToString(iv))
	})
	mux.HandleFunc("/key", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(key) })
	for index := range parts {
		index := index
		mux.HandleFunc(fmt.Sprintf("/media/%d.ts", index), func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write(encrypt(parts[index], key, iv))
		})
	}

	var output bytes.Buffer
	var updates []hls.Progress
	err := (hls.Downloader{Client: server.Client()}).Download(context.Background(), server.URL+"/master.m3u8", &output, func(current hls.Progress) {
		if current.Total != len(parts) {
			t.Fatalf("total = %d", current.Total)
		}
		updates = append(updates, current)
	})
	if err != nil {
		t.Fatal(err)
	}
	want := append(append([]byte(nil), parts[0]...), parts[1]...)
	if !bytes.Equal(output.Bytes(), want) {
		t.Fatalf("output = %q, want %q", output.Bytes(), want)
	}
	if len(updates) != 2 || updates[1].Completed != 2 || updates[1].DownloadedBytes != int64(len(want)) {
		t.Fatalf("progress = %#v", updates)
	}
}

func TestParsePlaylistUsesSequenceAsIV(t *testing.T) {
	key := []byte("0123456789abcdef")
	iv := make([]byte, aes.BlockSize)
	iv[len(iv)-1] = 7
	want := []byte("sequence-derived initialization vector")

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	mux.HandleFunc("/media.m3u8", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "#EXTM3U\n#EXT-X-MEDIA-SEQUENCE:7\n#EXT-X-KEY:METHOD=AES-128,URI=key.bin\n#EXTINF:1,\na.ts\n#EXT-X-ENDLIST\n")
	})
	mux.HandleFunc("/key.bin", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(key) })
	mux.HandleFunc("/a.ts", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(encrypt(want, key, iv)) })

	var output bytes.Buffer
	err := (hls.Downloader{Client: server.Client()}).Download(context.Background(), server.URL+"/media.m3u8", &output, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(output.Bytes(), want) {
		t.Fatalf("output = %q, want %q", output.Bytes(), want)
	}
}

func TestDownloadRetriesMalformedEncryptedSegment(t *testing.T) {
	key := []byte("0123456789abcdef")
	iv := []byte("abcdef0123456789")
	want := []byte("transport stream after transient response")
	requests := 0

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	mux.HandleFunc("/media.m3u8", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, "#EXTM3U\n#EXT-X-KEY:METHOD=AES-128,URI=%q,IV=0x%s\n#EXTINF:1,\nsegment.ts\n#EXT-X-ENDLIST\n", server.URL+"/key", hex.EncodeToString(iv))
	})
	mux.HandleFunc("/key", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(key) })
	mux.HandleFunc("/segment.ts", func(w http.ResponseWriter, _ *http.Request) {
		requests++
		if requests == 1 {
			_, _ = w.Write([]byte("temporary invalid response"))
			return
		}
		_, _ = w.Write(encrypt(want, key, iv))
	})

	var output bytes.Buffer
	err := (hls.Downloader{Client: server.Client(), RetryDelay: time.Millisecond}).Download(context.Background(), server.URL+"/media.m3u8", &output, nil)
	if err != nil {
		t.Fatal(err)
	}
	if requests != 2 {
		t.Fatalf("segment requests = %d, want 2", requests)
	}
	if !bytes.Equal(output.Bytes(), want) {
		t.Fatalf("output = %q, want %q", output.Bytes(), want)
	}
}

func TestDownloadRefreshingReplacesSignedPlaylist(t *testing.T) {
	refreshes := 0
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	playlist := func(token string) string {
		return fmt.Sprintf("#EXTM3U\n#EXTINF:1,\n0.ts?token=%s\n#EXTINF:1,\n1.ts?token=%s\n#EXTINF:1,\n2.ts?token=%s\n#EXT-X-ENDLIST\n", token, token, token)
	}
	mux.HandleFunc("/initial.m3u8", func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, playlist("old")) })
	mux.HandleFunc("/fresh.m3u8", func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, playlist("new")) })
	for index := 0; index < 3; index++ {
		index := index
		mux.HandleFunc(fmt.Sprintf("/%d.ts", index), func(w http.ResponseWriter, r *http.Request) {
			if index >= 2 && r.URL.Query().Get("token") != "new" {
				http.Error(w, "expired signed URL", http.StatusForbidden)
				return
			}
			fmt.Fprintf(w, "part-%d", index)
		})
	}

	var output bytes.Buffer
	downloader := hls.Downloader{Client: server.Client(), RefreshInterval: 2, RetryDelay: time.Millisecond, RefreshDelay: time.Millisecond}
	err := downloader.DownloadRefreshing(context.Background(), server.URL+"/initial.m3u8", func(context.Context) (string, error) {
		refreshes++
		return server.URL + "/fresh.m3u8", nil
	}, &output, nil)
	if err != nil {
		t.Fatal(err)
	}
	if refreshes != 1 {
		t.Fatalf("refreshes = %d, want 1", refreshes)
	}
	if got := output.String(); got != "part-0part-1part-2" {
		t.Fatalf("output = %q", got)
	}
}

func TestDownloadFetchesSegmentsConcurrentlyAndWritesInOrder(t *testing.T) {
	const segmentCount = 12
	var active atomic.Int32
	var maximum atomic.Int32
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	mux.HandleFunc("/media.m3u8", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintln(w, "#EXTM3U")
		for index := 0; index < segmentCount; index++ {
			fmt.Fprintf(w, "#EXTINF:1,\n%d.ts\n", index)
		}
		fmt.Fprintln(w, "#EXT-X-ENDLIST")
	})
	for index := 0; index < segmentCount; index++ {
		index := index
		mux.HandleFunc(fmt.Sprintf("/%d.ts", index), func(w http.ResponseWriter, _ *http.Request) {
			current := active.Add(1)
			for {
				previous := maximum.Load()
				if current <= previous || maximum.CompareAndSwap(previous, current) {
					break
				}
			}
			defer active.Add(-1)
			time.Sleep(20 * time.Millisecond)
			fmt.Fprintf(w, "[%02d]", index)
		})
	}

	var output bytes.Buffer
	err := (hls.Downloader{Client: server.Client(), Concurrency: 4}).Download(context.Background(), server.URL+"/media.m3u8", &output, nil)
	if err != nil {
		t.Fatal(err)
	}
	if maximum.Load() < 2 {
		t.Fatalf("maximum concurrent requests = %d, want at least 2", maximum.Load())
	}
	var want strings.Builder
	for index := 0; index < segmentCount; index++ {
		fmt.Fprintf(&want, "[%02d]", index)
	}
	if output.String() != want.String() {
		t.Fatalf("output = %q, want %q", output.String(), want.String())
	}
}

func TestDownloadReportsProgressBeforeSlowBatchFinishes(t *testing.T) {
	releaseSlowSegment := make(chan struct{})
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	mux.HandleFunc("/media.m3u8", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "#EXTM3U\n#EXTINF:1,\n0.ts\n#EXTINF:1,\n1.ts\n#EXT-X-ENDLIST\n")
	})
	mux.HandleFunc("/0.ts", func(w http.ResponseWriter, _ *http.Request) {
		<-releaseSlowSegment
		fmt.Fprint(w, "slow")
	})
	mux.HandleFunc("/1.ts", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "fast")
	})

	updates := make(chan hls.Progress, 2)
	done := make(chan error, 1)
	go func() {
		var output bytes.Buffer
		done <- (hls.Downloader{Client: server.Client(), Concurrency: 2}).Download(
			context.Background(), server.URL+"/media.m3u8", &output,
			func(current hls.Progress) { updates <- current },
		)
	}()

	select {
	case current := <-updates:
		if current.Completed != 1 || current.DownloadedBytes != int64(len("fast")) {
			t.Fatalf("first progress = %#v", current)
		}
		close(releaseSlowSegment)
	case <-time.After(time.Second):
		close(releaseSlowSegment)
		t.Fatal("progress waited for the entire batch")
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestFetchRedactsSignedQueryFromErrors(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("connection failed")
	})}
	var output bytes.Buffer
	err := (hls.Downloader{Client: client, RetryDelay: time.Millisecond, RefreshDelay: time.Millisecond}).Download(
		context.Background(),
		"https://media.example/video.m3u8?sign=secret-token&expires=123",
		&output,
		nil,
	)
	if err == nil {
		t.Fatal("fetch unexpectedly succeeded")
	}
	if strings.Contains(err.Error(), "secret-token") || strings.Contains(err.Error(), "expires=") {
		t.Fatalf("error leaked signed query: %v", err)
	}
	if !strings.Contains(err.Error(), "https://media.example/video.m3u8") {
		t.Fatalf("error omitted safe URL context: %v", err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func TestRecordPollsAndDeduplicatesSegments(t *testing.T) {
	requests := 0
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	mux.HandleFunc("/live.m3u8", func(w http.ResponseWriter, _ *http.Request) {
		requests++
		fmt.Fprint(w, "#EXTM3U\n#EXT-X-MEDIA-SEQUENCE:0\n#EXTINF:1,\n0.ts\n")
		if requests > 1 {
			fmt.Fprint(w, "#EXTINF:1,\n1.ts\n")
		}
	})
	mux.HandleFunc("/0.ts", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("first")) })
	mux.HandleFunc("/1.ts", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("second")) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var output bytes.Buffer
	err := (hls.Downloader{Client: server.Client()}).Record(ctx, server.URL+"/live.m3u8", &output, time.Millisecond, func(total int) {
		if total == 2 {
			cancel()
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := output.String(); got != "firstsecond" {
		t.Fatalf("recorded output = %q", got)
	}
}

func encrypt(plain, key, iv []byte) []byte {
	padding := aes.BlockSize - len(plain)%aes.BlockSize
	padded := append(append([]byte(nil), plain...), bytes.Repeat([]byte{byte(padding)}, padding)...)
	block, _ := aes.NewCipher(key)
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(padded, padded)
	return padded
}
