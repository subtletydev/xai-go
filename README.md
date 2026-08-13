# xai-go

A Go SDK for the [xAI](https://x.ai) API, speaking gRPC directly.

The `gen/` tree holds the Go code generated from xAI's protobuf definitions; `client.go` adds a
thin client that owns the connection, attaches your API key to every call, and exposes each
service as a field.

This is an independent SDK — it is not published or endorsed by xAI.

> **Status:** early. The generated types track xAI's protos and can change with them, and the
> hand-written client surface may still shift. Pin a version.

## Requirements

- Go 1.26 or newer
- An xAI API key from [console.x.ai](https://console.x.ai)

## Install

```sh
go get github.com/subtletydev/xai-go
```

## Quick start

```go
package main

import (
	"context"
	"fmt"
	"log"

	xai "github.com/subtletydev/xai-go"
	xaiv1 "github.com/subtletydev/xai-go/gen/go/xai/api/v1"
)

func main() {
	// Reads the API key from $XAI_API_KEY.
	client, err := xai.NewClient()
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	resp, err := client.Chat.GetCompletion(context.Background(), &xaiv1.GetCompletionsRequest{
		Model: "grok-4",
		Messages: []*xaiv1.Message{{
			Role: xaiv1.MessageRole_ROLE_USER,
			Content: []*xaiv1.Content{{
				Content: &xaiv1.Content_Text{Text: "In one sentence: why is the sky blue?"},
			}},
		}},
	})
	if err != nil {
		log.Fatal(err)
	}

	for _, out := range resp.GetOutputs() {
		fmt.Println(out.GetMessage().GetContent())
	}
}
```

The Go package is named `xai_go`, so most code imports it under an alias (`xai` above).

## Authentication

The API key is sent as an `authorization: Bearer <key>` header on every RPC. It is read from the
`XAI_API_KEY` environment variable unless you pass one explicitly:

```go
client, err := xai.NewClient(xai.WithAPIKey(key))
```

Constructing a client without a key from either source is an error. Because the credentials are
per-RPC, gRPC refuses to send them over a plaintext connection unless you opt in with
`WithInsecure`.

## Client options

| Option | Purpose |
| --- | --- |
| `WithAPIKey(key)` | API key to send; defaults to `$XAI_API_KEY`. |
| `WithEndpoint(hostport)` | Override the endpoint (default `api.x.ai:443`). |
| `WithTLSConfig(cfg)` | Custom TLS settings, e.g. a pinned root pool. |
| `WithInsecure()` | Disable transport security — for local/test endpoints only. |
| `WithUserAgent(ua)` | Set the user agent prefix sent to the server. |
| `WithDialOptions(opts...)` | Append raw `grpc.DialOption`s (interceptors, keepalive, retry policy, …). |

The connection is created lazily, so `NewClient` succeeding does not mean the endpoint is
reachable — the first RPC surfaces connection errors. Set deadlines with `context.WithTimeout` per
call, and always `defer client.Close()`.

## Streaming

Streaming methods return a `grpc.ServerStreamingClient`, which you drain until `io.EOF`:

```go
stream, err := client.Chat.GetCompletionChunk(ctx, req)
if err != nil {
	log.Fatal(err)
}
for {
	chunk, err := stream.Recv()
	if errors.Is(err, io.EOF) {
		break
	}
	if err != nil {
		log.Fatal(err)
	}
	for _, out := range chunk.GetOutputs() {
		fmt.Print(out.GetDelta().GetContent())
	}
}
```

## Services

Each field on `Client` is a generated gRPC service client:

| Field | Service | Methods |
| --- | --- | --- |
| `Chat` | Chat completions | `GetCompletion`, `GetCompletionChunk`, `StartDeferredCompletion`, `GetDeferredCompletion`, `GetStoredCompletion`, `DeleteStoredCompletion`, `CompactContext` |
| `Models` | Model discovery | `ListLanguageModels`, `ListEmbeddingModels`, `ListImageGenerationModels`, `GetLanguageModel`, `GetEmbeddingModel`, `GetImageGenerationModel` |
| `Sample` | Raw text sampling | `SampleText`, `SampleTextStreaming` |
| `Embedder` | Embeddings | `Embed` |
| `Image` | Image generation | `GenerateImage` |
| `Video` | Video generation | `GenerateVideo`, `ExtendVideo`, `GetDeferredVideo` |
| `Files` | File storage | `UploadFile`, `ListFiles`, `RetrieveFile`, `RetrieveFileContent`, `DeleteFile` |
| `Documents` | Document search | `Search` |
| `Batch` | Batch jobs | `CreateBatch`, `GetBatch`, `ListBatches`, `CancelBatch`, `AddBatchRequests`, `ListBatchRequestMetadata`, `ListBatchResults`, `GetBatchRequestResult` |
| `Tokenize` | Tokenization | `TokenizeText` |
| `Auth` | Key introspection | `GetApiKeyInfo` |

Request and response types live in `gen/go/xai/api/v1`; consult that package (or xAI's API docs)
for the full field set.

## Management API

Billing and account management live behind a separate endpoint and get their own client:

```go
mgmt, err := xai.NewManagementClient()
if err != nil {
	log.Fatal(err)
}
defer mgmt.Close()

info, err := mgmt.UI.GetBillingInfo(ctx, &mgmtv1.GetBillingInfoReq{})
```

`ManagementClient` takes the same options and defaults to `management-api.x.ai:443`. Its `UI` field
covers billing info, payment methods, invoices, prepaid balance, and spending limits.

## Escape hatches

`Conn()` returns the underlying `*grpc.ClientConn` for anything the wrapper does not cover — using
a generated service client that is not exposed as a field, or wrapping calls in your own
interceptors:

```go
client, err := xai.NewClient(xai.WithDialOptions(
	grpc.WithChainUnaryInterceptor(myLoggingInterceptor),
))

raw := xaiv1.NewChatClient(client.Conn())
```

Errors come back as ordinary gRPC status errors, so `status.FromError` and `codes.*` work as usual.

## Regenerating the protos

The code under `gen/` is generated from xAI's `.proto` definitions and is checked in. Two notes if
you regenerate it:

- The generated `go_package` options point at `github.com/xai-org/xai-proto/...`, which does not
  publish those packages. The imports in `gen/go/xai/management_api/v1` are rewritten to this
  module's path so the tree compiles; re-apply that (or set a `go_package`/`M` override at
  generation time) after regenerating.
- New services need a corresponding field added to `Client` in `client.go`.

## Contributing

Issues and pull requests are welcome. Please run `go build ./...` and `go vet ./...` before
submitting, and keep hand-written code out of `gen/`.

## License

[MIT](LICENSE). The code under `gen/` is generated from xAI's protobuf definitions and remains
subject to their terms.
