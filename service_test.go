package main

import (
	"fmt"
	"testing"
	"time"
)

type NoOpSender struct{}

func (NoOpSender) Send(job EmailJob) error { return nil }

type FailingSender struct{ Calls int }

func (f *FailingSender) Send(job EmailJob) error {
	f.Calls++
	return fmt.Errorf("forced failure")
}

func newTestService(qSize, numWorkers, maxRetries int, sender EmailSender) *EmailService {
	cfg := Config{QueueSize: qSize, NumWorkers: numWorkers, MaxRetries: maxRetries}
	queue := NewQueue(qSize)
	return NewEmailService(cfg, sender, queue)
}

func TestEnqueueJobSuccess(t *testing.T) {
	s := newTestService(2, 1, 0, NoOpSender{})
	if err := s.EnqueueJob(EmailRequest{"u@x.y", "hi", "body"}); err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}
	if got := s.GetQueueLength(); got != 1 {
		t.Errorf("want queue length 1, got %d", got)
	}
}

func TestEnqueueJobFull(t *testing.T) {
	s := newTestService(1, 1, 0, NoOpSender{})
	if err := s.EnqueueJob(EmailRequest{"a@b.c", "", ""}); err != nil {
		t.Fatalf("first enqueue: %v", err)
	}
	if err := s.EnqueueJob(EmailRequest{"d@e.f", "", ""}); err == nil {
		t.Fatal("expected queue-full error, got nil")
	}
	if got := s.GetQueueLength(); got != 1 {
		t.Errorf("queue len should still be 1, got %d", got)
	}
}

func TestGetDLQAndAddToDLQ(t *testing.T) {
	s := newTestService(1, 1, 0, NoOpSender{})
	if dlq := s.GetDLQ(); len(dlq) != 0 {
		t.Errorf("expected empty DLQ, got %v", dlq)
	}
	job := EmailJob{ID: "x", To: "u@v.w", Subject: "S", Body: "B", Retries: 5}
	s.addToDLQ(job)
	dlq := s.GetDLQ()
	if len(dlq) != 1 || dlq[0].ID != "x" {
		t.Errorf("expected DLQ[%v], got %v", job, dlq)
	}
}

func TestProcessJobSuccess(t *testing.T) {
	s := newTestService(1, 1, 0, NoOpSender{})
	job := EmailJob{ID: "j1", To: "a@b.c", Subject: "s", Body: "b", Retries: 0}
	s.processJob(0, job)
	if dlq := s.GetDLQ(); len(dlq) != 0 {
		t.Errorf("expected no DLQ entries, got %v", dlq)
	}
}

func TestStartStopLifecycle(t *testing.T) {
	s := newTestService(5, 2, 0, NoOpSender{})
	if err := s.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	done := make(chan struct{})
	go func() {
		s.Stop()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Stop() timed out")
	}
	if got := s.GetQueueLength(); got != 0 {
		t.Errorf("after stop, want queue=0, got %d", got)
	}
}

func TestRetryFailedJobsUpToMaxRetries(t *testing.T) {
	maxR := 3
	fail := &FailingSender{}
	s := newTestService(1, 1, maxR, fail)

	if err := s.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer s.Stop()

	if err := s.EnqueueJob(EmailRequest{"x@y.z", "retry", "body"}); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	time.Sleep(7 * time.Second)

	dlq := s.GetDLQ()
	if len(dlq) != 1 {
		t.Fatalf("want DLQ=1, got %d", len(dlq))
	}
	wantCalls := maxR + 1
	if fail.Calls != wantCalls {
		t.Errorf("want %d send attempts, got %d", wantCalls, fail.Calls)
	}
}
