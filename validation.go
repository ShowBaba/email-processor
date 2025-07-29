package main

import (
	"fmt"
	"regexp"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

func ValidateEmailRequest(req EmailRequest) error {
	if req.To == "" {
		return fmt.Errorf("'to' field is required")
	}

	if req.Subject == "" {
		return fmt.Errorf("'subject' field is required")
	}

	if req.Body == "" {
		return fmt.Errorf("'body' field is required")
	}

	if !emailRegex.MatchString(req.To) {
		return fmt.Errorf("'to' field must be a valid email address")
	}

	return nil
}
