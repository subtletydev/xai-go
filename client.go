// Package xai_go provides a Go client for the xAI gRPC APIs.
package xai_go

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	xaiv1 "github.com/subtletydev/xai-go/gen/go/xai/api/v1"
	mgmtv1 "github.com/subtletydev/xai-go/gen/go/xai/management_api/v1"
)

const (
	// DefaultEndpoint is the host:port of the public xAI API.
	DefaultEndpoint = "api.x.ai:443"
	// DefaultManagementEndpoint is the host:port of the xAI management API.
	DefaultManagementEndpoint = "management-api.x.ai:443"

	// APIKeyEnvVar is consulted when no API key is passed explicitly.
	APIKeyEnvVar = "XAI_API_KEY"
)

// Client is a connection to the xAI API. It owns a single gRPC connection that
// is shared by every service client hanging off of it, and must be closed with
// Close when no longer needed. A Client is safe for concurrent use.
type Client struct {
	conn *grpc.ClientConn

	Auth      xaiv1.AuthClient
	Batch     xaiv1.BatchMgmtClient
	Chat      xaiv1.ChatClient
	Documents xaiv1.DocumentsClient
	Embedder  xaiv1.EmbedderClient
	Files     xaiv1.FilesClient
	Image     xaiv1.ImageClient
	Models    xaiv1.ModelsClient
	Sample    xaiv1.SampleClient
	Tokenize  xaiv1.TokenizeClient
	Video     xaiv1.VideoClient
}

// ManagementClient is a connection to the xAI management API. Like Client it
// owns its gRPC connection and must be closed with Close.
type ManagementClient struct {
	conn *grpc.ClientConn

	UI mgmtv1.UISvcClient
}

type options struct {
	endpoint    string
	apiKey      string
	insecure    bool
	tlsConfig   *tls.Config
	userAgent   string
	dialOptions []grpc.DialOption
}

// Option configures a Client or ManagementClient.
type Option func(*options)

// WithEndpoint overrides the host:port the client connects to.
func WithEndpoint(endpoint string) Option {
	return func(o *options) { o.endpoint = endpoint }
}

// WithAPIKey sets the API key sent as a bearer token on every RPC. When unset,
// the XAI_API_KEY environment variable is used.
func WithAPIKey(key string) Option {
	return func(o *options) { o.apiKey = key }
}

// WithInsecure disables transport security. It is intended for local testing
// against a plaintext endpoint; the API key is still sent, so only use it with
// endpoints you trust.
func WithInsecure() Option {
	return func(o *options) { o.insecure = true }
}

// WithTLSConfig overrides the TLS configuration used for the connection.
func WithTLSConfig(cfg *tls.Config) Option {
	return func(o *options) { o.tlsConfig = cfg }
}

// WithUserAgent sets the user agent prefix reported to the server.
func WithUserAgent(ua string) Option {
	return func(o *options) { o.userAgent = ua }
}

// WithDialOptions appends raw gRPC dial options, applied after the ones the
// client derives from the other Options.
func WithDialOptions(opts ...grpc.DialOption) Option {
	return func(o *options) { o.dialOptions = append(o.dialOptions, opts...) }
}

// NewClient returns a Client for the xAI API. The connection is established
// lazily, so a returned Client does not imply the endpoint is reachable.
func NewClient(opts ...Option) (*Client, error) {
	conn, err := dial(DefaultEndpoint, opts)
	if err != nil {
		return nil, err
	}
	return &Client{
		conn:      conn,
		Auth:      xaiv1.NewAuthClient(conn),
		Batch:     xaiv1.NewBatchMgmtClient(conn),
		Chat:      xaiv1.NewChatClient(conn),
		Documents: xaiv1.NewDocumentsClient(conn),
		Embedder:  xaiv1.NewEmbedderClient(conn),
		Files:     xaiv1.NewFilesClient(conn),
		Image:     xaiv1.NewImageClient(conn),
		Models:    xaiv1.NewModelsClient(conn),
		Sample:    xaiv1.NewSampleClient(conn),
		Tokenize:  xaiv1.NewTokenizeClient(conn),
		Video:     xaiv1.NewVideoClient(conn),
	}, nil
}

// NewManagementClient returns a Client for the xAI management API.
func NewManagementClient(opts ...Option) (*ManagementClient, error) {
	conn, err := dial(DefaultManagementEndpoint, opts)
	if err != nil {
		return nil, err
	}
	return &ManagementClient{
		conn: conn,
		UI:   mgmtv1.NewUISvcClient(conn),
	}, nil
}

// Conn exposes the underlying gRPC connection, for service clients that are not
// surfaced as fields or for interceptor-based wrapping.
func (c *Client) Conn() *grpc.ClientConn { return c.conn }

// Close releases the underlying gRPC connection.
func (c *Client) Close() error { return c.conn.Close() }

// Conn exposes the underlying gRPC connection.
func (c *ManagementClient) Conn() *grpc.ClientConn { return c.conn }

// Close releases the underlying gRPC connection.
func (c *ManagementClient) Close() error { return c.conn.Close() }

func dial(defaultEndpoint string, opts []Option) (*grpc.ClientConn, error) {
	o := options{
		endpoint: defaultEndpoint,
		apiKey:   os.Getenv(APIKeyEnvVar),
	}
	for _, opt := range opts {
		opt(&o)
	}
	if o.endpoint == "" {
		return nil, fmt.Errorf("xai: endpoint is empty")
	}
	if o.apiKey == "" {
		return nil, fmt.Errorf("xai: no API key: pass WithAPIKey or set %s", APIKeyEnvVar)
	}

	dialOpts := []grpc.DialOption{
		grpc.WithPerRPCCredentials(bearerToken{key: o.apiKey, allowInsecure: o.insecure}),
	}
	if o.insecure {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	} else {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(credentials.NewTLS(o.tlsConfig)))
	}
	if o.userAgent != "" {
		dialOpts = append(dialOpts, grpc.WithUserAgent(o.userAgent))
	}
	dialOpts = append(dialOpts, o.dialOptions...)

	conn, err := grpc.NewClient(o.endpoint, dialOpts...)
	if err != nil {
		return nil, fmt.Errorf("xai: dial %s: %w", o.endpoint, err)
	}
	return conn, nil
}

// bearerToken attaches the API key to every outgoing RPC.
type bearerToken struct {
	key           string
	allowInsecure bool
}

func (t bearerToken) GetRequestMetadata(context.Context, ...string) (map[string]string, error) {
	return map[string]string{"authorization": "Bearer " + t.key}, nil
}

func (t bearerToken) RequireTransportSecurity() bool { return !t.allowInsecure }
