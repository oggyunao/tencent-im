# tencent-im

腾讯云即时通信 IM 服务端 Go SDK，基于官方 REST API 封装。

- 官方文档：[即时通信 IM](https://cloud.tencent.com/product/im/developer)
- 服务端 REST API：[API 文档](https://cloud.tencent.com/document/product/269)

## 如何使用

```shell script
go get github.com/oggyunao/tencent-im
```

## 调用方法

```go
package main

import (
    "fmt"
    "log"
    "net/http"

    "github.com/oggyunao/tencent-im"
    "github.com/oggyunao/tencent-im/account"
    "github.com/oggyunao/tencent-im/callback"
)

func main() {
    tim := im.NewIM(&im.Options{
        AppId:     1400579830,                                                          // 无效的AppId,请勿直接使用
        AppSecret: "0d2a321b087fdb8fd5ed5ea14fe0489139086eb1b03541283fc9feeab8f2bfd3", // 无效的AppSecret,请勿直接使用
        UserId:    "administrator",                                                     // 管理员用户账号，请在腾讯云IM后台设置管理账号
    })

    // 导入账号
    if err := tim.Account().ImportAccount(&account.Account{
        UserId:   "test1",
        Nickname: "测试账号1",
        FaceUrl:  "https://www.baidu.com/img/PCtm_d9c8750bed0b3c7d089fa7d55720d6cf.png",
    }); err != nil {
        if e, ok := err.(im.Error); ok {
            fmt.Println(fmt.Sprintf("import account failed, code:%d, message:%s.", e.Code(), e.Message()))
        } else {
            fmt.Println(fmt.Sprintf("import account failed:%s.", err.Error()))
        }
    }

    fmt.Println("import account success.")

    // 注册回调事件
    tim.Callback().Register(callback.EventAfterFriendAdd, func(ack callback.Ack, data interface{}) {
        fmt.Printf("%+v", data.(callback.AfterFriendAdd))
        _ = ack.AckSuccess(0)
    })

    // 注册回调事件
    tim.Callback().Register(callback.EventAfterFriendDelete, func(ack callback.Ack, data interface{}) {
        fmt.Printf("%+v", data.(callback.AfterFriendDelete))
        _ = ack.AckSuccess(0)
    })

    // 开启监听
    http.HandleFunc("/callback", func(writer http.ResponseWriter, request *http.Request) {
        tim.Callback().Listen(writer, request)
    })

    // 启动服务器
    if err := http.ListenAndServe(":8080", nil); err != nil {
        log.Fatal("ListenAndServe: ", err)
    }
}
```

## SDK 列表

各方法的详细说明见对应的官方文档链接，以及源码中的接口注释。

### 账号管理（Account）

| 接口 | 方法 | 说明 |
| --- | --- | --- |
| [导入单个帐号](https://cloud.tencent.com/document/product/269/1608) | `Account.ImportAccount` | 将 App 自有帐号导入 IM 帐号系统 |
| [导入多个帐号](https://cloud.tencent.com/document/product/269/4919) | `Account.ImportAccounts` | 批量导入帐号，单次最多 100 个 |
| [删除单个帐号](https://cloud.tencent.com/document/product/269/36443) | `Account.DeleteAccount` | 拓展自 DeleteAccounts，仅 IM 体验版帐号可删除 |
| [删除多个帐号](https://cloud.tencent.com/document/product/269/36443) | `Account.DeleteAccounts` | 单次最多 100 个，仅 IM 体验版帐号可删除 |
| [查询单个帐号导入状态](https://cloud.tencent.com/document/product/269/38417) | `Account.CheckAccount` | 拓展自 CheckAccounts |
| [查询多个帐号导入状态](https://cloud.tencent.com/document/product/269/38417) | `Account.CheckAccounts` | 批量查询帐号是否已导入，单次最多 100 个 |
| [失效帐号登录状态](https://cloud.tencent.com/document/product/269/3853) | `Account.KickAccount` | 使帐号当前的登录状态（UserSig）失效 |
| [查询单个帐号在线状态](https://cloud.tencent.com/document/product/269/2566) | `Account.GetAccountOnlineState` | 拓展自 GetAccountsOnlineState |
| [查询多个帐号在线状态](https://cloud.tencent.com/document/product/269/2566) | `Account.GetAccountsOnlineState` | 获取用户当前的登录状态 |

### 资料管理（Profile）

| 接口 | 方法 | 说明 |
| --- | --- | --- |
| [设置资料](https://cloud.tencent.com/document/product/269/1640) | `Profile.SetProfile` | 支持标配资料字段和自定义资料字段 |
| [拉取资料](https://cloud.tencent.com/document/product/269/1639) | `Profile.GetProfiles` | 支持拉取好友和非好友的资料，建议单次不超过 100 个用户 |

### 关系链管理（SNS）

| 接口 | 方法 | 说明 |
| --- | --- | --- |
| [添加单个好友](https://cloud.tencent.com/document/product/269/1643) | `SNS.AddFriend` | 拓展自 AddFriends |
| [添加多个好友](https://cloud.tencent.com/document/product/269/1643) | `SNS.AddFriends` | 批量添加好友 |
| [导入单个好友](https://cloud.tencent.com/document/product/269/8301) | `SNS.ImportFriend` | 拓展自 ImportFriends |
| [导入多个好友](https://cloud.tencent.com/document/product/269/8301) | `SNS.ImportFriends` | 批量导入单向好友，避免并发写冲突 |
| [更新单个好友](https://cloud.tencent.com/document/product/269/12525) | `SNS.UpdateFriend` | 拓展自 UpdateFriends |
| [更新多个好友](https://cloud.tencent.com/document/product/269/12525) | `SNS.UpdateFriends` | 批量更新同一用户的好友关系链数据 |
| [删除单个好友](https://cloud.tencent.com/document/product/269/1644) | `SNS.DeleteFriend` | 拓展自 DeleteFriends |
| [删除多个好友](https://cloud.tencent.com/document/product/269/1644) | `SNS.DeleteFriends` | 支持单向删除和双向删除 |
| [删除所有好友](https://cloud.tencent.com/document/product/269/1645) | `SNS.DeleteAllFriends` | 清除指定用户的标配好友数据和自定义好友数据 |
| [校验单个好友](https://cloud.tencent.com/document/product/269/1646) | `SNS.CheckFriend` | 拓展自 CheckFriends |
| [校验多个好友](https://cloud.tencent.com/document/product/269/1646) | `SNS.CheckFriends` | 批量校验好友关系，单次最多 100 个 |
| [拉取单个指定好友](https://cloud.tencent.com/document/product/269/8609) | `SNS.GetFriend` | 拓展自 GetFriends |
| [拉取多个指定好友](https://cloud.tencent.com/document/product/269/8609) | `SNS.GetFriends` | 拉取指定好友的好友数据和资料数据，建议单次不超过 100 个 |
| [拉取好友](https://cloud.tencent.com/document/product/269/1647) | `SNS.FetchFriends` | 分页拉取全量好友数据，不含资料数据 |
| [续拉取好友](https://cloud.tencent.com/document/product/269/1647) | `SNS.PullFriends` | 拓展自 FetchFriends，自动分页拉取全量好友 |
| [添加黑名单](https://cloud.tencent.com/document/product/269/3718) | `SNS.AddBlacklist` | 支持批量添加，单次最多 1000 个 |
| [删除黑名单](https://cloud.tencent.com/document/product/269/3719) | `SNS.DeleteBlacklist` | 支持批量删除，单次最多 1000 个 |
| [拉取黑名单](https://cloud.tencent.com/document/product/269/3722) | `SNS.FetchBlacklist` | 分页拉取所有黑名单 |
| [续拉取黑名单](https://cloud.tencent.com/document/product/269/3722) | `SNS.PullBlacklist` | 拓展自 FetchBlacklist，自动分页拉取全部黑名单 |
| [校验黑名单](https://cloud.tencent.com/document/product/269/3725) | `SNS.CheckBlacklist` | 批量校验黑名单关系 |
| [添加分组](https://cloud.tencent.com/document/product/269/10107) | `SNS.AddGroups` | 批量添加分组，并将指定好友加入新增分组 |
| [删除分组](https://cloud.tencent.com/document/product/269/10108) | `SNS.DeleteGroups` | 删除指定分组 |
| [拉取分组](https://cloud.tencent.com/document/product/269/54763) | `SNS.GetGroups` | 支持拉取分组下的好友列表 |

### 私聊消息（Private）

| 接口 | 方法 | 说明 |
| --- | --- | --- |
| [单发单聊消息](https://cloud.tencent.com/document/product/269/2282) | `Private.SendMessage` | 不检查好友关系和禁言状态 |
| [批量发单聊消息](https://cloud.tencent.com/document/product/269/1612) | `Private.SendMessages` | 单次最多 500 个用户，不触发回调 |
| [导入单聊消息](https://cloud.tencent.com/document/product/269/2568) | `Private.ImportMessage` | 导入历史单聊消息，不触发回调 |
| [查询单聊消息](https://cloud.tencent.com/document/product/269/42794) | `Private.FetchMessages` | 按时间范围查询单聊会话消息记录 |
| [续拉取单聊消息](https://cloud.tencent.com/document/product/269/42794) | `Private.PullMessages` | 拓展自 FetchMessages，自动续拉全部消息 |
| [撤回单聊消息](https://cloud.tencent.com/document/product/269/38980) | `Private.RevokeMessage` | 可撤回任何时间的单聊消息 |
| [设置单聊消息已读](https://cloud.tencent.com/document/product/269/50349) | `Private.SetMessageRead` | 将某个单聊会话的消息全部标记已读 |
| [查询单聊未读消息计数](https://cloud.tencent.com/document/product/269/56043) | `Private.GetUnreadMessageNum` | 查询单聊总未读数或单个会话未读数 |

### 全员推送（Push）

| 接口 | 方法 | 说明 |
| --- | --- | --- |
| [全员推送](https://cloud.tencent.com/document/product/269/45934) | `Push.PushMessage` | 支持全员推送、按用户属性推送、按用户标签推送 |
| [设置应用属性名称](https://cloud.tencent.com/document/product/269/45935) | `Push.SetAttrNames` | 最多可设置 10 个用户属性名称 |
| [获取应用属性名称](https://cloud.tencent.com/document/product/269/45936) | `Push.GetAttrNames` | 使用前请先设置应用属性名称 |
| [获取用户属性](https://cloud.tencent.com/document/product/269/45937) | `Push.GetUserAttrs` | 单次最多 100 个用户 |
| [设置用户属性](https://cloud.tencent.com/document/product/269/45938) | `Push.SetUserAttrs` | 单次最多 100 个用户 |
| [删除用户属性](https://cloud.tencent.com/document/product/269/45939) | `Push.DeleteUserAttrs` | 单次最多 100 个用户 |
| [获取用户标签](https://cloud.tencent.com/document/product/269/45940) | `Push.GetUserTags` | 单次最多 100 个用户 |
| [添加用户标签](https://cloud.tencent.com/document/product/269/45941) | `Push.AddUserTags` | 单次最多 100 个用户，单用户单次最多 10 个标签 |
| [删除用户标签](https://cloud.tencent.com/document/product/269/45942) | `Push.DeleteUserTags` | 单次最多 100 个用户 |
| [删除用户所有标签](https://cloud.tencent.com/document/product/269/45943) | `Push.DeleteUserAllTags` | 单次最多 100 个用户 |

### 全局禁言管理（Mute）

| 接口 | 方法 | 说明 |
| --- | --- | --- |
| [设置全局禁言](https://cloud.tencent.com/document/product/269/4230) | `Mute.SetNoSpeaking` | 设置单聊消息和群组消息的全局禁言 |
| [查询全局禁言](https://cloud.tencent.com/document/product/269/4229) | `Mute.GetNoSpeaking` | 查询单聊消息和群组消息的全局禁言状态 |

### 运营管理（Operation）

| 接口 | 方法 | 说明 |
| --- | --- | --- |
| [拉取运营数据](https://cloud.tencent.com/document/product/269/4193) | `Operation.GetOperationData` | 拉取最近 30 天的运营数据 |
| [下载最近消息记录](https://cloud.tencent.com/document/product/269/1650) | `Operation.GetHistoryData` | 获取最近 7 天内某天某小时的消息记录下载地址 |
| [获取服务器IP地址](https://cloud.tencent.com/document/product/269/45438) | `Operation.GetIPList` | 获取 SDK、第三方回调使用的服务器 IP 列表 |

### 群组管理（Group）

| 接口 | 方法 | 说明 |
| --- | --- | --- |
| [拉取App中的所有群组ID](https://cloud.tencent.com/document/product/269/1614) | `Group.FetchGroupIds` | 获取 App 中所有群组的 ID |
| [拉取App中的所有群组](https://cloud.tencent.com/document/product/269/1614) | `Group.FetchGroups` | 拓展自 FetchGroupIds，返回群组详细资料 |
| [续拉取App中的所有群组](https://cloud.tencent.com/document/product/269/1614) | `Group.PullGroups` | 拓展自 FetchGroups，自动分页拉取全部群组 |
| [创建群组](https://cloud.tencent.com/document/product/269/1615) | `Group.CreateGroup` | 支持自定义群组 ID、指定群主和初始成员 |
| [获取单个群详细资料](https://cloud.tencent.com/document/product/269/1616) | `Group.GetGroup` | 拓展自 GetGroups |
| [获取多个群详细资料](https://cloud.tencent.com/document/product/269/1616) | `Group.GetGroups` | 单次最多 50 个群 |
| [拉取群成员详细资料](https://cloud.tencent.com/document/product/269/1617) | `Group.FetchMembers` | Limit 为 0 时获取群内全部成员 |
| [续拉取群成员详细资料](https://cloud.tencent.com/document/product/269/1617) | `Group.PullMembers` | 拓展自 FetchMembers，自动分页拉取全部成员 |
| [修改群基础资料](https://cloud.tencent.com/document/product/269/1620) | `Group.UpdateGroup` | 修改群名称、头像、简介、公告等基础信息 |
| [增加群成员](https://cloud.tencent.com/document/product/269/1621) | `Group.AddMembers` | 向指定群中添加新成员 |
| [删除群成员](https://cloud.tencent.com/document/product/269/1622) | `Group.DeleteMembers` | 支持静默删人和填写踢出原因 |
| [修改群成员资料](https://cloud.tencent.com/document/product/269/1623) | `Group.UpdateMember` | 修改成员身份、群名片、禁言时间等 |
| [解散群组](https://cloud.tencent.com/document/product/269/1624) | `Group.DestroyGroup` | 解散指定群组 |
| [拉取用户所加入的群组](https://cloud.tencent.com/document/product/269/1625) | `Group.FetchMemberGroups` | 默认不返回未激活的 Work 群和 AVChatRoom 群 |
| [续拉取用户所加入的群组](https://cloud.tencent.com/document/product/269/1625) | `Group.PullMemberGroups` | 拓展自 FetchMemberGroups，自动分页拉取 |
| [查询用户在群组中的身份](https://cloud.tencent.com/document/product/269/1626) | `Group.GetRolesInGroup` | 批量查询用户的成员角色 |
| [批量禁言](https://cloud.tencent.com/document/product/269/1627) | `Group.ForbidSendMessage` | 禁言时间单位为秒，0 表示取消禁言，4294967295 为永久禁言 |
| [取消禁言](https://cloud.tencent.com/document/product/269/1627) | `Group.AllowSendMessage` | 拓展自 ForbidSendMessage |
| [获取被禁言群成员列表](https://cloud.tencent.com/document/product/269/2925) | `Group.GetShuttedUpMembers` | 获取群组中被禁言的用户列表 |
| [在群组中发送普通消息](https://cloud.tencent.com/document/product/269/1629) | `Group.SendMessage` | 支持 @指定成员或全体成员、指定接收者 |
| [直播群广播消息](https://cloud.tencent.com/document/product/269/77402) | `Group.SendBroadcastMessage` | 向所有直播群（AVChatRoom）下发广播消息 |
| [在群组中发送系统通知](https://cloud.tencent.com/document/product/269/1630) | `Group.SendNotification` | 支持全员下发或指定成员下发 |
| [转让群主](https://cloud.tencent.com/document/product/269/1633) | `Group.ChangeGroupOwner` | 新群主必须为群内成员 |
| [撤回单条群消息](https://cloud.tencent.com/document/product/269/12341) | `Group.RevokeMessage` | 拓展自 RevokeMessages |
| [撤回多条群消息](https://cloud.tencent.com/document/product/269/12341) | `Group.RevokeMessages` | 消息需要在漫游有效期以内 |
| [导入群基础资料](https://cloud.tencent.com/document/product/269/1634) | `Group.ImportGroup` | 用于从其他 IM 系统迁移存量群组数据 |
| [导入群消息](https://cloud.tencent.com/document/product/269/1635) | `Group.ImportMessages` | 用于从其他 IM 系统迁移存量群消息数据 |
| [导入多个群成员](https://cloud.tencent.com/document/product/269/1636) | `Group.ImportMembers` | 用于从其他 IM 系统迁移存量群成员数据 |
| [设置成员未读消息计数](https://cloud.tencent.com/document/product/269/1637) | `Group.SetMemberUnreadMsgNum` | 用于迁移场景下设置群成员未读消息数 |
| [撤回指定用户发送的消息](https://cloud.tencent.com/document/product/269/2359) | `Group.RevokeMemberMessages` | 撤回最近 1000 条消息中指定用户发送的消息 |
| [拉取群历史消息](https://cloud.tencent.com/document/product/269/2738) | `Group.FetchMessages` | 按 Seq 倒序拉取，单次最多 20 条 |
| [续拉取群历史消息](https://cloud.tencent.com/document/product/269/2738) | `Group.PullMessages` | 拓展自 FetchMessages，自动续拉历史消息 |
| [获取直播群在线人数](https://cloud.tencent.com/document/product/269/49180) | `Group.GetOnlineMemberNum` | 仅适用于直播群（AVChatRoom） |
| [获取群自定义属性](https://cloud.tencent.com/document/product/269/67012) | `Group.GetGroupAttr` | 获取群维度的自定义属性 |
| [修改群自定义属性](https://cloud.tencent.com/document/product/269/67010) | `Group.ModifyGroupAttr` | key 最多 16 个，单个 key 最大 32 字节 |
| [清空群自定义属性](https://cloud.tencent.com/document/product/269/67009) | `Group.ClearGroupAttr` | 清空指定群的全部自定义属性 |
| [群成员封禁](https://cloud.tencent.com/document/product/269/79249) | `Group.BanMember` | 封禁后成员无法接收消息，单次最多 20 个成员 |
| [群成员解封](https://cloud.tencent.com/document/product/269/79250) | `Group.UnbanMember` | 解封后成员可重新进群接收消息 |

### 最近联系人（RecentContact）

| 接口 | 方法 | 说明 |
| --- | --- | --- |
| [拉取会话列表](https://cloud.tencent.com/document/product/269/62118) | `RecentContact.FetchSessions` | 分页拉取会话列表 |
| [续拉取会话列表](https://cloud.tencent.com/document/product/269/62118) | `RecentContact.PullSessions` | 拓展自 FetchSessions，自动分页拉取全部会话 |
| [删除单个会话](https://cloud.tencent.com/document/product/269/62119) | `RecentContact.DeleteSession` | 支持同步清理漫游消息 |

### 回调与其他（Callback）

| 接口 | 方法 | 说明 |
| --- | --- | --- |
| 注册回调事件 | `Callback.Register` | 注册第三方回调事件处理器，事件类型见 `callback` 包常量 |
| 监听回调请求 | `Callback.Listen` | 在 HTTP Handler 中处理腾讯云回调请求并应答 |
| 获取UserSig签名 | `IM.GetUserSig` | 本地生成 UserSig 签名，非 REST API |
