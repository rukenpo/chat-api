package common

import (
	"bytes"
	"io"

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

func UnmarshalBodyReusable(c *gin.Context, v map[string]interface{}) error {
	requestBody, err := GetRequestBody(c)
	if err != nil {
		return err
	}

	// Reset request body for future use
	c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))

	if err = c.Bind(&v); err != nil {
		return errors.Wrap(err, "bind request body failed")
	}

	// Modify the request body based on certain conditions
	if v["model"] == "o1-preview-2024-09-12" {
		if _, ok := v["max_tokens"]; ok {
			v["max_completion_tokens"] = v["max_tokens"]
			delete(v, "max_tokens")
		}
		v["presence_penalty"] = 0
		v["temperature"] = 1
	}

	// Reset the modified body back to the request
	modifiedBody, err := json.Marshal(v)
	if err != nil {
		return errors.Wrap(err, "failed to marshal modified request body")
	}

	c.Request.Body = io.NopCloser(bytes.NewBuffer(modifiedBody))
	c.Set(KeyRequestBody, modifiedBody)

	return nil
}

func SetEventStreamHeaders(c *gin.Context) {
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("Transfer-Encoding", "chunked")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
}
