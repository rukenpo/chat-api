package common

import (
	"bytes"
	"io"
        "encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"one-api/relay/model"
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

func UnmarshalBodyReusable(c *gin.Context, v *GeneralOpenAIRequest) error {
	requestBody, err := GetRequestBody(c)
	if err != nil {
		return err
	}

	// Reset request body for future use
	c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))

	// Unmarshal JSON into the provided struct
	if err = json.Unmarshal(requestBody, v); err != nil {
		return errors.New("bind request body failed: " + err.Error())
	}

	// Modify fields if necessary based on model name
	if v.Model == "o1-preview" || v.Model == "o1-mini" {
		// Move max_tokens to max_completion_tokens if it exists
		if v.MaxTokens != 0 {
			v.MaxCompletionTokens = v.MaxTokens
			v.MaxTokens = 0
		}
		v.PresencePenalty = 0
		v.Temperature = 1
		v.Stream = false
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
