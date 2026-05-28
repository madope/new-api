package dto

import (
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
)

type MiniMaxLyricsRequest struct {
	Model string `json:"model"`
}

func (r *MiniMaxLyricsRequest) GetTokenCountMeta() *types.TokenCountMeta {
	return &types.TokenCountMeta{
		TokenType: types.TokenTypeTokenizer,
		MaxTokens: 1,
	}
}

func (r *MiniMaxLyricsRequest) IsStream(c *gin.Context) bool {
	return false
}

func (r *MiniMaxLyricsRequest) SetModelName(modelName string) {
	if modelName != "" {
		r.Model = modelName
	}
}
