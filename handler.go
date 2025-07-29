package main

import (
	"encoding/json"
	"net/http"
)

func (s *EmailService) SendEmailHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req EmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, "Invalid JSON", http.StatusUnprocessableEntity)
		return
	}

	if err := ValidateEmailRequest(req); err != nil {
		writeErrorResponse(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	if err := s.EnqueueJob(req); err != nil {
		writeErrorResponse(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	writeSuccessResponse(w, "accepted", "Email job queued successfully", http.StatusAccepted)
}

func (s *EmailService) DLQHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	dlq := s.GetDLQ()
	response := DLQResponse{
		DeadLetterQueue: dlq,
		Count:           len(dlq),
	}

	writeJSONResponse(w, response, http.StatusOK)
}

func (s *EmailService) HealthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := HealthResponse{
		Status:        "healthy",
		QueueLength:   s.GetQueueLength(),
		QueueCapacity: s.config.QueueSize,
		Workers:       s.config.NumWorkers,
	}

	writeJSONResponse(w, response, http.StatusOK)
}

func writeJSONResponse(w http.ResponseWriter, data interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

func writeErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	response := ErrorResponse{
		Error:   http.StatusText(statusCode),
		Message: message,
	}
	writeJSONResponse(w, response, statusCode)
}

func writeSuccessResponse(w http.ResponseWriter, status, message string, statusCode int) {
	response := SuccessResponse{
		Status:  status,
		Message: message,
	}
	writeJSONResponse(w, response, statusCode)
}
