package client

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
	"github.com/pulumi/pulumi/pkg/util/contract"
)

// Function represents an OpenFaaS function definition.
type Function struct {
	Service      string            `json:"service"`
	Network      string            `json:"network"`
	Image        string            `json:"image"`
	EnvProcess   string            `json:"envProcess"`
	EnvVars      map[string]string `json:"envVars"`
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	Secrets      []string          `json:"secrets"`
	RegistryAuth string            `json:"registryAuth"`
}

// Client is a simple client for the OpenFaaS REST API.
type Client struct {
	httpClient    *http.Client
	baseURL       string
	authorization string
}

// ErrNotFound is returned by the client if a resource cannot be found.
var ErrNotFound = errors.New("not found")

// NewClient creates a new OpenFaaS client with the given HTTP client, base URL, and optional credentials.
func NewClient(c *http.Client, baseURL, username, password string) *Client {
	authorization := ""
	if username != "" {
		authorization = "Basic " + base64.StdEncoding.EncodeToString([]byte(username+":"+password))
	}
	return &Client{
		httpClient:    c,
		baseURL:       baseURL,
		authorization: authorization,
	}
}

func (c *Client) do(ctx context.Context, method, path string, body []byte) (*http.Response, error) {
	req, err := http.NewRequest(method, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	if c.authorization != "" {
		req.Header.Set("Authorization", c.authorization)
	}
	resp, err := c.httpClient.Do(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	switch resp.StatusCode {
	case http.StatusOK, http.StatusCreated, http.StatusAccepted:
		return resp, nil
	case http.StatusNotFound:
		return nil, ErrNotFound
	default:
		defer contract.IgnoreClose(resp.Body)
		b, err := ioutil.ReadAll(resp.Body)
		contract.IgnoreError(err)
		return nil, errors.Errorf("%d response from server (%s)", resp.StatusCode, string(b))
	}
}

// CreateFunction creates a new function from the given function specification.
func (c *Client) CreateFunction(ctx context.Context, f *Function) error {
	body, err := json.Marshal(f)
	if err != nil {
		return err
	}

	_, err = c.do(ctx, "POST", "/system/functions", body)
	return err
}

// GetFunction gets the function specificiation for the function with the given name.
func (c *Client) GetFunction(ctx context.Context, name string) (*Function, error) {
	resp, err := c.do(ctx, "GET", "/system/function/"+url.PathEscape(name), nil)
	if err != nil {
		return nil, err
	}
	defer contract.IgnoreClose(resp.Body)

	var f Function
	if err := json.NewDecoder(resp.Body).Decode(&f); err != nil {
		return nil, err
	}
	return &f, nil
}

// UpdateFunction updates the function with the given specification.
func (c *Client) UpdateFunction(ctx context.Context, f *Function) error {
	body, err := json.Marshal(f)
	if err != nil {
		return err
	}

	_, err = c.do(ctx, "PUT", "/system/functions", body)
	return err
}

// DeleteFunction deletes the function with the given name.
func (c *Client) DeleteFunction(ctx context.Context, name string) error {
	body, err := json.Marshal(map[string]string{"functionName": name})
	if err != nil {
		return err
	}

	_, err = c.do(ctx, "DELETE", "/system/functions", body)
	return err
}
