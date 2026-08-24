/**
 * @Author: fuxiao
 * @Email: 576101059@qq.com
 * @Date: 2021/8/27 11:31 上午
 * @Desc: TODO
 */

package core

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/oggyunao/tencent-im/internal/enum"
	"github.com/oggyunao/tencent-im/internal/sign"
	"github.com/oggyunao/tencent-im/internal/types"
	"github.com/robin-hzc/http"
)

const (
	defaultBaseUrl     = "https://console.tim.qq.com"
	defaultVersion     = "v4"
	defaultContentType = "json"
	defaultExpiration  = 3600

	// defaultTimeout 单次请求整体超时（含拨号、TLS 握手、读取 body）。
	// 不设置会导致网络抖动/服务端慢响应时 TCP 读无限期挂起（read: connection timed out）。
	defaultTimeout = 30 * time.Second
	// defaultRetryCount 网络层错误（超时、连接失败、IO 等）重试次数，业务错误不重试。
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
	client          *http.Client
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

	c := new(client)
	c.opt = opt
	c.client = http.NewClient()
	c.client.SetContentType(http.ContentTypeJson)
	c.client.SetBaseUrl(baseUrl)
	c.client.SetTimeout(defaultTimeout)
	// 仅对网络层错误（超时/连接失败/IO）重试；业务错误（core.Error）在 Do 成功后由上层处理，不会触发此处重试
	c.client.SetRetry(defaultRetryCount, defaultRetryInterval)

	return c
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

// request Request请求
func (c *client) request(method, serviceName, command string, data, resp interface{}) error {
	res, err := c.client.Request(method, c.buildUrl(serviceName, command), data)
	if err != nil {
		return err
	}
	if err = res.ScanBody(resp); err != nil {
		return err
	}

	if r, ok := resp.(types.ActionBaseRespInterface); ok {
		if r.GetActionStatus() == enum.FailActionStatus {
			return NewError(r.GetErrorCode(), r.GetErrorInfo())
		}

		if r.GetErrorCode() != enum.SuccessCode {
			return NewError(r.GetErrorCode(), r.GetErrorInfo())
		}
	} else if r, ok := resp.(types.BaseRespInterface); ok {
		if r.GetErrorCode() != enum.SuccessCode {
			return NewError(r.GetErrorCode(), r.GetErrorInfo())
		}
	} else {
		return invalidResponse
	}

	return nil
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
