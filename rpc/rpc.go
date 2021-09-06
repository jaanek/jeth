package rpc

import (
	"bytes"
	"encoding/json"
	"io/ioutil"

	"github.com/jaanek/jeth/httpclient"
	"github.com/jaanek/jeth/ui"
)

type endpoint struct {
	url string
}

func (e *endpoint) Url() string {
	return e.url
}

type RpcEndpoint interface {
	Url() string
}

func NewEndpoint(url string) RpcEndpoint {
	return &endpoint{url: url}
}

type RpcRequest struct {
	Id      uint     `json:"id"`
	Version string   `json:"jsonrpc"`
	Method  string   `json:"method"`
	Params  []string `json:"params"`
}
type RpcResponse struct {
	Id      uint   `json:"id"`
	Version string `json:"jsonrpc"`
	Result  string `json:"result"`
}

func Call(ui ui.Screen, client httpclient.HttpClient, endpoint RpcEndpoint, method string, params []string) (*RpcResponse, error) {
	payload, err := json.Marshal(&RpcRequest{
		Id:      1,
		Version: "2.0",
		Method:  method,
		Params:  params,
	})
	if err != nil {
		return nil, err
	}
	res, err := client.Post(endpoint.Url(), "application/json", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	ui.Log(string(body))
	var resp = RpcResponse{}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
