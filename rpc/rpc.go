package rpc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/jaanek/jeth/httpclient"
	"github.com/jaanek/jeth/ui"
)

type rpcEndpoint struct {
	url string
}

func (e *rpcEndpoint) Url() string {
	return e.url
}

type Endpoint interface {
	Url() string
}

func NewEndpoint(url string) Endpoint {
	return &rpcEndpoint{url: url}
}

type RpcRequest struct {
	Id      uint          `json:"id"`
	Version string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
}
type RpcResponse interface {
	Error() *RpcError
}
type RpcResultStr struct {
	Id      uint      `json:"id"`
	Version string    `json:"jsonrpc"`
	Result  string    `json:"result"`
	Err     *RpcError `json:"error"`
}

func (r *RpcResultStr) Error() *RpcError {
	return r.Err
}

type RpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *RpcError) Error() string {
	return fmt.Sprintf("code: %d, message: %s", e.Code, e.Message)
}

func Call(ui ui.Screen, client httpclient.HttpClient, endpoint Endpoint, method string, params []interface{}, resp RpcResponse) error {
	payload, err := json.Marshal(&RpcRequest{
		Id:      1,
		Version: "2.0",
		Method:  method,
		Params:  params,
	})
	if err != nil {
		return err
	}
	ui.Log(string(payload))
	res, err := client.Post(endpoint.Url(), "application/json", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	ui.Log(string(body))
	if err := json.Unmarshal(body, &resp); err != nil {
		return err
	}
	if resp.Error() != nil {
		return resp.Error()
	}
	return nil
}
