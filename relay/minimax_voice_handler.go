package relay

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

func MiniMaxVoiceHelper(c *gin.Context, info *relaycommon.RelayInfo) (newAPIError *types.NewAPIError) {
	info.InitChannelMeta(c)

	adaptor := GetAdaptor(info.ApiType)
	if adaptor == nil {
		return types.NewError(fmt.Errorf("invalid api type: %d", info.ApiType), types.ErrorCodeInvalidApiType, types.ErrOptionWithSkipRetry())
	}
	adaptor.Init(info)

	storage, err := common.GetBodyStorage(c)
	if err != nil {
		return types.NewErrorWithStatusCode(err, types.ErrorCodeReadRequestBodyFailed, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
	}

	rawBody, err := io.ReadAll(storage)
	if err != nil {
		return types.NewErrorWithStatusCode(err, types.ErrorCodeReadRequestBodyFailed, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
	}

	logger.LogInfo(c, fmt.Sprintf("MiniMax native voice request body: %s", string(rawBody)))

	resp, err := adaptor.DoRequest(c, info, bytes.NewBuffer(rawBody))
	if err != nil {
		return types.NewOpenAIError(err, types.ErrorCodeDoRequestFailed, http.StatusInternalServerError)
	}

	httpResp := resp.(*http.Response)

	responseBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return types.NewOpenAIError(err, types.ErrorCodeReadResponseBodyFailed, http.StatusInternalServerError)
	}

	logger.LogInfo(c, fmt.Sprintf("MiniMax native voice response body: %s", string(responseBody)))

	responseBody = bytes.ReplaceAll(responseBody, []byte("\\u0026"), []byte("&"))

	service.CloseResponseBodyGracefully(httpResp)

	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.WriteHeader(httpResp.StatusCode)
	if _, err := c.Writer.Write(responseBody); err != nil {
		return types.NewError(err, types.ErrorCodeBadResponseBody)
	}

	if httpResp.StatusCode != http.StatusOK {
		if info.Billing != nil {
			info.Billing.Refund(c)
		}
		return nil
	}

	usage := &dto.Usage{
		PromptTokens:     1,
		CompletionTokens: 0,
		TotalTokens:      1,
	}

	logContent := []string{"MiniMax voice generation (native)"}

	service.PostTextConsumeQuota(c, info, usage, logContent)
	return nil
}
