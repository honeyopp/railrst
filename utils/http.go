package utils

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"time"
)

// HttpGetJSON GET请求并解析json
func HttpGetJSON(url string, respObj interface{}) error {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, respObj)
}

// HttpPostJSON POST请求json并解析json
func HttpPostJSON(url string, data interface{}, respObj interface{}, headers map[string]string) error {
	client := &http.Client{Timeout: 10 * time.Second}

	b, err := json.Marshal(data)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(b))
	if err != nil {
		return err
	}

	// 设置 Content-Type
	req.Header.Set("Content-Type", "application/json;charset=utf-8")

	// 设置额外的 Header
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	return json.Unmarshal(body, respObj)
}
