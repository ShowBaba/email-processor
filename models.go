package main

type EmailJob struct {
	ID      string `json:"id"`
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
	Retries int    `json:"retries"`
}

type EmailRequest struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

type HealthResponse struct {
	Status        string `json:"status"`
	QueueLength   int    `json:"queue_length"`
	QueueCapacity int    `json:"queue_capacity"`
	Workers       int    `json:"workers"`
}

type DLQResponse struct {
	DeadLetterQueue []EmailJob `json:"dead_letter_queue"`
	Count           int        `json:"count"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

type SuccessResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}
