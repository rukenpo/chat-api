package gcpclaude

import (
	"fmt"
	"io"
	"math/rand"
	"net/http"

	"one-api/relay/channel"
	"one-api/relay/channel/anthropic"
	"one-api/relay/channel/openai"
	"one-api/relay/model"
	"one-api/relay/util"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

var _ channel.Adaptor = new(Adaptor)

var gcpModelIDMap = map[string]string{
	"claude-3-sonnet-20240229":   "claude-3-sonnet@20240229",
	"claude-3-5-sonnet-20240620": "claude-3-5-sonnet@20240620",
	"claude-3-opus-20240229":     "claude-3-opus@20240229",
	"claude-3-haiku-20240307":    "claude-3-haiku@20240307",
}

var modelLocations = map[string][]string{
	"claude-3-sonnet-20240229":   {"asia-southeast1", "us-central1", "us-east5"},
	"claude-3-5-sonnet-20240620": {"us-east5", "europe-west1"},
	"claude-3-opus-20240229":     {"us-east5"},
	"claude-3-haiku-20240307":    {"europe-west1", "europe-west4", "us-central1", "us-east5"},
}

type Adaptor struct {
}

func (a *Adaptor) Init(meta *util.RelayMeta) {

}

func getRandomLocation(modelName string) (string, error) {
	locations, ok := modelLocations[modelName]
	if !ok {
		return "", fmt.Errorf("no locations available for model: %s", modelName)
	}

	if len(locations) == 0 {
		return "", fmt.Errorf("empty locations list for model: %s", modelName)
	}

	return locations[rand.Intn(len(locations))], nil
}

func (a *Adaptor) GetRequestURL(meta *util.RelayMeta) (string, error) {
	gcpModelID, ok := gcpModelIDMap[meta.ActualModelName]
	if !ok {
		return "", fmt.Errorf("unsupported model: %s", meta.ActualModelName)
	}

	location, err := getRandomLocation(meta.ActualModelName)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("https://%s-aiplatform.googleapis.com/v1/projects/%s/locations/%s/publishers/anthropic/models/%s:streamRawPredict",
		location,
		meta.Config.ProjectId,
		location,
		gcpModelID), nil
}

func (a *Adaptor) SetupRequestHeader(c *gin.Context, req *http.Request, meta *util.RelayMeta) error {
	channel.SetupCommonRequestHeader(c,
