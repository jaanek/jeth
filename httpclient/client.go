package httpclient

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/jaanek/jeth/ui"
)

var DefaultRetryMax = 1

func NewDefault(ui ui.Screen) HttpClient {
	return New(ui, DefaultRetryMax)
}

func New(ui ui.Screen, retryMax int) HttpClient {
	return &httpClient{
		ui: ui,
		client: http.Client{
			Timeout: time.Second * 1 * 60,
			// Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
		},
		RetryMax:       retryMax,
		RetryCheck:     WithDefaultRetryPolicy(),
		RetryWaitDelay: WithDefaultRetryWaitDelay(),
	}
}

type HttpClient interface {
	Post(url, contentType string, body io.ReadSeeker) (resp *http.Response, err error)
	Get(url string) (resp *http.Response, err error)
	Do(req *Request) (*http.Response, error)
}

type httpClient struct {
	ui             ui.Screen
	client         http.Client
	RetryCheck     RetryCheck
	RetryMax       int
	RetryWaitDelay RetryWaitDelay
	ErrorHandler   ErrorHandler
}

type RetryCheck func(req *Request, resp *http.Response, err error) (bool, error)
type RetryWaitDelay func(attemptNum int, resp *http.Response) time.Duration
type ErrorHandler func(resp *http.Response, err error, numTries int) (*http.Response, error)

func WithDefaultRetryPolicy() RetryCheck {
	return func(req *Request, resp *http.Response, err error) (bool, error) {
		if err != nil {
			return true, err
		}
		if resp.StatusCode == 0 || resp.StatusCode >= 500 {
			return true, nil
		}
		// Request Throttling - Too many requests
		if resp.StatusCode == http.StatusTooManyRequests {
			return true, nil
		}
		return false, nil
	}
}

func WithDefaultRetryWaitDelay() RetryWaitDelay {
	return func(attemptNum int, resp *http.Response) (waitDelay time.Duration) {
		waitDelay = time.Second * 10

		// on "net/http: TLS handshake timeout" the resp is nil
		if resp == nil {
			return
		}

		// Request Throttling - Too many requests
		if resp.StatusCode == http.StatusTooManyRequests {
			waitDelay = time.Minute * 1
		} else if resp.StatusCode == http.StatusUnauthorized {
			waitDelay = time.Second * 1
		}
		return
	}
}

// Request wraps the metadata needed to create HTTP requests.
type Request struct {
	body io.ReadSeeker
	*http.Request
}

func NewRequest(method, url string, body io.ReadSeeker) (*Request, error) {
	var rcBody io.ReadCloser
	if body != nil {
		rcBody = ioutil.NopCloser(body)
	}

	httpReq, err := http.NewRequest(method, url, rcBody)
	if err != nil {
		return nil, err
	}

	return &Request{body, httpReq}, nil
}

func (c *httpClient) Do(req *Request) (*http.Response, error) {
	var resp *http.Response
	var err error
	defer c.client.CloseIdleConnections()

	for i := 1; ; i++ {
		c.ui.Logf("%s %s\n", req.Method, req.URL)

		// Always rewind the request body when non-nil
		if req.body != nil {
			if _, err := req.body.Seek(0, 0); err != nil {
				return nil, fmt.Errorf("failed to seek body %w", err)
			}
		}

		// Attempt the request
		resp, err = c.client.Do(req.Request)
		if err != nil {
			c.ui.Errorf("%s %s request failed: %v\n", req.Method, req.URL, err)
		}
		var code int // HTTP response code
		if resp != nil {
			code = resp.StatusCode
		}

		// Check if we should continue with retries
		checkOk, checkErr := c.RetryCheck(req, resp, err)
		if !checkOk {
			if checkErr != nil {
				err = checkErr
			}
			return resp, err
		}
		waitDelay := c.RetryWaitDelay(i, resp)

		// consume any response to reuse the connection
		if err == nil && resp != nil {
			c.drainBody(resp.Body)
		}

		// Check if any retries left
		remain := c.RetryMax - i
		if remain == 0 {
			break
		}

		// Wait specified delay
		desc := fmt.Sprintf("%s %s", req.Method, req.URL)
		if code > 0 {
			desc = fmt.Sprintf("%s (status: %d)", desc, code)
		}
		c.ui.Logf("%s: retrying in %s (%d left)\n", desc, waitDelay, remain)
		time.Sleep(waitDelay)
	}

	if c.ErrorHandler != nil {
		return c.ErrorHandler(resp, err, c.RetryMax)
	}

	// By default, when max retries done, we close the response body and return an error without
	// returning the response
	if resp != nil {
		resp.Body.Close()
	}
	return nil, fmt.Errorf("%s %s giving up after %d attempts", req.Method, req.URL, c.RetryMax)
}

// We need to consume response bodies to maintain http connections, but
// limit the size we consume to respReadLimit.
var respReadLimit = int64(4096)

func (c *httpClient) drainBody(body io.ReadCloser) {
	defer body.Close()
	_, err := io.Copy(ioutil.Discard, io.LimitReader(body, respReadLimit))
	if err != nil {
		c.ui.Errorf("error draining response body: %v\n", err)
	}
}

func (c *httpClient) Post(url, contentType string, body io.ReadSeeker) (resp *http.Response, err error) {
	req, err := NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	return c.Do(req)
}

func (c *httpClient) Get(url string) (resp *http.Response, err error) {
	req, err := NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(req)
}
