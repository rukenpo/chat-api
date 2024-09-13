package common

import (
	"bytes"
	"io"
        "encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

const KeyRequestBody = "key_request_body"

func GetRequestBody(c *gin.Context) ([]byte, error) {
	requestBody, exists := c.Get(KeyRequestBody)
	if exists {
		if body, ok := requestBody.([]byte); ok {
			return body, nil
		}
		return nil, errors.New("invalid request body type")
	}
	requestBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return nil, err
	}
	_ = c.Request.Body.Close()
	c.Set(KeyRequestBody, requestBody)
	return requestBody.([]byte), nil
}

func UnmarshalBodyReusable(c *gin.Context, v interface{}) error {
	requestBody, err := GetRequestBody(c)
	if err != nil {
		return err
	}

	// Reset request body for future use
	c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))

	// Unmarshal JSON into the provided interface
	if err = json.Unmarshal(requestBody, v); err != nil {
		return errors.Wrap(err, "bind request body failed")
	}

	// Modify fields if necessary
	if reqMap, ok := v.(map[string]interface{}); ok {
		if reqMap["model"] == "o1-preview-2024-09-12" {
			if val, exists := reqMap["max_tokens"]; exists {
				reqMap["max_completion_tokens"] = val
				delete(reqMap, "max_tokens")
			}
			reqMap["presence_penalty"] = 0
			reqMap["temperature"] = 0
		}
	}

	return nil
}

func SetEventStreamHeaders(c *gin.Context) {
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("Transfer-Encoding", "chunked")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
}
