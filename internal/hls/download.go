package hls

import (
	"bufio"
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/tls"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	maxPlaylistDepth   = 3
	maxSegmentAttempts = 4
)

type Downloader struct {
	Client          *http.Client
	RetryDelay      time.Duration
	RefreshDelay    time.Duration
	RefreshInterval int
	BatchSize       int
	Concurrency     int
	segmentClient   *http.Client
}

type segment struct {
	URL    *url.URL
	KeyURL *url.URL
	IV     []byte
}

type playlist struct {
	Variant       *url.URL
	Segments      []segment
	MediaSequence uint64
}

// Download writes a VOD HLS stream as a concatenated MPEG-TS stream. It
// follows standard AES-128 key declarations from the playlist and only uses
// URLs that the server returned to the caller.
func (d Downloader) Download(ctx context.Context, playlistURL string, dst io.Writer, progress func(done, total int)) error {
	return d.download(ctx, playlistURL, nil, dst, progress)
}

// DownloadRefreshing periodically obtains a newly signed playlist URL. Some
// media CDNs cap the number of resource requests allowed by one signed URL,
// even though the VOD playlist contains more segments than that cap.
func (d Downloader) DownloadRefreshing(ctx context.Context, playlistURL string, refresh func(context.Context) (string, error), dst io.Writer, progress func(done, total int)) error {
	if refresh == nil {
		return errors.New("HLS 刷新函数不能为空")
	}
	return d.download(ctx, playlistURL, refresh, dst, progress)
}

func (d Downloader) download(ctx context.Context, playlistURL string, refresh func(context.Context) (string, error), dst io.Writer, progress func(done, total int)) error {
	d.renewClient()
	u, err := url.Parse(playlistURL)
	if err != nil {
		return fmt.Errorf("解析 HLS 地址: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("不支持的 HLS 协议 %q", u.Scheme)
	}
	pl, err := d.loadPlaylistWithRetry(ctx, u)
	if err != nil {
		return err
	}
	if len(pl.Segments) == 0 {
		return errors.New("HLS 清单中没有媒体片段")
	}

	refreshInterval := d.RefreshInterval
	if refreshInterval <= 0 {
		refreshInterval = 90
	}
	batchSize := d.BatchSize
	if batchSize <= 0 {
		concurrency := d.Concurrency
		if concurrency <= 0 {
			concurrency = 12
		}
		batchSize = concurrency * 2
	}
	batchSize = min(batchSize, refreshInterval)
	total := len(pl.Segments)
	nextRefresh := refreshInterval
	for start := 0; start < total; {
		if refresh != nil && start == nextRefresh {
			freshPlaylist, refreshErr := d.refreshPlaylistWithRetry(ctx, refresh, total)
			if refreshErr != nil {
				return fmt.Errorf("刷新 HLS 播放地址: %w", refreshErr)
			}
			pl = freshPlaylist
			nextRefresh += refreshInterval
		}
		end := min(start+batchSize, total)
		if refresh != nil {
			end = min(end, nextRefresh)
		}
		var batchErr error
		for attempt := 0; attempt < maxSegmentAttempts; attempt++ {
			batchErr = d.downloadBatch(ctx, pl.Segments[start:end], start, total, dst, progress)
			if batchErr == nil {
				break
			}
			if refresh == nil {
				break
			}
			freshPlaylist, refreshErr := d.refreshPlaylistWithRetry(ctx, refresh, total)
			if refreshErr != nil {
				batchErr = fmt.Errorf("%w；刷新播放地址失败: %v", batchErr, refreshErr)
				continue
			}
			pl = freshPlaylist
		}
		if batchErr != nil {
			return fmt.Errorf("处理 HLS 片段 %d-%d/%d: %w", start+1, end, total, batchErr)
		}
		start = end
	}
	return nil
}

func (d *Downloader) loadPlaylistWithRetry(ctx context.Context, playlistURL *url.URL) (playlist, error) {
	var lastErr error
	for attempt := 0; attempt < maxSegmentAttempts; attempt++ {
		if attempt > 0 {
			delay := d.RefreshDelay
			if delay <= 0 {
				delay = 1500 * time.Millisecond
			}
			timer := time.NewTimer(delay << (attempt - 1))
			select {
			case <-ctx.Done():
				if !timer.Stop() {
					<-timer.C
				}
				return playlist{}, ctx.Err()
			case <-timer.C:
			}
			d.renewClient()
		}
		pl, err := d.loadPlaylist(ctx, playlistURL, 0)
		if err == nil {
			return pl, nil
		}
		lastErr = err
	}
	return playlist{}, fmt.Errorf("重试 %d 次后仍无法获取 HLS 清单: %w", maxSegmentAttempts, lastErr)
}

func (d *Downloader) refreshPlaylistWithRetry(ctx context.Context, refresh func(context.Context) (string, error), expectedSegments int) (playlist, error) {
	delay := d.RefreshDelay
	if delay <= 0 {
		delay = 1500 * time.Millisecond
	}
	var lastErr error
	for attempt := 0; attempt < maxSegmentAttempts; attempt++ {
		timer := time.NewTimer(delay << attempt)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return playlist{}, ctx.Err()
		case <-timer.C:
		}
		d.renewClient()
		pl, err := d.refreshPlaylist(ctx, refresh, expectedSegments)
		if err == nil {
			return pl, nil
		}
		lastErr = err
	}
	return playlist{}, fmt.Errorf("重试 %d 次后仍失败: %w", maxSegmentAttempts, lastErr)
}

func (d Downloader) downloadBatch(ctx context.Context, items []segment, offset, total int, dst io.Writer, progress func(done, total int)) error {
	keys, err := d.loadKeys(ctx, items)
	if err != nil {
		return err
	}
	concurrency := d.Concurrency
	if concurrency <= 0 {
		concurrency = 12
	}
	concurrency = min(concurrency, len(items))
	results := make([][]byte, len(items))
	jobs := make(chan int, len(items))
	workerContext, cancel := context.WithCancel(ctx)
	defer cancel()
	var (
		workers   sync.WaitGroup
		errorOnce sync.Once
		batchErr  error
	)
	for worker := 0; worker < concurrency; worker++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for index := range jobs {
				data, fetchErr := d.readSegmentWithRetry(workerContext, items[index], keys)
				if fetchErr != nil {
					errorOnce.Do(func() {
						batchErr = fmt.Errorf("片段 %d/%d: %w", offset+index+1, total, fetchErr)
						cancel()
					})
					continue
				}
				results[index] = data
			}
		}()
	}
	for index := range items {
		jobs <- index
	}
	close(jobs)
	workers.Wait()
	if batchErr != nil {
		return batchErr
	}
	for index, data := range results {
		if _, err := dst.Write(data); err != nil {
			return fmt.Errorf("写入 HLS 片段: %w", err)
		}
		if progress != nil {
			progress(offset+index+1, total)
		}
	}
	return nil
}

func (d Downloader) loadKeys(ctx context.Context, items []segment) (map[string][]byte, error) {
	keys := make(map[string][]byte)
	for _, item := range items {
		if item.KeyURL == nil {
			continue
		}
		identity := item.KeyURL.String()
		if keys[identity] != nil {
			continue
		}
		key, err := d.fetch(ctx, item.KeyURL, 1<<20)
		if err != nil {
			return nil, fmt.Errorf("获取 HLS AES 密钥: %w", err)
		}
		if len(key) != aes.BlockSize {
			return nil, fmt.Errorf("HLS AES 密钥长度为 %d，期望 %d", len(key), aes.BlockSize)
		}
		keys[identity] = key
	}
	return keys, nil
}

func (d *Downloader) renewClient() {
	client := d.Client
	if client == nil {
		client = http.DefaultClient
	}
	client.CloseIdleConnections()
	clone := *client
	if transport, ok := client.Transport.(*http.Transport); ok {
		clone.Transport = transport.Clone()
		segmentTransport := transport.Clone()
		// Koushare's media edge starts returning a short HTML body after a
		// burst of segment requests over a reused HTTP/2/TLS session. Only
		// segment traffic uses short-lived HTTP/1 connections; playlists and
		// keys retain normal HTTP/2 because the CDN expects it during refresh.
		segmentTransport.DisableKeepAlives = true
		segmentTransport.ForceAttemptHTTP2 = false
		segmentTransport.TLSNextProto = make(map[string]func(string, *tls.Conn) http.RoundTripper)
		segmentTransport.MaxConnsPerHost = 0
		segmentClone := clone
		segmentClone.Transport = segmentTransport
		d.segmentClient = &segmentClone
	}
	d.Client = &clone
}

func (d Downloader) refreshPlaylist(ctx context.Context, refresh func(context.Context) (string, error), expectedSegments int) (playlist, error) {
	playlistURL, err := refresh(ctx)
	if err != nil {
		return playlist{}, err
	}
	u, err := url.Parse(playlistURL)
	if err != nil {
		return playlist{}, fmt.Errorf("解析刷新的 HLS 地址: %w", err)
	}
	pl, err := d.loadPlaylist(ctx, u, 0)
	if err != nil {
		return playlist{}, err
	}
	if len(pl.Segments) != expectedSegments {
		return playlist{}, fmt.Errorf("刷新后的片段数为 %d，期望 %d", len(pl.Segments), expectedSegments)
	}
	return pl, nil
}

// Record polls a live HLS playlist and appends each media segment once. It
// returns cleanly when ctx is cancelled by the caller after the live API says
// the broadcast has ended.
func (d Downloader) Record(ctx context.Context, playlistURL string, dst io.Writer, pollInterval time.Duration, progress func(total int)) error {
	u, err := url.Parse(playlistURL)
	if err != nil {
		return fmt.Errorf("解析直播 HLS 地址: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("不支持的 HLS 协议 %q", u.Scheme)
	}
	if pollInterval <= 0 {
		pollInterval = 2 * time.Second
	}
	seen := make(map[string]bool)
	keys := make(map[string][]byte)
	total := 0
	for {
		pl, loadErr := d.loadPlaylist(ctx, u, 0)
		if loadErr != nil {
			if ctx.Err() != nil {
				return nil
			}
			return loadErr
		}
		for _, item := range pl.Segments {
			identity := item.URL.String()
			if seen[identity] {
				continue
			}
			if err := d.writeSegment(ctx, item, dst, keys); err != nil {
				if ctx.Err() != nil {
					return nil
				}
				return err
			}
			seen[identity] = true
			total++
			if progress != nil {
				progress(total)
			}
		}
		timer := time.NewTimer(pollInterval)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return nil
		case <-timer.C:
		}
	}
}

func (d Downloader) writeSegment(ctx context.Context, item segment, dst io.Writer, keys map[string][]byte) error {
	data, err := d.readSegmentWithRetry(ctx, item, keys)
	if err != nil {
		return err
	}
	if _, err := dst.Write(data); err != nil {
		return fmt.Errorf("写入 HLS 片段: %w", err)
	}
	return nil
}

func (d Downloader) readSegmentWithRetry(ctx context.Context, item segment, keys map[string][]byte) ([]byte, error) {
	var err error
	for attempt := 0; attempt < maxSegmentAttempts; attempt++ {
		data, readErr := d.readSegment(ctx, item, keys)
		err = readErr
		if err == nil {
			return data, nil
		}
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if attempt == maxSegmentAttempts-1 {
			return nil, fmt.Errorf("重试 %d 次后仍失败: %w", maxSegmentAttempts, err)
		}
		// Some media CDNs start returning a small HTML response after many
		// requests on one keep-alive connection. Drop idle connections before
		// retrying so the same signed segment URL can be fetched afresh.
		d.closeIdleConnections()
		delay := d.RetryDelay
		if delay <= 0 {
			delay = 500 * time.Millisecond
		}
		timer := time.NewTimer(delay << attempt)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
	return nil, err
}

func (d Downloader) closeIdleConnections() {
	client := d.Client
	if client == nil {
		client = http.DefaultClient
	}
	client.CloseIdleConnections()
	if d.segmentClient != nil {
		d.segmentClient.CloseIdleConnections()
	}
}

func (d Downloader) readSegment(ctx context.Context, item segment, keys map[string][]byte) ([]byte, error) {
	data, err := d.fetchWithClient(ctx, item.URL, 512<<20, d.segmentClient)
	if err != nil {
		return nil, err
	}
	if item.KeyURL != nil {
		key := keys[item.KeyURL.String()]
		if key == nil {
			key, err = d.fetch(ctx, item.KeyURL, 1<<20)
			if err != nil {
				return nil, fmt.Errorf("获取 HLS AES 密钥: %w", err)
			}
			if len(key) != aes.BlockSize {
				return nil, fmt.Errorf("HLS AES 密钥长度为 %d，期望 %d", len(key), aes.BlockSize)
			}
			keys[item.KeyURL.String()] = key
		}
		data, err = decryptAES128(data, key, item.IV)
		if err != nil {
			return nil, err
		}
	}
	return data, nil
}

func (d Downloader) loadPlaylist(ctx context.Context, playlistURL *url.URL, depth int) (playlist, error) {
	if depth > maxPlaylistDepth {
		return playlist{}, errors.New("HLS 主清单嵌套过深")
	}
	data, err := d.fetch(ctx, playlistURL, 16<<20)
	if err != nil {
		return playlist{}, fmt.Errorf("获取 HLS 清单: %w", err)
	}
	pl, err := parsePlaylist(data, playlistURL)
	if err != nil {
		return playlist{}, err
	}
	if pl.Variant != nil {
		return d.loadPlaylist(ctx, pl.Variant, depth+1)
	}
	return pl, nil
}

func (d Downloader) fetch(ctx context.Context, target *url.URL, limit int64) ([]byte, error) {
	return d.fetchWithClient(ctx, target, limit, d.Client)
}

func (d Downloader) fetchWithClient(ctx context.Context, target *url.URL, limit int64, client *http.Client) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Origin", "https://www.koushare.com")
	req.Header.Set("Referer", "https://www.koushare.com/")
	req.Header.Set("User-Agent", "KouShare-dl/1")
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, redactedRequestError(target, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	limited := io.LimitReader(resp.Body, limit+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("响应超过 %d 字节上限", limit)
	}
	return data, nil
}

func redactedRequestError(target *url.URL, requestErr error) error {
	underlying := requestErr
	var urlErr *url.Error
	if errors.As(requestErr, &urlErr) {
		underlying = urlErr.Err
	}
	safeURL := *target
	safeURL.RawQuery = ""
	safeURL.ForceQuery = false
	safeURL.Fragment = ""
	return fmt.Errorf("请求 %s: %w", safeURL.String(), underlying)
}

func parsePlaylist(data []byte, base *url.URL) (playlist, error) {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 64<<10), 2<<20)
	pl := playlist{}
	var (
		variantNext bool
		keyURL      *url.URL
		iv          []byte
		sequence    uint64
	)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		switch {
		case strings.HasPrefix(line, "#EXT-X-MEDIA-SEQUENCE:"):
			value := strings.TrimPrefix(line, "#EXT-X-MEDIA-SEQUENCE:")
			parsed, err := strconv.ParseUint(strings.TrimSpace(value), 10, 64)
			if err != nil {
				return playlist{}, fmt.Errorf("无效的 HLS 媒体序号 %q", value)
			}
			pl.MediaSequence = parsed
			sequence = parsed
		case strings.HasPrefix(line, "#EXT-X-STREAM-INF:"):
			variantNext = true
		case strings.HasPrefix(line, "#EXT-X-KEY:"):
			attrs := parseAttributes(strings.TrimPrefix(line, "#EXT-X-KEY:"))
			method := strings.ToUpper(attrs["METHOD"])
			if method == "NONE" {
				keyURL = nil
				iv = nil
				continue
			}
			if method != "AES-128" {
				return playlist{}, fmt.Errorf("不支持的 HLS 加密方法 %q", method)
			}
			uri := attrs["URI"]
			if uri == "" {
				return playlist{}, errors.New("HLS AES-128 密钥缺少 URI")
			}
			resolvedKeyURL, err := resolveURL(base, uri)
			if err != nil {
				return playlist{}, err
			}
			keyURL = resolvedKeyURL
			iv = nil
			if rawIV := strings.TrimPrefix(strings.TrimPrefix(attrs["IV"], "0x"), "0X"); rawIV != "" {
				decoded, err := hex.DecodeString(rawIV)
				if err != nil || len(decoded) != aes.BlockSize {
					return playlist{}, fmt.Errorf("无效的 HLS AES IV %q", attrs["IV"])
				}
				iv = decoded
			}
		case strings.HasPrefix(line, "#"):
			continue
		default:
			resolved, err := resolveURL(base, line)
			if err != nil {
				return playlist{}, err
			}
			if variantNext {
				pl.Variant = resolved
				return pl, nil
			}
			segmentIV := append([]byte(nil), iv...)
			if keyURL != nil && len(segmentIV) == 0 {
				segmentIV = make([]byte, aes.BlockSize)
				binary.BigEndian.PutUint64(segmentIV[8:], sequence)
			}
			pl.Segments = append(pl.Segments, segment{URL: resolved, KeyURL: keyURL, IV: segmentIV})
			sequence++
		}
	}
	if err := scanner.Err(); err != nil {
		return playlist{}, err
	}
	return pl, nil
}

func resolveURL(base *url.URL, ref string) (*url.URL, error) {
	u, err := url.Parse(strings.TrimSpace(ref))
	if err != nil {
		return nil, fmt.Errorf("解析 HLS 子资源地址: %w", err)
	}
	resolved := base.ResolveReference(u)
	if resolved.Scheme != "http" && resolved.Scheme != "https" {
		return nil, fmt.Errorf("不支持的 HLS 子资源协议 %q", resolved.Scheme)
	}
	return resolved, nil
}

func parseAttributes(input string) map[string]string {
	result := make(map[string]string)
	start := 0
	inQuotes := false
	consume := func(item string) {
		key, value, ok := strings.Cut(item, "=")
		if !ok {
			return
		}
		result[strings.TrimSpace(key)] = strings.Trim(strings.TrimSpace(value), `"`)
	}
	for index, char := range input {
		switch char {
		case '"':
			inQuotes = !inQuotes
		case ',':
			if !inQuotes {
				consume(input[start:index])
				start = index + 1
			}
		}
	}
	consume(input[start:])
	return result
}

func decryptAES128(data, key, iv []byte) ([]byte, error) {
	if len(iv) != aes.BlockSize {
		return nil, fmt.Errorf("AES IV 长度为 %d，期望 %d", len(iv), aes.BlockSize)
	}
	if len(data) == 0 || len(data)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("AES 密文长度 %d 不是块大小的整数倍", len(data))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	plain := make([]byte, len(data))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(plain, data)
	if padding := int(plain[len(plain)-1]); padding > 0 && padding <= aes.BlockSize && padding <= len(plain) {
		valid := true
		for _, value := range plain[len(plain)-padding:] {
			if int(value) != padding {
				valid = false
				break
			}
		}
		if valid {
			plain = plain[:len(plain)-padding]
		}
	}
	return plain, nil
}
