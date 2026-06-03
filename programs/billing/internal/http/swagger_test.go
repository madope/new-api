package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRegisterRoutesServesSwaggerAssets(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	handler := Handler{}
	handler.RegisterRoutes(router)

	specReq := httptest.NewRequest(http.MethodGet, "/api/v1/billing/swagger/openapi.json", nil)
	specResp := httptest.NewRecorder()
	router.ServeHTTP(specResp, specReq)
	if specResp.Code != http.StatusOK {
		t.Fatalf("expected openapi route to return 200, got %d", specResp.Code)
	}
	if !strings.Contains(specResp.Body.String(), "\"openapi\"") {
		t.Fatalf("expected openapi response body, got %s", specResp.Body.String())
	}
	for _, required := range []string{
		"BillingQueryResponse",
		"BillingSummary",
		"ErrorResponse",
		"total_pages",
		"amount_divisor",
	} {
		if !strings.Contains(specResp.Body.String(), required) {
			t.Fatalf("expected openapi spec to contain %q", required)
		}
	}

	uiReq := httptest.NewRequest(http.MethodGet, "/api/v1/billing/swagger/index.html", nil)
	uiResp := httptest.NewRecorder()
	router.ServeHTTP(uiResp, uiReq)
	if uiResp.Code != http.StatusOK {
		t.Fatalf("expected swagger ui route to return 200, got %d", uiResp.Code)
	}
	if !strings.Contains(uiResp.Body.String(), "SwaggerUIBundle") {
		t.Fatalf("expected swagger ui html, got %s", uiResp.Body.String())
	}
}
