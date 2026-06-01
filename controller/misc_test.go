package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type statusResponse struct {
	Success bool                   `json:"success"`
	Data    map[string]interface{} `json:"data"`
}

func TestGetStatusIncludesRegisterAndNoticeFlags(t *testing.T) {
	gin.SetMode(gin.TestMode)

	originalRegisterEnabled := common.RegisterEnabled
	originalPasswordRegisterEnabled := common.PasswordRegisterEnabled
	originalNoticeButtonEnabled := common.NoticeButtonEnabled
	t.Cleanup(func() {
		common.RegisterEnabled = originalRegisterEnabled
		common.PasswordRegisterEnabled = originalPasswordRegisterEnabled
		common.NoticeButtonEnabled = originalNoticeButtonEnabled
	})

	common.RegisterEnabled = false
	common.PasswordRegisterEnabled = false
	common.NoticeButtonEnabled = false

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/status", nil)

	GetStatus(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)

	var payload statusResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &payload))
	require.True(t, payload.Success)
	require.Equal(t, false, payload.Data["register_enabled"])
	require.Equal(t, false, payload.Data["password_register_enabled"])
	require.Equal(t, false, payload.Data["notice_button_enabled"])
}
