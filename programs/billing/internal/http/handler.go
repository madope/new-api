package httpapi

import (
	_ "embed"
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/programs/billing/internal/query"
	"github.com/QuantumNous/new-api/programs/billing/internal/store"
	"github.com/gin-gonic/gin"
)

//go:embed openapi.json
var openAPISpec []byte

type Handler struct {
	Store           store.Store
	DefaultPageSize int
	MaxPageSize     int
}

type responseEnvelope struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

type pageInfo struct {
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

type responseData struct {
	Summary query.SummaryRow `json:"summary"`
	Items   []map[string]any `json:"items"`
	Page    pageInfo         `json:"page"`
	Meta    map[string]any   `json:"meta"`
}

func (h Handler) RegisterRoutes(router *gin.Engine) {
	router.GET("/api/v1/billing/healthz", h.Healthz)
	router.GET("/api/v1/billing/query", h.Query)
	router.GET("/api/v1/billing/swagger/openapi.json", h.OpenAPISpec)
	router.GET("/api/v1/billing/swagger/index.html", h.SwaggerUI)
}

func (h Handler) Healthz(c *gin.Context) {
	if err := h.Store.Ping(); err != nil {
		c.JSON(http.StatusServiceUnavailable, responseEnvelope{
			Success: false,
			Message: err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, responseEnvelope{
		Success: true,
		Message: "",
		Data: gin.H{
			"status": "ok",
		},
	})
}

func (h Handler) Query(c *gin.Context) {
	req, err := h.parseRequest(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, responseEnvelope{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	summary, err := h.Store.QuerySummary(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, responseEnvelope{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	items, err := h.itemsForRequest(req, summary)
	if err != nil {
		c.JSON(http.StatusInternalServerError, responseEnvelope{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	total, err := h.Store.CountItems(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, responseEnvelope{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	groupNames, _ := query.ParseGroupNames(req.GroupBy)
	data := responseData{
		Summary: summary,
		Items:   items,
		Page: pageInfo{
			Page:       req.Page,
			PageSize:   req.PageSize,
			Total:      total,
			TotalPages: totalPages(total, req.PageSize),
		},
		Meta: map[string]any{
			"group_by":       groupNames,
			"order_by":       req.OrderBy,
			"order":          req.Order,
			"amount_divisor": 500000,
			"filters": gin.H{
				"start_timestamp": req.StartTimestamp,
				"end_timestamp":   req.EndTimestamp,
				"user_id":         req.UserID,
				"username":        req.Username,
				"token_id":        req.TokenID,
				"token_name":      req.TokenName,
				"channel_id":      req.ChannelID,
				"channel_name":    req.ChannelName,
				"model_name":      req.ModelName,
				"group_name":      req.GroupName,
				"type":            derefInt(req.Type),
				"is_stream":       derefBool(req.IsStream),
			},
		},
	}

	c.JSON(http.StatusOK, responseEnvelope{
		Success: true,
		Message: "",
		Data:    data,
	})
}

func (h Handler) OpenAPISpec(c *gin.Context) {
	c.Data(http.StatusOK, "application/json; charset=utf-8", openAPISpec)
}

func (h Handler) SwaggerUI(c *gin.Context) {
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Billing Swagger UI</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    window.ui = SwaggerUIBundle({
      url: '/api/v1/billing/swagger/openapi.json',
      dom_id: '#swagger-ui'
    });
  </script>
</body>
</html>`))
}

func (h Handler) parseRequest(c *gin.Context) (query.Request, error) {
	req := query.Request{
		StartTimestamp: parseInt64(c.Query("start_timestamp")),
		EndTimestamp:   parseInt64(c.Query("end_timestamp")),
		UserID:         parseInt64(c.Query("user_id")),
		Username:       c.Query("username"),
		TokenID:        parseInt64(c.Query("token_id")),
		TokenName:      c.Query("token_name"),
		ChannelID:      parseInt64(c.Query("channel_id")),
		ChannelName:    c.Query("channel_name"),
		ModelName:      c.Query("model_name"),
		GroupName:      c.Query("group_name"),
		GroupBy:        c.Query("group_by"),
		Page:           parseInt(c.Query("page")),
		PageSize:       parseInt(c.Query("page_size")),
		OrderBy:        c.Query("order_by"),
		Order:          c.Query("order"),
	}
	if raw := c.Query("type"); raw != "" {
		value := parseInt(raw)
		req.Type = &value
	}
	if raw := c.Query("is_stream"); raw != "" {
		value := raw == "true" || raw == "1"
		req.IsStream = &value
	}
	return query.NormalizeRequest(req, h.DefaultPageSize, h.MaxPageSize)
}

func (h Handler) itemsForRequest(req query.Request, summary query.SummaryRow) ([]map[string]any, error) {
	groupNames, _ := query.ParseGroupNames(req.GroupBy)
	if len(groupNames) == 0 {
		return []map[string]any{
			{
				"requests":          summary.Requests,
				"quota":             summary.Quota,
				"amount":            summary.Amount,
				"prompt_tokens":     summary.PromptTokens,
				"completion_tokens": summary.CompletionTokens,
				"total_tokens":      summary.TotalTokens,
				"avg_use_time_ms":   summary.AvgUseTimeMS,
				"max_use_time_ms":   summary.MaxUseTimeMS,
			},
		}, nil
	}
	return h.Store.QueryItems(req)
}

func parseInt(raw string) int {
	value, _ := strconv.Atoi(raw)
	return value
}

func parseInt64(raw string) int64 {
	value, _ := strconv.ParseInt(raw, 10, 64)
	return value
}

func totalPages(total, pageSize int) int {
	if total == 0 {
		return 0
	}
	return (total + pageSize - 1) / pageSize
}

func derefInt(value *int) any {
	if value == nil {
		return nil
	}
	return *value
}

func derefBool(value *bool) any {
	if value == nil {
		return nil
	}
	return *value
}
