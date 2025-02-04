package main

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/JasonKhew96/biliroaming-go-server/entity"
	"github.com/mailru/easyjson"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpproxy"
)

func (b *BiliroamingGo) initProxy(c *Config) {
	b.cnClient = b.newClient(c.Proxy.CN)
	b.hkClient = b.newClient(c.Proxy.HK)
	b.twClient = b.newClient(c.Proxy.TW)
	b.thClient = b.newClient(c.Proxy.TH)
	b.defaultClient = &fasthttp.Client{}
}

func (b *BiliroamingGo) newClient(proxy string) *fasthttp.Client {
	if proxy != "" {
		b.sugar.Debug("New socks proxy client: ", proxy)
		return &fasthttp.Client{
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 5 * time.Second,
			Dial:         fasthttpproxy.FasthttpSocksDialer(proxy),
		}
	}
	b.sugar.Debug("New normal client")
	return &fasthttp.Client{
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}
}

func (b *BiliroamingGo) getClientByArea(area string) *fasthttp.Client {
	switch strings.ToLower(area) {
	case "cn":
		return b.cnClient
	case "hk":
		return b.hkClient
	case "tw":
		return b.twClient
	case "th":
		return b.thClient
	default:
		return b.defaultClient
	}
}

func (b *BiliroamingGo) getReverseProxyByArea(area string) string {
	switch strings.ToLower(area) {
	case "cn":
		return b.config.Reverse.CN
	case "hk":
		return b.config.Reverse.HK
	case "tw":
		return b.config.Reverse.TW
	case "th":
		return b.config.Reverse.TH
	default:
		return ""
	}
}

func (b *BiliroamingGo) getReverseSearchProxyByArea(area string) string {
	switch strings.ToLower(area) {
	case "cn":
		return b.config.ReverseSearch.CN
	case "hk":
		return b.config.ReverseSearch.HK
	case "tw":
		return b.config.ReverseSearch.TW
	default:
		return ""
	}
}

func (b *BiliroamingGo) getReverseWebSearchProxyByArea(area string) string {
	switch strings.ToLower(area) {
	case "cn":
		return b.config.ReverseWebSearch.CN
	case "hk":
		return b.config.ReverseWebSearch.HK
	case "tw":
		return b.config.ReverseWebSearch.TW
	default:
		return ""
	}
}

func setDefaultHeaders(ctx *fasthttp.RequestCtx) {
	ctx.Response.Header.SetBytesKV([]byte("Access-Control-Allow-Origin"), []byte("https://www.bilibili.com"))
	ctx.Response.Header.SetBytesKV([]byte("Access-Control-Allow-Credentials"), []byte("true"))
	ctx.SetContentTypeBytes([]byte("application/json"))
}

func writeErrorJSON(ctx *fasthttp.RequestCtx, code int, msg []byte) {
	setDefaultHeaders(ctx)
	resp := &entity.SimpleResponse{
		Code:    code,
		Message: string(msg),
		TTL:     1,
	}
	respData, err := easyjson.Marshal(resp)
	if err != nil {
		ctx.Write([]byte(`{"code":-500,"message":"服务器错误","ttl":1}`))
		return
	}
	ctx.Write(respData)
	// ctx.Write([]byte(`{"accept_format":"mp4","code":0,"seek_param":"start","is_preview":0,"fnval":1,"video_project":true,"fnver":0,"type":"MP4","bp":0,"result":"suee","seek_type":"offset","qn_extras":[{"attribute":0,"icon":"http://i0.hdslb.com/bfs/app/81dab3a04370aafa93525053c4e760ac834fcc2f.png","icon2":"http://i0.hdslb.com/bfs/app/4e6f14c2806f7cc508d8b6f5f1d8306f94a71ecc.png","need_login":true,"need_vip":true,"qn":112},{"attribute":0,"icon":"","icon2":"","need_login":false,"need_vip":false,"qn":80},{"attribute":0,"icon":"","icon2":"","need_login":false,"need_vip":false,"qn":64},{"attribute":0,"icon":"","icon2":"","need_login":false,"need_vip":false,"qn":32},{"attribute":0,"icon":"","icon2":"","need_login":false,"need_vip":false,"qn":16}],"accept_watermark":[false,false,false,false,false],"from":"local","video_codecid":7,"durl":[{"order":1,"length":16740,"size":172775,"ahead":"","vhead":"","url":"https://s1.hdslb.com/bfs/static/player/media/error.mp4","backup_url":[]}],"no_rexcode":0,"format":"mp4","support_formats":[{"display_desc":"360P","superscript":"","format":"mp4","description":"流畅 360P","quality":16,"new_description":"360P 流畅"}],"message":"","accept_quality":[16],"quality":16,"timelength":16740,"has_paid":false,"accept_description":["流畅 360P"],"status":2}`))
}

func writeHealthJSON(ctx *fasthttp.RequestCtx, health *entity.Health) {
	setDefaultHeaders(ctx)
	if health == nil {
		ctx.Write([]byte(`{"code":-500,"message":"服务器错误","ttl":1}`))
		return
	}
	respData, err := easyjson.Marshal(health)
	if err != nil {
		ctx.Write([]byte(`{"code":-500,"message":"服务器错误","ttl":1}`))
		return
	}
	ctx.Write(respData)
}

func (b *BiliroamingGo) checkRoamingVer(ctx *fasthttp.RequestCtx) bool {
	versionCode := ctx.Request.Header.PeekBytes([]byte("build"))
	versionName := ctx.Request.Header.PeekBytes([]byte("x-from-biliroaming"))

	if len(versionCode) == 0 && len(versionName) == 0 {
		return true
	}

	if len(versionCode) > 0 && len(versionName) > 0 {
		build, err := strconv.Atoi(string(versionCode))
		if err != nil {
			writeErrorJSON(ctx, 500, []byte("错误请求头"))
			return false
		}
		if build < b.config.RoamingMinVer {
			writeErrorJSON(ctx, 500, []byte("哔哩漫游模块版本过低"))
			return false
		}
		return true
	}

	writeErrorJSON(ctx, 500, []byte("错误请求头"))
	return false
}

func (b *BiliroamingGo) doRequest(client *fasthttp.Client, ua []byte, url string, method []byte) ([]byte, error) {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	req.SetRequestURI(url)
	req.Header.SetBytesKV([]byte("Accept-Encoding"), []byte("br, gzip"))
	req.Header.SetUserAgentBytes(ua)
	req.Header.SetMethodBytes(method)

	b.sugar.Debugf("doRequest: %s", req.RequestURI())

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	err := client.DoRedirects(req, resp, 1)
	if err != nil {
		return nil, err
	}

	b.sugar.Debugf("doRedirects: %d", resp.StatusCode())

	if resp.StatusCode() != fasthttp.StatusOK {
		return nil, fmt.Errorf("error code: %d\nbody: %s", resp.StatusCode(), string(resp.Body()))
	}

	contentEncoding := resp.Header.Peek("Content-Encoding")
	var bodyBytes []byte
	if bytes.EqualFold(contentEncoding, []byte("gzip")) {
		bodyBytes, err = resp.BodyGunzip()
	} else if bytes.EqualFold(contentEncoding, []byte("br")) {
		bodyBytes, err = resp.BodyUnbrotli()
	} else {
		bodyBytes = resp.Body()
	}

	if err != nil {
		return nil, err
	}

	b.sugar.Debug("Content: ", string(bodyBytes))

	return bodyBytes, nil
}

func (b *BiliroamingGo) doRequestJsonWithRetry(client *fasthttp.Client, ua []byte, url string, method []byte, retry int) ([]byte, error) {
	var err error
	for i := 0; i < retry; i++ {
		bodyBytes, err := b.doRequestJson(client, ua, url, method)
		if err == nil {
			return bodyBytes, nil
		}
		b.sugar.Errorf("doRequestJsonWithRetry: %s", err)
	}
	return nil, err
}

func (b *BiliroamingGo) doRequestJson(client *fasthttp.Client, ua []byte, url string, method []byte) ([]byte, error) {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	req.SetRequestURI(url)
	req.Header.SetBytesKV([]byte("Accept-Encoding"), []byte("br, gzip"))
	req.Header.SetUserAgentBytes(ua)
	req.Header.SetMethodBytes(method)

	b.sugar.Debugf("doRequest: %s", req.RequestURI())

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	err := client.DoRedirects(req, resp, 3)
	if err != nil {
		return nil, err
	}

	b.sugar.Debugf("doRedirects: %d", resp.StatusCode())

	if resp.StatusCode() != fasthttp.StatusOK {
		return nil, fmt.Errorf("error code: %d\nbody: %s", resp.StatusCode(), string(resp.Body()))
	}

	// Verify the content type
	contentType := resp.Header.Peek("Content-Type")
	if bytes.Index(contentType, []byte("application/json")) != 0 {
		return nil, fmt.Errorf("expected content-type json but %s", string(contentType))
	}

	contentEncoding := resp.Header.Peek("Content-Encoding")
	var bodyBytes []byte
	if bytes.EqualFold(contentEncoding, []byte("gzip")) {
		bodyBytes, err = resp.BodyGunzip()
	} else if bytes.EqualFold(contentEncoding, []byte("br")) {
		bodyBytes, err = resp.BodyUnbrotli()
	} else {
		bodyBytes = resp.Body()
	}

	if err != nil {
		return nil, err
	}

	body := string(bodyBytes)

	b.sugar.Debug("Content: ", body)

	// Remove mid from json content
	if strings.Contains(url, "/playurl?") {
		body = removeMid(body)
		b.sugar.Debug("New content: ", body)
	}

	return []byte(body), nil
}
