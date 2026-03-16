package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/zeningc/mini-kafka/broker"
)

type APIServer struct {
	b *broker.Broker
}

func NewServer(b *broker.Broker) *APIServer {
	return &APIServer{b: b}
}


func (s *APIServer) HandleCreateTopic(w http.ResponseWriter, r *http.Request) {
    topicName := r.PathValue("name")


	_, err := s.b.AddTopic(topicName)
	if err != nil	{
		w.WriteHeader(http.StatusConflict)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (s *APIServer) HandleProduce(w http.ResponseWriter, r *http.Request) {
    topicName := r.PathValue("name")

	var req ProduceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeJSONError(w, "invalid JSON payload", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	topic, err := s.b.GetTopic(topicName)
	if err != nil {
		s.writeJSONError(w, "topic not found", http.StatusNotFound)
		return
	}

	offset, err := topic.Append(req.Value)
	if err != nil {
		s.writeJSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := ProduceResponse{Offset: offset}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func (s *APIServer) HandleConsume(w http.ResponseWriter, r *http.Request)	{
    topicName := r.PathValue("name")
	t, err := s.b.GetTopic(topicName)
	if err != nil	{
		s.writeJSONError(w, err.Error(), http.StatusNotFound)
		return
	}
	startStr := r.URL.Query().Get("offset")
	start, err := strconv.ParseInt(startStr, 10, 64)
	if err != nil || start < 0 {
		s.writeJSONError(w, "invalid offset", http.StatusBadRequest)
		return
	}

	limitStr := r.URL.Query().Get("max")
	limit, err := strconv.ParseInt(limitStr, 10, 64)
	if err != nil || limit < 0 {
		s.writeJSONError(w, "invalid max", http.StatusBadRequest)
		return
	}
	messages := t.ReadFrom(start, limit)
	resp := ConsumeResponse{Messages: messages}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}



func (s *APIServer) writeJSONError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(ErrorResponse{Error: message})
}