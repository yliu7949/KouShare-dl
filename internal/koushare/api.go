package koushare

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultBaseURL = "https://api-core.koushare.com"
	signSalt       = "arfw2r4k4rdwrlmchvcu7q61fs"
)

var errEmptyID = errors.New("KouShare ID 不能为空")

// Client implements the current api-core.koushare.com API used by the web
// client. The request signature is not an authentication credential; private
// resources still require a valid access token in Authorization.
type Client struct {
	HTTPClient  *http.Client
	BaseURL     string
	AccessToken string
	now         func() time.Time
}

func NewClient(httpClient *http.Client, accessToken string) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{
		HTTPClient:  httpClient,
		BaseURL:     DefaultBaseURL,
		AccessToken: accessToken,
		now:         time.Now,
	}
}

type envelope[T any] struct {
	Code            int    `json:"code"`
	Message         string `json:"msg"`
	MessageFallback string `json:"message"`
	Data            T      `json:"data"`
}

type Reporter struct {
	RealName string `json:"realName"`
	Title    string `json:"title"`
	Unit     string `json:"unit"`
}

type VideoInfo struct {
	ID                 int64      `json:"id"`
	Title              string     `json:"title"`
	Blurb              string     `json:"blurb"`
	UserName           string     `json:"userName"`
	ReportingLocation  string     `json:"reportingLocation"`
	ReportingTime      string     `json:"reportingTime"`
	ReleaseTime        string     `json:"releaseTime"`
	VideoLength        int64      `json:"videoLength"`
	TotalPlayLength    int64      `json:"totalPlayLength"`
	Authority          int        `json:"authority"`
	CanPlay            bool       `json:"canPlay"`
	UnplayReason       string     `json:"unPlayReason"`
	IsSkipLogin        int        `json:"isSkipLogin"`
	Price              float64    `json:"price"`
	CoursewareName     string     `json:"coursewareName"`
	CoursewareURL      string     `json:"coursewareUrl"`
	HighDefinitionURL  string     `json:"highDefinitionUrl"`
	StandardDefinition string     `json:"standardDefinition"`
	ValidVideoURL      string     `json:"validVideoUrl"`
	VideoURL           string     `json:"videoUrl"`
	SpecialID          int64      `json:"specialId"`
	VideoSysReporters  []Reporter `json:"videoSysReporters"`
}

type VideoSeries struct {
	SeriesID   *int64      `json:"seriesId"`
	SeriesName string      `json:"seriesName"`
	LiveID     int64       `json:"liveId"`
	VideoList  []VideoInfo `json:"videoList"`
}

type VideoStream struct {
	ID         int64  `json:"id"`
	Label      string `json:"label"`
	LabelEN    string `json:"labelEn"`
	Width      int    `json:"width"`
	Height     int    `json:"height"`
	FileURL    string `json:"fileUrl"`
	DRMType    string `json:"drmType"`
	AppLoginPX int    `json:"appLoginPx"`
}

type VideoStreamGroup struct {
	Type string        `json:"type"`
	List []VideoStream `json:"list"`
}

type CaptchaConfig struct {
	CaptchaAppID string `json:"captchaAppId"`
	AidEncrypted string `json:"aidEncrypted"`
}

type LoginTokens struct {
	AccessToken     string `json:"access_token"`
	RefreshToken    string `json:"refresh_token"`
	ExpireIn        int64  `json:"expire_in"`
	RefreshExpireIn int64  `json:"refresh_expire_in"`
}

type phoneLoginData struct {
	Code int         `json:"code"`
	Data LoginTokens `json:"data"`
}

type LiveOrganizer struct {
	Name string `json:"organName"`
}

type LiveInfo struct {
	ID                      int64           `json:"id"`
	RoomNo                  string          `json:"roomNo"`
	Title                   string          `json:"title"`
	LiveStart               string          `json:"liveSt"`
	LiveEnd                 string          `json:"liveEt"`
	LiveStatus              int             `json:"liveStatus"`
	Auth                    int             `json:"auth"`
	Notice                  string          `json:"notice"`
	Clicks                  int64           `json:"clicks"`
	Views                   int64           `json:"views"`
	SubjectName             string          `json:"subjectName"`
	HasFastPlayback         int             `json:"hasFastPlayback"`
	IsPublicPlayback        int             `json:"isPublicPlayback"`
	IsSkipLoginPlayback     int             `json:"isSkipLoginPlayback"`
	PlaybackURL             string          `json:"playbackUrl"`
	VideoID                 *int64          `json:"videoId"`
	LiveMajorOrganizers     []LiveOrganizer `json:"liveMajorOrganizers"`
	LiveAssistOrganizers    []LiveOrganizer `json:"liveAssistOrganizers"`
	LiveUndertakeOrganizers []LiveOrganizer `json:"liveUndertakeOrganizers"`
}

type LiveStream struct {
	ID           int64  `json:"id"`
	QualityValue int    `json:"qualityValue"`
	URL          string `json:"url"`
	IsLogin      bool   `json:"isLogIn"`
}

type LiveStreamGroup struct {
	Type string       `json:"type"`
	List []LiveStream `json:"list"`
}

type LiveAuthorization struct {
	LiveID   int64  `json:"liveId"`
	Result   string `json:"result"`
	Status   int    `json:"status"`
	Type     int    `json:"type"`
	IsPaging bool   `json:"isPaging"`
}

type LiveStreamResult struct {
	Authorization LiveAuthorization `json:"auth"`
	StreamURLs    []LiveStreamGroup `json:"streamUrls"`
}

type FastbackItem struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Duration int64  `json:"duration"`
	Views    int64  `json:"views"`
}

type FastbackPlay struct {
	Authorization LiveAuthorization `json:"auth"`
	FastbackURL   string            `json:"fastbackUrl"`
	Views         int64             `json:"views"`
}

func (c *Client) VideoInfo(ctx context.Context, id string) (VideoInfo, error) {
	var data VideoInfo
	if strings.TrimSpace(id) == "" {
		return data, errEmptyID
	}
	err := c.get(ctx, "/video/v1/video/infoV2", url.Values{"id": {id}}, &data)
	return data, err
}

func (c *Client) LoginCaptchaConfig(ctx context.Context) (CaptchaConfig, error) {
	var data CaptchaConfig
	err := c.doExpected(ctx, http.MethodGet, "/iam/captcha/appIdEncrypt", url.Values{
		"projectType": {"KS_CORE"}, "terminalType": {"PC_H5_APP"},
	}, nil, &data, 200)
	return data, err
}

func (c *Client) SendLoginSMS(ctx context.Context, phone string, captcha CaptchaConfig, ticket, randstr string) error {
	phone = strings.TrimSpace(phone)
	if phone == "" || captcha.CaptchaAppID == "" || strings.TrimSpace(ticket) == "" || strings.TrimSpace(randstr) == "" {
		return errors.New("发送短信所需的手机号或人机验证结果不完整")
	}
	body := map[string]any{
		"phone": phone, "areaCode": "86", "scope": "LOGIN",
		"captchaAppId": captcha.CaptchaAppID, "ticket": ticket, "randstr": randstr,
	}
	return c.post(ctx, "/iam/register/sendSmsSlideVerification", body, nil)
}

func (c *Client) PhoneLogin(ctx context.Context, phone, code string) (LoginTokens, error) {
	var result phoneLoginData
	phone = strings.TrimSpace(phone)
	code = strings.TrimSpace(code)
	if phone == "" || code == "" {
		return result.Data, errors.New("手机号和短信验证码不能为空")
	}
	err := c.post(ctx, "/iam/userLogin/phoneLogin", map[string]any{
		"code": code, "areaCode": "86", "phone": phone,
	}, &result)
	if err != nil {
		return result.Data, err
	}
	if result.Code != 200 || result.Data.AccessToken == "" {
		return result.Data, errors.New("短信登录响应中没有有效的访问令牌")
	}
	return result.Data, nil
}

func (c *Client) VideoSeries(ctx context.Context, id string) (VideoSeries, error) {
	var data VideoSeries
	if strings.TrimSpace(id) == "" {
		return data, errEmptyID
	}
	err := c.get(ctx, "/video/v1/video/listVideoSeriesByVideoIdV3", url.Values{
		"id": {id}, "sortOrder": {"asc"},
	}, &data)
	return data, err
}

func (c *Client) VideoPlayAddress(ctx context.Context, id, ticket string) ([]VideoStreamGroup, error) {
	var data []VideoStreamGroup
	if strings.TrimSpace(id) == "" {
		return data, errEmptyID
	}
	err := c.get(ctx, "/video/v1/video/getVideoPlayAddressV2", url.Values{
		"videoId": {id}, "ticket": {ticket},
	}, &data)
	return data, err
}

func (c *Client) LiveInfo(ctx context.Context, id string) (LiveInfo, error) {
	var data LiveInfo
	if strings.TrimSpace(id) == "" {
		return data, errEmptyID
	}
	err := c.get(ctx, "/live/v2/live/"+url.PathEscape(id), nil, &data)
	return data, err
}

func (c *Client) LiveStream(ctx context.Context, id, password string) (LiveStreamResult, error) {
	var data LiveStreamResult
	if strings.TrimSpace(id) == "" {
		return data, errEmptyID
	}
	body := map[string]any{}
	if password != "" {
		body["password"] = password
	}
	err := c.post(ctx, "/live/v2/live/stream/"+url.PathEscape(id), body, &data)
	return data, err
}

func (c *Client) LiveFastbackList(ctx context.Context, id string) ([]FastbackItem, error) {
	var data []FastbackItem
	if strings.TrimSpace(id) == "" {
		return data, errEmptyID
	}
	err := c.get(ctx, "/live/v2/live/fastback/list", url.Values{"liveId": {id}}, &data)
	return data, err
}

func (c *Client) LiveFastbackPlay(ctx context.Context, liveID, fastbackID int64, password string) (FastbackPlay, error) {
	var data FastbackPlay
	if liveID <= 0 || fastbackID <= 0 {
		return data, errEmptyID
	}
	body := map[string]any{"liveId": liveID, "fastBackId": fastbackID}
	if password != "" {
		body["password"] = password
	}
	err := c.post(ctx, "/live/v2/live/fastback/play", body, &data)
	return data, err
}

func (c *Client) get(ctx context.Context, path string, query url.Values, dst any) error {
	return c.do(ctx, http.MethodGet, path, query, nil, dst)
}

func (c *Client) post(ctx context.Context, path string, body map[string]any, dst any) error {
	return c.do(ctx, http.MethodPost, path, nil, body, dst)
}

func (c *Client) do(ctx context.Context, method, path string, query url.Values, body map[string]any, dst any) error {
	return c.doExpected(ctx, method, path, query, body, dst, 200000)
}

func (c *Client) doExpected(ctx context.Context, method, path string, query url.Values, body map[string]any, dst any, successCode int) error {
	base := strings.TrimRight(c.BaseURL, "/")
	if base == "" {
		base = DefaultBaseURL
	}
	u, err := url.Parse(base + path)
	if err != nil {
		return err
	}
	if query != nil {
		u.RawQuery = query.Encode()
	}

	params := make(map[string]any, len(query)+len(body))
	for key, values := range query {
		if len(values) > 0 {
			params[key] = values[0]
		}
	}
	for key, value := range body {
		params[key] = value
	}

	var requestBody io.Reader
	if body != nil {
		encoded, marshalErr := json.Marshal(body)
		if marshalErr != nil {
			return marshalErr
		}
		requestBody = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, u.String(), requestBody)
	if err != nil {
		return err
	}
	timestamp := c.now().UnixMilli()
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Client", "front_web")
	req.Header.Set("Ks-Sign", Signature(params, method, timestamp))
	req.Header.Set("Ks-Timestamp", strconv.FormatInt(timestamp, 10))
	req.Header.Set("Origin", "https://www.koushare.com")
	req.Header.Set("Referer", "https://www.koushare.com/")
	req.Header.Set("User-Agent", "KouShare-dl/1")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.AccessToken != "" {
		req.Header.Set("Authorization", c.AccessToken)
	}

	client := c.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))
		return fmt.Errorf("KouShare API 返回 HTTP %d", resp.StatusCode)
	}

	var raw json.RawMessage
	result := envelope[json.RawMessage]{Data: raw}
	decoder := json.NewDecoder(io.LimitReader(resp.Body, 16<<20))
	if err := decoder.Decode(&result); err != nil {
		return fmt.Errorf("解析 KouShare API 响应: %w", err)
	}
	if result.Code != successCode {
		if result.Message == "" {
			result.Message = result.MessageFallback
		}
		if result.Message == "" {
			result.Message = "未知错误"
		}
		return fmt.Errorf("KouShare API 错误 %d: %s", result.Code, result.Message)
	}
	if dst == nil || len(result.Data) == 0 || bytes.Equal(result.Data, []byte("null")) {
		return nil
	}
	if err := json.Unmarshal(result.Data, dst); err != nil {
		return fmt.Errorf("解析 KouShare API data: %w", err)
	}
	return nil
}

// Signature produces the checksum required by KouShare's private wire protocol.
// It is exported only within this repository because the package is internal.
func Signature(params map[string]any, method string, timestamp int64) string {
	keys := make([]string, 0, len(params))
	for key, value := range params {
		if value == nil || value == "" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys)+3)
	for _, key := range keys {
		parts = append(parts, key+"="+signatureValue(params[key]))
	}
	saltHash := md5.Sum([]byte(signSalt))
	parts = append(parts,
		"method="+strings.ToUpper(method),
		"timestamp="+strconv.FormatInt(timestamp, 10),
		"saltmd5="+hex.EncodeToString(saltHash[:]),
	)
	// KouShare's legacy wire protocol mandates MD5 here. This digest is only a
	// request compatibility checksum; HTTPS and the access token provide the
	// security boundary, and the digest is never used to store a password.
	// codeql[go/weak-sensitive-data-hashing]
	digest := md5.Sum([]byte(strings.Join(parts, "&")))
	return hex.EncodeToString(digest[:])
}

func signatureValue(value any) string {
	switch value := value.(type) {
	case string:
		return value
	case nil:
		return ""
	case []any, map[string]any:
		encoded, _ := json.Marshal(value)
		return string(encoded)
	default:
		return fmt.Sprint(value)
	}
}
