package dto

import (
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
)

type MiniMaxVoiceRequest struct {
	Model string `json:"model"`
}

func (r *MiniMaxVoiceRequest) GetTokenCountMeta() *types.TokenCountMeta {
	return &types.TokenCountMeta{
		TokenType: types.TokenTypeTokenizer,
		MaxTokens: 1,
	}
}

func (r *MiniMaxVoiceRequest) IsStream(c *gin.Context) bool {
	return false
}

func (r *MiniMaxVoiceRequest) SetModelName(modelName string) {
	if modelName != "" {
		r.Model = modelName
	}
}
