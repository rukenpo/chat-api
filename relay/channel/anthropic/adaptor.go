package anthropic

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"one-api/relay/channel"
	"one-api/relay/channel/openai"
	"one-api/relay/model"
	"one-api/relay/util"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

type Adaptor struct {
}

func (a *Adaptor) Init(meta *util.RelayMeta) {

}

func (a *Adaptor) GetRequestURL(meta *util.RelayMeta) (string, error) {
	return fmt.Sprintf("%s/v1/messages", meta.BaseURL), nil
}

func (a *Adaptor) SetupRequestHeader(c *gin.Context, req *http.Request, meta *util.RelayMeta) error {
	channel.SetupCommonRequestHeader(c, req, meta)
	req.Header.Set("x-api-key", meta.APIKey)
	anthropicVersion := c.Request.Header.Get("anthropic-version")
	if anthropicVersion == "" {
		anthropicVersion = "2023-06-01"
	}
	req.Header.Set("anthropic-version", anthropicVersion)
	req.Header.Set("anthropic-beta", "messages-2023-12-15")
	return nil
}
func (a *Adaptor) ConvertImageRequest(request *model.ImageRequest) (any, error) {
	if request == nil {
		return nil, errors.New("request is nil")
	}
	return request, nil
}
func (a *Adaptor) ConvertRequest(c *gin.Context, relayMode int, request *model.GeneralOpenAIRequest) (any, error) {
	if request == nil {
		return nil, errors.New("request is nil")
	}
	return ConvertRequest(*request), nil
}

func (a *Adaptor) DoRequest(c *gin.Context, meta *util.RelayMeta, requestBody io.Reader) (*http.Response, error) {
	Model, _ := c.Get("model")
	if Model == "claude-3-opus" {
		// 序列化原始请求体
		var buf bytes.Buffer
		_, err := buf.ReadFrom(c.Request.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read request body: %v", err)
		}
		originalRequestBody := buf.Bytes()

		// 反序列化到TextRequest结构体
		var textRequest model.GeneralOpenAIRequest
		if err := json.Unmarshal(originalRequestBody, &textRequest); err != nil {
			return nil, fmt.Errorf("failed to unmarshal request body: %v", err)
		}

		updatedTextRequest, err := updateTextRequestForVision(&textRequest)
		if err != nil {
			return nil, fmt.Errorf("failed to update text request for vision: %v", err)
		}
		textRequest = *updatedTextRequest
		// 将更新后的TextRequest重新序列化
		updatedRequestBody, err := json.Marshal(textRequest)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal updated request body: %v", err)
		}

		// 使用新的请求体创建reader
		requestBodyReader := bytes.NewReader(updatedRequestBody)

		// 调用DoRequestHelper函数，传入新的请求体
		return channel.DoRequestHelper(a, c, meta, requestBodyReader)
	} else {
		return channel.DoRequestHelper(a, c, meta, requestBody)
	}
}

func (a *Adaptor) DoResponse(c *gin.Context, resp *http.Response, meta *util.RelayMeta) (aitext string, usage *model.Usage, err *model.ErrorWithStatusCode) {
	if !meta.IsClaude {
		if meta.IsStream {
			var responseText string
			err, usage, responseText = StreamHandler(c, resp)

			aitext = responseText
		} else {
			err, usage, aitext = Handler(c, resp, meta.PromptTokens, meta.ActualModelName)
		}
	} else {
		if meta.IsStream {
			var responseText string
			err, usage, responseText = ClaudeStreamHandler(c, resp)
			aitext = responseText
		} else {
			err, usage, aitext = ClaudeHandler(c, resp, meta.PromptTokens, meta.ActualModelName)
		}
	}
	return
}

func (a *Adaptor) GetModelList() []string {
	return ModelList
}

func (a *Adaptor) GetChannelName() string {
	return "authropic"
}
func updateTextRequestForVision(textRequest *model.GeneralOpenAIRequest) (*model.GeneralOpenAIRequest, error) {
	textRequest.Model = "claude-3-opus-20240229"
	for i, msg := range textRequest.Messages {
		// 假设msg.Content就是string，或者你需要根据Content的实际结构来调整
		contentStr := msg.Content.(string)
		// 正则查找URL并构建新的消息内容
		newContent, err := createNewContentWithImages(contentStr)
		if err != nil {
			log.Printf("Create new content error: %v\n", err)
			continue
		}
		newContentBytes, err := json.Marshal(newContent)
		if err != nil {
			return nil, fmt.Errorf("cannot marshal new content: %v", err)
		}
		textRequest.Messages[i].Content = json.RawMessage(newContentBytes)
	}
	return textRequest, nil
}

// 正则查找URL并构建新的消息内容
func createNewContentWithImages(contentStr string) ([]interface{}, error) {
	re := regexp.MustCompile(`http[s]?:\/\/[^\s]+`)
	matches := re.FindAllString(contentStr, -1)
	description := re.ReplaceAllString(contentStr, "")

	newContent := []interface{}{
		openai.OpenAIMessageContent{Type: "text", Text: strings.TrimSpace(description)},
	}
	// 如果没有找到匹配的URL，直接返回已有结果和nil错误
	if len(matches) == 0 {
		return newContent, nil
	}

	for _, url := range matches {
		newContent = append(newContent, openai.MediaMessageImage{
			Type:     "image_url",
			ImageUrl: openai.MessageImageUrl{Url: url},
		})
	}
	return newContent, nil
}
