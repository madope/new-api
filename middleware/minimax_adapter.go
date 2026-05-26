package middleware

import (
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"

	"github.com/gin-gonic/gin"
)

func MiniMaxNativeRequestConvert() func(c *gin.Context) {
	return func(c *gin.Context) {
		common.SetContextKey(c, constant.ContextKeyMiniMaxNativeCompat, true)

		var originalReq map[string]any
		if err := common.UnmarshalBodyReusable(c, &originalReq); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"task_id": "",
				"base_resp": gin.H{
					"status_code": 2013,
					"status_msg":  "invalid params",
				},
			})
			c.Abort()
			return
		}

		unifiedReq := make(map[string]any, len(originalReq))
		metadata := make(map[string]any)

		for k, v := range originalReq {
			switch k {
			case "model", "prompt", "duration":
				unifiedReq[k] = v
			default:
				metadata[k] = v
			}
		}

		if len(metadata) > 0 {
			unifiedReq["metadata"] = metadata
		}

		jsonData, err := common.Marshal(unifiedReq)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"task_id": "",
				"base_resp": gin.H{
					"status_code": 2013,
					"status_msg":  "invalid params",
				},
			})
			c.Abort()
			return
		}

		if err := common.ReplaceRequestBody(c, jsonData); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"task_id": "",
				"base_resp": gin.H{
					"status_code": 2013,
					"status_msg":  "invalid params",
				},
			})
			c.Abort()
			return
		}

		c.Request.URL.Path = "/v1/video/generations"
		c.Next()
	}
}
