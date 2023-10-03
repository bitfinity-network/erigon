package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type jsonResponse struct {
	Jsonrpc string
	Id      int
	Result  interface{}
	Error   interface{}
}

func makeJsonRpcRequest(name string, params []string) map[string]interface{} {
	return map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  name,
		"params":  params,
		"id":      1, // we don't send parallel requests, os it's ok to hardcode id
	}
}

func makeRpcRequest(client *http.Client, url string, args map[string]interface{}) (interface{}, error) {
	requestBody, err := json.Marshal(args)
	if err != nil {
		return "", err
	}

	resp, err := client.Post(url, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)
	var response jsonResponse
	if err = decoder.Decode(&response); err != nil {
		return "", err
	}

	if response.Error != nil {
		return "", fmt.Errorf("%v", response.Error)
	}

	return response.Result, nil
}
