package dto

import (
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
)

type MiniMaxNativeImageRequest struct {
	Model string `json:"model"`
}

func (r *MiniMaxNativeImageRequest) GetTokenCountMeta() *types.TokenCountMeta {
	return &types.TokenCountMeta{
		TokenType: types.TokenTypeTokenizer,
		MaxTokens: 1,
	}
}

func (r *MiniMaxNativeImageRequest) IsStream(c *gin.Context) bool {
	return false
}

func (r *MiniMaxNativeImageRequest) SetModelName(modelName string) {
	if modelName != "" {
		r.Model = modelName
	}
}
