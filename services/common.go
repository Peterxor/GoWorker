package services

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"math"
	"net/http"
)

func HttpRequest(method, url string, header map[string]string, data interface{}) ([]byte, error) {

	var requestBody []byte
	var err error
	var req *http.Request

	// 序列化參數
	if data != nil {
		if requestBody, err = json.Marshal(data); err != nil {
			return nil, err
		}
		if req, err = http.NewRequest(method, url, bytes.NewBuffer(requestBody)); err != nil {
			return nil, err
		}
	} else {
		if req, err = http.NewRequest(method, url, nil); err != nil {
			return nil, err
		}
	}

	// 初始化 client
	client := &http.Client{}

	// 發請求
	req.Header.Set("Content-Type", "application/json")
	if header != nil {
		for key, element := range header {
			req.Header.Set(key, element)
		}
	}
	if resp, err := client.Do(req); err != nil {
		return nil, err
	} else {

		// 讀取 body
		defer resp.Body.Close()
		if body, err := ioutil.ReadAll(resp.Body); err != nil {
			return nil, err
		} else {
			return body, nil
		}
	}
}

func Round(x float64) int {
	return int(math.Floor(x + 0.5))
}
