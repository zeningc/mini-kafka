package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zeningc/mini-kafka/broker"
)

func newTestServer() (*APIServer, *http.ServeMux) {
	b := broker.NewBroker()
	s := NewServer(b)
	mux := http.NewServeMux()
	mux.HandleFunc("POST /topics/{name}", s.HandleCreateTopic)
	mux.HandleFunc("POST /topics/{name}/messages", s.HandleProduce)
	mux.HandleFunc("GET /topics/{name}/messages", s.HandleConsume)
	return s, mux
}

func TestHandleCreateTopic_Success(t *testing.T) {
	t.Chdir(t.TempDir())
	_, mux := newTestServer()

	req := httptest.NewRequest(http.MethodPost, "/topics/orders", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", w.Code)
	}
}

func TestHandleCreateTopic_Duplicate(t *testing.T) {
	t.Chdir(t.TempDir())
	_, mux := newTestServer()

	for i, want := range []int{http.StatusCreated, http.StatusConflict} {
		req := httptest.NewRequest(http.MethodPost, "/topics/orders", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code != want {
			t.Errorf("request %d: expected %d, got %d", i+1, want, w.Code)
		}
	}
}

func TestHandleProduce_Success(t *testing.T) {
	t.Chdir(t.TempDir())
	_, mux := newTestServer()

	httptest.NewRecorder() // create topic first
	mux.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/topics/events", nil))

	for i := 0; i < 2; i++ {
		body, _ := json.Marshal(ProduceRequest{Value: fmt.Sprintf("msg-%d", i)})
		req := httptest.NewRequest(http.MethodPost, "/topics/events/messages", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d", w.Code)
		}
		var resp ProduceResponse
		json.NewDecoder(w.Body).Decode(&resp)
		if resp.Offset != int64(i) {
			t.Errorf("expected offset %d, got %d", i, resp.Offset)
		}
	}
}

func TestHandleProduce_TopicNotFound(t *testing.T) {
	t.Chdir(t.TempDir())
	_, mux := newTestServer()

	body, _ := json.Marshal(ProduceRequest{Value: "hello"})
	req := httptest.NewRequest(http.MethodPost, "/topics/nonexistent/messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestHandleProduce_MalformedJSON(t *testing.T) {
	t.Chdir(t.TempDir())
	_, mux := newTestServer()
	mux.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/topics/events", nil))

	req := httptest.NewRequest(http.MethodPost, "/topics/events/messages", bytes.NewReader([]byte("not-json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleConsume_Success(t *testing.T) {
	t.Chdir(t.TempDir())
	_, mux := newTestServer()

	mux.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/topics/sensor-data", nil))
	for _, val := range []string{"temperature: 22", "temperature: 25"} {
		body, _ := json.Marshal(ProduceRequest{Value: val})
		req := httptest.NewRequest(http.MethodPost, "/topics/sensor-data/messages", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		mux.ServeHTTP(httptest.NewRecorder(), req)
	}

	req := httptest.NewRequest(http.MethodGet, "/topics/sensor-data/messages?offset=0&max=10", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp ConsumeResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if len(resp.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(resp.Messages))
	}
	if resp.Messages[0].Value != "temperature: 22" || resp.Messages[1].Value != "temperature: 25" {
		t.Errorf("unexpected message values: %+v", resp.Messages)
	}
}

func TestHandleConsume_TopicNotFound(t *testing.T) {
	t.Chdir(t.TempDir())
	_, mux := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/topics/nonexistent/messages?offset=0&max=10", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestHandleConsume_InvalidOffset(t *testing.T) {
	t.Chdir(t.TempDir())
	_, mux := newTestServer()
	mux.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/topics/t", nil))

	cases := []string{
		"/topics/t/messages?offset=abc&max=10",
		"/topics/t/messages?offset=-1&max=10",
		"/topics/t/messages?max=10", // missing offset
	}
	for _, url := range cases {
		req := httptest.NewRequest(http.MethodGet, url, nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("%s: expected 400, got %d", url, w.Code)
		}
		var resp ErrorResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Errorf("%s: could not decode error response: %v", url, err)
		}
		if resp.Error == "" {
			t.Errorf("%s: expected non-empty error message in body", url)
		}
	}
}

func TestHandleConsume_InvalidMax(t *testing.T) {
	t.Chdir(t.TempDir())
	_, mux := newTestServer()
	mux.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/topics/t", nil))

	cases := []string{
		"/topics/t/messages?offset=0&max=abc",
		"/topics/t/messages?offset=0&max=-1",
		"/topics/t/messages?offset=0", // missing max
	}
	for _, url := range cases {
		req := httptest.NewRequest(http.MethodGet, url, nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("%s: expected 400, got %d", url, w.Code)
		}
		var resp ErrorResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Errorf("%s: could not decode error response: %v", url, err)
		}
		if resp.Error == "" {
			t.Errorf("%s: expected non-empty error message in body", url)
		}
	}
}

func TestRouting_UnknownRoute(t *testing.T) {
	t.Chdir(t.TempDir())
	_, mux := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/unknown", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}
