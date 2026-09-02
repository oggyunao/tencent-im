/**
 * @Author: fuxiao
 * @Email: 576101059@qq.com
 * @Date: 2021/8/27 11:31 上午
 * @Desc: HTTP客户端（标准库实现，含网络层错误重试）
 */

package core

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/oggyunao/tencent-im/internal/enum"
	"github.com/oggyunao/tencent-im/internal/sign"
	"github.com/oggyunao/tencent-im/internal/types"
)

const (
	defaultBaseUrl     = "https://console.tim.qq.com"
	defaultVersion     = "v4"
	defaultContentType = "json"
	defaultExpiration  = 3600

	// jsonContentType 请求体编码类型。
	jsonContentType = "application/json"

	// defaultTimeout 单次请求整体超时（含拨号、TLS 握手、读取 body）。
	// 不设置会导致网络抖动/服务端慢响应时 TCP 读无限期挂起（read: connection timed out）。
	defaultTimeout = 30 * time.Second
	// defaultRetryCount 网络层错误重试次数，业务错误与解析错误不重试。
	defaultRetryCount = 2
	// defaultRetryInterval 网络层错误重试间隔。
	defaultRetryInterval = 100 * time.Millisecond
)

var invalidResponse = NewError(enum.InvalidResponseCode, "invalid response")

type Client interface {
	// Get GET请求
	Get(serviceName string, command string, data interface{}, resp interface{}) error
	// Post POST请求
	Post(serviceName string, command string, data interface{}, resp interface{}) error
	// Put PUT请求
	Put(serviceName string, command string, data interface{}, resp interface{}) error
	// Patch PATCH请求
	Patch(serviceName string, command string, data interface{}, resp interface{}) error
	// Delete DELETE请求
	Delete(serviceName string, command string, data interface{}, resp interface{}) error
}

type client struct {
	http            *http.Client
	baseUrl         string
	opt             *Options
	mu              sync.Mutex
	userSig         string
	userSigExpireAt int64
}

type Options struct {
	BaseUrl    string
	AppId      int    // 应用SDKAppID，可在即时通信 IM 控制台 的应用卡片中获取。
	AppSecret  string // 密钥信息，可在即时通信 IM 控制台 的应用详情页面中获取，具体操作请参见 获取密钥
	UserId     string // 用户ID
	Expiration int    // UserSig过期时间
}

func NewClient(opt *Options) Client {
	baseUrl := opt.BaseUrl
	if baseUrl == "" {
		baseUrl = defaultBaseUrl
	}

	// 复用标准库默认 Transport：开启连接复用与空闲连接回收，可显著减少
	// TLS handshake timeout / connection reset 类错误（旧实现每个请求新建连接）；
	// 复用陈旧连接偶发的 EOF/reset 由重试层兜底
	transport := http.DefaultTransport.(*http.Transport).Clone()
	// 行为兼容：与旧实现一致跳过证书校验，待评估后放开
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec

	return &client{
		http: &http.Client{
			Transport: transport,
			Timeout:   defaultTimeout,
		},
		baseUrl: strings.TrimRight(baseUrl, "/"),
		opt:     opt,
	}
}

// Get GET请求
func (c *client) Get(serviceName string, command string, data interface{}, resp interface{}) error {
	return c.request(http.MethodGet, serviceName, command, data, resp)
}

// Post POST请求
func (c *client) Post(serviceName string, command string, data interface{}, resp interface{}) error {
	return c.request(http.MethodPost, serviceName, command, data, resp)
}

// Put PUT请求
func (c *client) Put(serviceName string, command string, data interface{}, resp interface{}) error {
	return c.request(http.MethodPut, serviceName, command, data, resp)
}

// Patch PATCH请求
func (c *client) Patch(serviceName string, command string, data interface{}, resp interface{}) error {
	return c.request(http.MethodPatch, serviceName, command, data, resp)
}

// Delete DELETE请求
func (c *client) Delete(serviceName string, command string, data interface{}, resp interface{}) error {
	return c.request(http.MethodDelete, serviceName, command, data, resp)
}

// request 发起请求，仅对网络层错误（见 retryable）做有限次重试；
// 业务错误与 JSON 解析错误直接返回，不重试。
// 重试计数为函数内局部变量，无共享状态，并发调用天然安全。
func (c *client) request(method, serviceName, command string, data, resp interface{}) error {
	body, err := marshalBody(data)
	if err != nil {
		return err
	}

	for attempt := 0; ; attempt++ {
		// 每次尝试重新构建 URL（新 random）与请求对象，body 全新序列化，请求天然可重放
		target := c.baseUrl + c.buildUrl(serviceName, command)
		if err := c.doOnce(method, target, body, resp); err != nil {
			if !retryable(err) || attempt >= defaultRetryCount {
				return err
			}
			time.Sleep(defaultRetryInterval)
			continue
		}
		return nil
	}
}

// doOnce 发起一次 HTTP 请求并解析响应，重试仅包裹本函数覆盖的
// “HTTP 往返 + 响应体读取”阶段；JSON 解码与业务错误判断不会被重试。
func (c *client) doOnce(method, url string, body []byte, resp interface{}) error {
	var reader io.Reader
	if len(body) > 0 {
		reader = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", jsonContentType)

	httpResp, err := c.http.Do(req)
	if err != nil {
		// 传输层错误原样透传（形如 *url.Error 包裹 net.OpError 等），是否重试由 retryable 判定
		return err
	}
	defer httpResp.Body.Close()

	data, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return err
	}

	return parseBody(data, resp)
}

// parseBody 解析响应体并进行业务错误判断。
func parseBody(data []byte, resp interface{}) error {
	if err := json.Unmarshal(data, resp); err != nil {
		return fmt.Errorf("unmarshal response body: %w", err)
	}

	if r, ok := resp.(types.ActionBaseRespInterface); ok {
		if r.GetActionStatus() == enum.FailActionStatus {
			return NewError(r.GetErrorCode(), r.GetErrorInfo())
		}

		if r.GetErrorCode() != enum.SuccessCode {
			return NewError(r.GetErrorCode(), r.GetErrorInfo())
		}
		return nil
	}

	if r, ok := resp.(types.BaseRespInterface); ok {
		if r.GetErrorCode() != enum.SuccessCode {
			return NewError(r.GetErrorCode(), r.GetErrorInfo())
		}
		return nil
	}

	return invalidResponse
}

// marshalBody 序列化请求体，data 为 nil 时不携带请求体。
func marshalBody(data interface{}) ([]byte, error) {
	if data == nil {
		return nil, nil
	}

	body, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("marshal request body: %w", err)
	}

	return body, nil
}

// retryable 判断 err 是否为可重试的网络层错误。
// 错误链形如 *url.Error{Op:"Post", Err: ...}，url.Error 实现 Unwrap，
// errors.As/Is 可穿透解包；不使用已废弃的 net.Error.Temporary()。
// 覆盖生产实测的六类错误：
//   - no such host           -> *net.DNSError
//   - connection timed out   -> dial 阶段 *net.OpError{Op:"dial"}，Timeout()==true
//   - i/o timeout            -> os.ErrDeadlineExceeded / context.DeadlineExceeded
//   - TLS handshake timeout  -> transport 的 tlsHandshakeTimeoutError（实现 net.Error）
//   - connection reset by peer -> syscall.ECONNRESET
//   - EOF / unexpected EOF   -> io.EOF / io.ErrUnexpectedEOF（响应读取中断）
//
// 注意：connection reset by peer / EOF / 读阶段超时意味着请求可能已被服务端处理，
// 重试为 at-least-once 语义（发消息场景可配合 MsgSeq 去重）；而拨号、DNS、
// TLS 握手类失败时请求确定未发出，重试无副作用。
func retryable(err error) bool {
	// 域名解析失败：请求未发出
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return true
	}

	// 连接被重置：连接可能已建立并处理请求
	if errors.Is(err, syscall.ECONNRESET) {
		return true
	}

	// 响应中断（服务端在返回完整响应前关闭连接）
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}

	// 超时类：拨号超时、读写超时、TLS 握手超时、Client.Timeout
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

// buildUrl 构建一个请求URL
func (c *client) buildUrl(serviceName string, command string) string {
	format := "/%s/%s/%s?sdkappid=%d&identifier=%s&usersig=%s&random=%d&contenttype=%s"
	random := rand.Int31()
	userSig := c.getUserSig()
	return fmt.Sprintf(format, defaultVersion, serviceName, command, c.opt.AppId, c.opt.UserId, userSig, random, defaultContentType)
}

// getUserSig 获取签名
func (c *client) getUserSig() string {
	c.mu.Lock()
	defer c.mu.Unlock()

	now, expiration := time.Now(), c.opt.Expiration

	if expiration <= 0 {
		expiration = defaultExpiration
	}

	// 提前 5 分钟刷新，避免边界时钟偏差导致 UserSig expired 错误
	const refreshSlack = 5 * 60
	refreshThreshold := c.userSigExpireAt - refreshSlack

	if c.userSig == "" || now.Unix() >= refreshThreshold {
		c.userSig, _ = sign.GenUserSig(c.opt.AppId, c.opt.AppSecret, c.opt.UserId, expiration)
		c.userSigExpireAt = now.Add(time.Duration(expiration) * time.Second).Unix()
	}

	return c.userSig
}
