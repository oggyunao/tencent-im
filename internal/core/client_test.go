/**
 * @Author: fuxiao
 * @Email: 576101059@qq.com
 * @Date: 2025/9/2
 * @Desc: core 客户端单元测试（httptest 模拟，不依赖真实服务）
 */

package core

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"

	"github.com/oggyunao/tencent-im/internal/types"
)

// newTestClient 构造指向测试服务的客户端。
func newTestClient(baseURL string) Client {
	return NewClient(&Options{
		BaseUrl:   baseURL,
		AppId:     1400000000,
		AppSecret: "test-secret",
		UserId:    "administrator",
	})
}

// hijackAndClose 劫持连接后立即关闭，模拟服务端在返回响应前断开连接。
// 客户端表现为 EOF / unexpected EOF / connection reset by peer（均在可重试六类内）。
func hijackAndClose(t *testing.T, w http.ResponseWriter, r *http.Request) {
	t.Helper()

	// 先排空请求体，确保客户端已进入读取响应阶段，避免写入阶段产生 broken pipe（不可重试）干扰判定
	_, _ = io.Copy(io.Discard, r.Body)

	hj, ok := w.(http.Hijacker)
	if !ok {
		t.Error("server does not support hijacking")
		return
	}

	conn, _, err := hj.Hijack()
	if err != nil {
		t.Error(err)
		return
	}
	_ = conn.Close()
}

// TestRequestRetryOnNetworkError 网络层错误重试：前 2 次断连，第 3 次成功。
func TestRequestRetryOnNetworkError(t *testing.T) {
	t.Parallel()

	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&attempts, 1) <= 2 {
			hijackAndClose(t, w, r)
			return
		}
		_, _ = w.Write([]byte(`{"ActionStatus":"OK","ErrorCode":0,"ErrorInfo":""}`))
	}))
	defer srv.Close()

	err := newTestClient(srv.URL).Post("openim", "send_msg", map[string]string{"text": "hi"}, &types.ActionBaseResp{})
	if err != nil {
		t.Fatalf("expected success after retry, got %v", err)
	}

	if got := atomic.LoadInt32(&attempts); got != 3 {
		t.Fatalf("attempts = %d, want 3 (initial + 2 retries)", got)
	}
}

// TestRequestRetryExhausted 持续网络故障：耗尽重试次数后返回最后一次错误。
func TestRequestRetryExhausted(t *testing.T) {
	t.Parallel()

	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		hijackAndClose(t, w, r)
	}))
	defer srv.Close()

	err := newTestClient(srv.URL).Post("openim", "send_msg", map[string]string{"text": "hi"}, &types.ActionBaseResp{})
	if err == nil {
		t.Fatal("expected error after exhausting retries, got nil")
	}

	if got := atomic.LoadInt32(&attempts); got != int32(1+defaultRetryCount) {
		t.Fatalf("attempts = %d, want %d", got, 1+defaultRetryCount)
	}
}

// TestBusinessErrorNoRetry 业务错误不重试，且错误为 im.Error（Code/Message 可读）。
func TestBusinessErrorNoRetry(t *testing.T) {
	t.Parallel()

	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		_, _ = w.Write([]byte(`{"ActionStatus":"FAIL","ErrorCode":1001,"ErrorInfo":"invalid sig"}`))
	}))
	defer srv.Close()

	err := newTestClient(srv.URL).Post("openim", "send_msg", map[string]string{}, &types.ActionBaseResp{})
	if err == nil {
		t.Fatal("expected business error, got nil")
	}

	ce, ok := err.(Error)
	if !ok {
		t.Fatalf("expected im.Error, got %T: %v", err, err)
	}
	if ce.Code() != 1001 || ce.Message() != "invalid sig" {
		t.Fatalf("code = %d, message = %q, want 1001 / %q", ce.Code(), ce.Message(), "invalid sig")
	}

	if got := atomic.LoadInt32(&attempts); got != 1 {
		t.Fatalf("attempts = %d, want 1 (business error must not retry)", got)
	}
}

// TestRequestSuccessWithUrlParams 请求成功（BaseResp 分支），并校验 URL 查询参数与 baseUrl 尾部斜杠处理。
func TestRequestSuccessWithUrlParams(t *testing.T) {
	t.Parallel()

	var query url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query = r.URL.Query()
		_, _ = w.Write([]byte(`{"ErrorCode":0,"ErrorInfo":""}`))
	}))
	defer srv.Close()

	// BaseUrl 带尾部斜杠，应被规范化
	err := newTestClient(srv.URL + "/").Post("im_open_login_svc", "account_import", map[string]string{"UserID": "u1"}, &types.BaseResp{})
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}

	if got := query.Get("sdkappid"); got != "1400000000" {
		t.Errorf("sdkappid = %q, want %q", got, "1400000000")
	}
	if got := query.Get("identifier"); got != "administrator" {
		t.Errorf("identifier = %q, want %q", got, "administrator")
	}
	if got := query.Get("usersig"); got == "" {
		t.Error("usersig should not be empty")
	}
	if got := query.Get("random"); got == "" {
		t.Error("random should not be empty")
	}
	if got := query.Get("contenttype"); got != "json" {
		t.Errorf("contenttype = %q, want %q", got, "json")
	}
}

// TestRequestInvalidResponse 响应结构未实现 BaseRespInterface 时返回 invalidResponse。
func TestRequestInvalidResponse(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	err := newTestClient(srv.URL).Post("openim", "send_msg", map[string]string{}, &struct{}{})
	if !errors.Is(err, invalidResponse) {
		t.Fatalf("expected invalidResponse, got %v", err)
	}
}

// TestRequestWithNilData data 为 nil 时不携带请求体。
func TestRequestWithNilData(t *testing.T) {
	t.Parallel()

	var bodyLen int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		bodyLen = len(body)
		_, _ = w.Write([]byte(`{"ErrorCode":0}`))
	}))
	defer srv.Close()

	err := newTestClient(srv.URL).Get("openim", "query_state", nil, &types.BaseResp{})
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}

	if bodyLen != 0 {
		t.Fatalf("request body length = %d, want 0", bodyLen)
	}
}

// TestRequestBadResponseJson 响应非法 JSON：返回解析错误且不重试。
func TestRequestBadResponseJson(t *testing.T) {
	t.Parallel()

	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		_, _ = w.Write([]byte(`<html>bad gateway</html>`))
	}))
	defer srv.Close()

	err := newTestClient(srv.URL).Post("openim", "send_msg", map[string]string{}, &types.ActionBaseResp{})
	if err == nil {
		t.Fatal("expected unmarshal error, got nil")
	}
	if !strings.Contains(err.Error(), "unmarshal response body") {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := atomic.LoadInt32(&attempts); got != 1 {
		t.Fatalf("attempts = %d, want 1 (unmarshal error must not retry)", got)
	}
}

// tlsHandshakeTimeoutError 模拟标准库 transport 的 TLS 握手超时错误
// （net/http 内部类型未导出，行为一致：实现 net.Error 且 Timeout() 为 true）。
type tlsHandshakeTimeoutError struct{}

func (tlsHandshakeTimeoutError) Error() string   { return "net/http: TLS handshake timeout" }
func (tlsHandshakeTimeoutError) Timeout() bool   { return true }
func (tlsHandshakeTimeoutError) Temporary() bool { return true }

// TestRetryable 表驱动验证六类可重试网络错误的识别与不可重试错误的排除。
func TestRetryable(t *testing.T) {
	const testUrl = "https://console.tim.qq.com/v4/openim/send_msg"

	wrap := func(err error) error {
		return &url.Error{Op: "Post", URL: testUrl, Err: err}
	}

	tests := []struct {
		name string
		err  error
		want bool
	}{
		// 六类可重试错误
		{"dns no such host", wrap(&net.DNSError{Err: "no such host", Name: "console.tim.qq.com", IsNotFound: true}), true},
		{"dial connection timed out", wrap(&net.OpError{Op: "dial", Net: "tcp", Err: syscall.ETIMEDOUT}), true},
		{"read i/o timeout", wrap(&net.OpError{Op: "read", Net: "tcp", Err: os.ErrDeadlineExceeded}), true},
		{"tls handshake timeout", wrap(tlsHandshakeTimeoutError{}), true},
		{"client timeout exceeded", wrap(context.DeadlineExceeded), true},
		{"connection reset by peer", wrap(&net.OpError{Op: "read", Net: "tcp", Err: syscall.ECONNRESET}), true},
		{"eof", wrap(io.EOF), true},
		{"unexpected eof", wrap(io.ErrUnexpectedEOF), true},
		// 不可重试错误
		{"business error", NewError(1001, "invalid sig"), false},
		{"invalid response", invalidResponse, false},
		{"unmarshal error", fmt.Errorf("unmarshal response body: %w", errors.New("invalid character '<'")), false},
		{"dial connection refused", wrap(&net.OpError{Op: "dial", Net: "tcp", Err: syscall.ECONNREFUSED}), false},
		{"write broken pipe", wrap(&net.OpError{Op: "write", Net: "tcp", Err: syscall.EPIPE}), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := retryable(tt.err); got != tt.want {
				t.Errorf("retryable(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

// TestConcurrentRetry 并发请求持续故障端点：重试计数为局部变量，无共享状态，
// 在 -race 下不得出现数据竞争报告（旧实现的全局 retryCount-- 会触发 DATA RACE）。
func TestConcurrentRetry(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hijackAndClose(t, w, r)
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)

	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = c.Post("openim", "send_msg", map[string]string{"text": "hi"}, &types.ActionBaseResp{})
		}()
	}
	wg.Wait()
}
