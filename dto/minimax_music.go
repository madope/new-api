package dto

import (
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
)

type MiniMaxMusicRequest struct {
	Model string `json:"model"`
}

func (r *MiniMaxMusicRequest) GetTokenCountMeta() *types.TokenCountMeta {
	return &types.TokenCountMeta{
		TokenType: types.TokenTypeTokenizer,
		MaxTokens: 1,
	}
}

func (r *MiniMaxMusicRequest) IsStream(c *gin.Context) bool {
	return false
}

func (r *MiniMaxMusicRequest) SetModelName(modelName string) {
	if modelName != "" {
		r.Model = modelName
	}
}
