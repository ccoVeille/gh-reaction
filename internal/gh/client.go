package gh

import (
	"context"
	"io"
	"net/http"

	"github.com/cli/go-gh/v2/pkg/api"
)

type RESTClient struct {
	client *api.RESTClient
}

type ClientOptions = api.ClientOptions

func DefaultRESTClient() (*RESTClient, error) {
	return NewRESTClient(ClientOptions{})
}

func NewRESTClient(opts ClientOptions) (*RESTClient, error) {
	client, err := api.NewRESTClient(opts)
	if err != nil {
		return nil, err
	}
	return &RESTClient{client: client}, nil
}

// Get calls the GitHub API to retrieve a resource, it mimics [api.RESTClient.Get], but
// adds context awareness.
func (c *RESTClient) Get(ctx context.Context, path string, response any) error {
	return c.client.DoWithContext(ctx, http.MethodGet, path, http.NoBody, response)
}

// Post calls the GitHub API to create a resource, it mimics [api.RESTClient.Post], but
// adds context awareness.
func (c *RESTClient) Post(ctx context.Context, path string, body io.Reader, response any) error {
	return c.client.DoWithContext(ctx, http.MethodPost, path, body, response)
}

func (c *RESTClient) Delete(ctx context.Context, path string, response any) error {
	return c.client.DoWithContext(ctx, http.MethodDelete, path, http.NoBody, response)
}

func (c *RESTClient) Patch(ctx context.Context, path string, body io.Reader, response any) error {
	return c.client.DoWithContext(ctx, http.MethodPatch, path, body, response)
}

func (c *RESTClient) Put(ctx context.Context, path string, body io.Reader, response any) error {
	return c.client.DoWithContext(ctx, http.MethodPut, path, body, response)
}

// Request sends an HTTP request to the GitHub API and returns the response.
// It's an alias for [api.RESTClient.RequestWithContext] provided for consistency.
func (c *RESTClient) Request(ctx context.Context, method string, path string, body io.Reader) (*http.Response, error) {
	return c.client.RequestWithContext(ctx, method, path, body)
}
