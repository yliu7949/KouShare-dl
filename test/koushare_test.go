package test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"

	"github.com/yliu7949/KouShare-dl/internal/koushare"
)

func TestSignatureMatchesHAR(t *testing.T) {
	got := koushare.Signature(map[string]any{"id": "226895"}, http.MethodGet, 1790845396925)
	const want = "64118c2789a524182fb6088e7c7d365e"
	if got != want {
		t.Fatalf("signature = %q, want %q", got, want)
	}
}

func TestSignatureIgnoresEmptyValuesLikeWebClient(t *testing.T) {
	got := koushare.Signature(map[string]any{"videoId": "226895", "ticket": ""}, http.MethodGet, 1790845398557)
	const want = "1501b84def82ac0932efb9aaeea30c8d"
	if got != want {
		t.Fatalf("signature = %q, want %q", got, want)
	}
}

func TestSignatureMatchesHAREmptyPost(t *testing.T) {
	got := koushare.Signature(map[string]any{}, http.MethodPost, 1790845412582)
	const want = "4d58c6a28d3312cb42298bba1c2c12a2"
	if got != want {
		t.Fatalf("signature = %q, want %q", got, want)
	}
}

func TestSignaturePreservesPasswordProtocolCompatibility(t *testing.T) {
	got := koushare.Signature(map[string]any{
		"id":       "226845",
		"password": "live-room-password",
	}, http.MethodPost, 1790845412582)
	const want = "8693e4522bcd65d387c2cbd8da68249b"
	if got != want {
		t.Fatalf("signature = %q, want %q", got, want)
	}
}

func TestVideoInfoAPI(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, http.MethodGet, "/video/v1/video/infoV2", url.Values{"id": {"226895"}})
		writeJSON(t, w, `{"code":200000,"msg":"操作成功","data":{"id":226895,"title":"示例视频","videoLength":2329,"coursewareName":"slides.pdf","coursewareUrl":"https://cdn.example/slides.pdf","videoSysReporters":[{"realName":"张华","title":"教授","unit":"厦门大学"}]}}`)
	})
	got, err := client.VideoInfo(context.Background(), "226895")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != 226895 || got.Title != "示例视频" || got.CoursewareURL == "" || len(got.VideoSysReporters) != 1 {
		t.Fatalf("unexpected video info: %#v", got)
	}
}

func TestLoginCaptchaConfigAPI(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, http.MethodGet, "/iam/captcha/appIdEncrypt", url.Values{
			"projectType": {"KS_CORE"}, "terminalType": {"PC_H5_APP"},
		})
		writeJSON(t, w, `{"code":200,"message":null,"data":{"captchaAppId":"194149696","aidEncrypted":"encrypted"}}`)
	})
	got, err := client.LoginCaptchaConfig(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.CaptchaAppID != "194149696" || got.AidEncrypted != "encrypted" {
		t.Fatalf("unexpected captcha config: %#v", got)
	}
}

func TestSendLoginSMSAPI(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, http.MethodPost, "/iam/register/sendSmsSlideVerification", nil)
		var got map[string]any
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		want := map[string]any{
			"phone": "13800138000", "areaCode": "86", "scope": "LOGIN",
			"captchaAppId": "app-id", "ticket": "ticket", "randstr": "@rand",
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("body = %#v, want %#v", got, want)
		}
		writeJSON(t, w, `{"code":200000,"msg":"操作成功","data":true}`)
	})
	err := client.SendLoginSMS(context.Background(), "13800138000", koushare.CaptchaConfig{CaptchaAppID: "app-id"}, "ticket", "@rand")
	if err != nil {
		t.Fatal(err)
	}
}

func TestPhoneLoginAPI(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, http.MethodPost, "/iam/userLogin/phoneLogin", nil)
		var got map[string]any
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		want := map[string]any{"phone": "13800138000", "areaCode": "86", "code": "123456"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("body = %#v, want %#v", got, want)
		}
		writeJSON(t, w, `{"code":200000,"msg":"操作成功","data":{"code":200,"data":{"access_token":"access","refresh_token":"refresh","expire_in":2592000000,"refresh_expire_in":5184000000}}}`)
	})
	got, err := client.PhoneLogin(context.Background(), "13800138000", "123456")
	if err != nil {
		t.Fatal(err)
	}
	if got.AccessToken != "access" || got.RefreshToken != "refresh" || got.ExpireIn != 2592000000 {
		t.Fatalf("unexpected login tokens: %#v", got)
	}
}

func TestVideoSeriesAPI(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, http.MethodGet, "/video/v1/video/listVideoSeriesByVideoIdV3", url.Values{"id": {"226895"}, "sortOrder": {"asc"}})
		writeJSON(t, w, `{"code":200000,"msg":"操作成功","data":{"seriesId":8,"seriesName":"专题","liveId":0,"videoList":[{"id":1,"title":"第一讲"},{"id":2,"title":"第二讲"}]}}`)
	})
	got, err := client.VideoSeries(context.Background(), "226895")
	if err != nil {
		t.Fatal(err)
	}
	if got.SeriesID == nil || *got.SeriesID != 8 || len(got.VideoList) != 2 {
		t.Fatalf("unexpected series: %#v", got)
	}
}

func TestVideoPlayAddressAPI(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, http.MethodGet, "/video/v1/video/getVideoPlayAddressV2", url.Values{"videoId": {"226895"}, "ticket": {""}})
		writeJSON(t, w, `{"code":200000,"msg":"操作成功","data":[{"type":"HLS","list":[{"id":1,"label":"高清","width":1280,"height":720,"fileUrl":"https://media.example/master.m3u8","drmType":"SimpleAES"}]}]}`)
	})
	got, err := client.VideoPlayAddress(context.Background(), "226895", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || len(got[0].List) != 1 || got[0].List[0].Height != 720 {
		t.Fatalf("unexpected streams: %#v", got)
	}
}

func TestLiveInfoAPI(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, http.MethodGet, "/live/v2/live/56428", nil)
		writeJSON(t, w, `{"code":200000,"msg":"操作成功","data":{"id":56428,"roomNo":"709634","title":"示例直播","liveStatus":1,"liveSt":"2026-10-01 13:00:00","liveMajorOrganizers":[{"organName":"天津大学"}]}}`)
	})
	got, err := client.LiveInfo(context.Background(), "56428")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != 56428 || got.RoomNo != "709634" || len(got.LiveMajorOrganizers) != 1 {
		t.Fatalf("unexpected live info: %#v", got)
	}
}

func TestLiveStreamAPI(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, http.MethodPost, "/live/v2/live/stream/56428", nil)
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		var got map[string]any
		if err := json.Unmarshal(body, &got); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, map[string]any{"password": "secret"}) {
			t.Fatalf("body = %#v", got)
		}
		writeJSON(t, w, `{"code":200000,"msg":"操作成功","data":{"auth":{"liveId":56428,"result":"AUTHENTICATED","status":1},"streamUrls":[{"type":"HLS","list":[{"qualityValue":720,"url":"https://media.example/live.m3u8","isLogIn":false}]}]}}`)
	})
	got, err := client.LiveStream(context.Background(), "56428", "secret")
	if err != nil {
		t.Fatal(err)
	}
	if got.Authorization.Result != "AUTHENTICATED" || len(got.StreamURLs) != 1 {
		t.Fatalf("unexpected live stream response: %#v", got)
	}
}

func TestLiveFastbackListAPI(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, http.MethodGet, "/live/v2/live/fastback/list", url.Values{"liveId": {"56413"}})
		writeJSON(t, w, `{"code":200000,"msg":"操作成功","data":[{"id":25882,"name":"回放2","duration":4097,"views":9}]}`)
	})
	got, err := client.LiveFastbackList(context.Background(), "56413")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != 25882 || got[0].Name != "回放2" {
		t.Fatalf("unexpected fastback list: %#v", got)
	}
}

func TestLiveFastbackPlayAPI(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, http.MethodPost, "/live/v2/live/fastback/play", nil)
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		var got map[string]any
		if err := json.Unmarshal(body, &got); err != nil {
			t.Fatal(err)
		}
		want := map[string]any{"liveId": float64(56413), "fastBackId": float64(25882), "password": "secret"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("body = %#v, want %#v", got, want)
		}
		writeJSON(t, w, `{"code":200000,"msg":"操作成功","data":{"auth":{"liveId":56413,"result":"AUTHENTICATED","status":1},"fastbackUrl":"https://media.example/replay.m3u8","views":10}}`)
	})
	got, err := client.LiveFastbackPlay(context.Background(), 56413, 25882, "secret")
	if err != nil {
		t.Fatal(err)
	}
	if got.FastbackURL == "" || got.Views != 10 {
		t.Fatalf("unexpected fastback play response: %#v", got)
	}
}

func TestAPIError(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(t, w, `{"code":100010,"msg":"登录已过期","data":null}`)
	})
	_, err := client.VideoInfo(context.Background(), "1")
	if err == nil {
		t.Fatal("expected API error")
	}
}

func testClient(t *testing.T, handler http.HandlerFunc) *koushare.Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client := koushare.NewClient(server.Client(), "test-token")
	client.BaseURL = server.URL
	return client
}

func assertRequest(t *testing.T, r *http.Request, method, path string, query url.Values) {
	t.Helper()
	if r.Method != method || r.URL.Path != path {
		t.Fatalf("request = %s %s, want %s %s", r.Method, r.URL.Path, method, path)
	}
	if query == nil && len(r.URL.Query()) == 0 {
		// url.Values{} and nil are equivalent when no query is expected.
	} else if !reflect.DeepEqual(r.URL.Query(), query) {
		t.Fatalf("query = %#v, want %#v", r.URL.Query(), query)
	}
	if r.Header.Get("Client") != "front_web" || r.Header.Get("Ks-Sign") == "" || r.Header.Get("Ks-Timestamp") == "" {
		t.Fatalf("missing current API headers: %#v", r.Header)
	}
	if r.Header.Get("Authorization") != "test-token" {
		t.Fatalf("Authorization = %q", r.Header.Get("Authorization"))
	}
}

func writeJSON(t *testing.T, w http.ResponseWriter, body string) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if _, err := io.WriteString(w, body); err != nil {
		t.Fatal(err)
	}
}
