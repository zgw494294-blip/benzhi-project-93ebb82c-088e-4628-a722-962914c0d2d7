package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"benzhi-project-93ebb82c-088e-4628-a722-962914c0d2d7/internal/store"
	"benzhi-project-93ebb82c-088e-4628-a722-962914c0d2d7/internal/workflow"
)

func TestIndexAndCreateAPI(t *testing.T) {
	h := New(workflow.New(store.NewMemory()))
	page := httptest.NewRecorder()
	h.ServeHTTP(page, httptest.NewRequest(http.MethodGet, "/", nil))
	if page.Code != http.StatusOK || !strings.Contains(page.Body.String(), "<body>") {
		t.Fatalf("bad page: %d %s", page.Code, page.Body.String())
	}
	body := `{"id":"p","title":"影片","language":"zh-CN","duration_ms":10000,"frame_rate":25,"participants":[{"name":"编剧","role":"WRITER"},{"name":"排演","role":"PERFORMER"},{"name":"审校","role":"REVIEWER"}],"idempotencyKey":"create"}`
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/productions", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(response, request)
	if response.Code != http.StatusCreated || !strings.Contains(response.Body.String(), `"state":"DRAFT"`) {
		t.Fatalf("bad create: %d %s", response.Code, response.Body.String())
	}
}

func TestUnknownJSONFieldReturnsFieldError(t *testing.T) {
	h := New(workflow.New(store.NewMemory()))
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/productions", strings.NewReader(`{"unknown":true}`))
	h.ServeHTTP(response, request)
	if response.Code != http.StatusUnprocessableEntity || !strings.Contains(response.Body.String(), "rule_violation") {
		t.Fatalf("bad response: %d %s", response.Code, response.Body.String())
	}
}
