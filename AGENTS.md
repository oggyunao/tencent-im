# AGENTS.md

Go SDK (module `github.com/oggyunao/tencent-im`, Go 1.19) wrapping the Tencent Cloud IM server REST API. Sole dependency: `github.com/robin-hzc/http`.

## Commands

- No Makefile, lint config, or CI. Verify with `go build ./...` and `go vet ./...`.
- All tests live in `im_test.go` (root package `im_test`) and are **live integration tests** against `https://console.tim.qq.com` with hardcoded AppId/AppSecret. They mutate real IM data (import/delete accounts, create groups, send messages). Do not run `go test ./...` without intent to hit the real service; there are no mocks or unit tests.
- Single test: `go test -run TestName .` (root dir only).

## Architecture

- `im.go` (package `im`) is the entrypoint: `NewIM(*Options)` returns an `IM` facade that lazily creates one sub-API per domain package (`account`, `sns`, `group`, `push`, `private`, `profile`, `operation`, `mute`, `recentcontact`, `callback`).
- Each domain package follows the same layout: `api.go` (interface + `NewAPI(core.Client)`), `types.go` (request/response structs), `enum.go` (constants).
- `internal/core`: HTTP client building `v4/{serviceName}/{command}` URLs with UserSig auth, plus the `Error` interface. `internal/sign`: UserSig generation. `internal/types`/`internal/entity`: shared request/response/message types.

## Conventions agents will miss

- Any response struct passed to `core.Client` methods **must** implement `types.BaseRespInterface` (or `ActionBaseRespInterface` for responses with `ActionStatus`) — normally by embedding `types.BaseResp` or `types.ActionBaseResp`. Otherwise the client returns an `invalidResponse` error instead of the API result.
- SDK errors surface as `im.Error` (alias of `core.Error`) with `Code()`/`Message()`; existing code checks them via type assertion (`err.(im.Error)`), not `errors.Is/As`.
- Doc comments and API method comments are in Chinese; keep that style when extending APIs. New API methods should link the matching `cloud.tencent.com/document/product/269/...` doc page, as existing ones do.
