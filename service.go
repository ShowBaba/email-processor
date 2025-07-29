package main

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

type EmailSender interface {
	Send(job EmailJob) error
}

type DefaultEmailSender struct{}

func (d DefaultEmailSender) Send(job EmailJob) error {
	time.Sleep(1 * time.Second)
	log.Printf("email sent to %s — Subject: %s", job.To, job.Subject)
	return nil
}

type JobQueue interface {
	Publish(job EmailJob) error
	Consume() (<-chan EmailJob, error)
	Close() error
	Length() int
}

type Queue struct {
	ch chan EmailJob
}

func NewQueue(size int) *Queue {
	return &Queue{ch: make(chan EmailJob, size)}
}

func (q *Queue) Publish(job EmailJob) error {
	select {
	case q.ch <- job:
		return nil
	default:
		return errors.New("queue is full")
	}
}

func (q *Queue) Consume() (<-chan EmailJob, error) {
	return q.ch, nil
}

func (q *Queue) Close() error {
	close(q.ch)
	return nil
}

func (q *Queue) Length() int {
	return len(q.ch)
}

type EmailService struct {
	config     Config
	queue      JobQueue
	dlq        []EmailJob
	dlqMutex   sync.RWMutex
	workers    sync.WaitGroup
	shutdown   chan struct{}
	jobCounter int64

	sender EmailSender
}

func NewEmailService(cfg Config, sender EmailSender, queue JobQueue) *EmailService {
	return &EmailService{
		config:   cfg,
		queue:    queue,
		dlq:      make([]EmailJob, 0),
		shutdown: make(chan struct{}),
		sender:   sender,
	}
}

func (s *EmailService) Start() error {
	jobs, err := s.queue.Consume()
	if err != nil {
		return fmt.Errorf("failed to start queue consumer: %w", err)
	}

	log.Printf("starting email service with %d workers", s.config.NumWorkers)
	for i := 0; i < s.config.NumWorkers; i++ {
		s.workers.Add(1)
		go s.worker(i, jobs)
	}
	return nil
}

func (s *EmailService) Stop() {
	log.Println("shutting down email service...")
	close(s.shutdown)
	s.queue.Close()
	s.workers.Wait()
	log.Println("email service stopped")
}

func (s *EmailService) EnqueueJob(req EmailRequest) error {
	jobID := fmt.Sprintf("job-%d", atomic.AddInt64(&s.jobCounter, 1))
	job := EmailJob{ID: jobID, To: req.To, Subject: req.Subject, Body: req.Body, Retries: 0}
	return s.queue.Publish(job)
}

func (s *EmailService) GetDLQ() []EmailJob {
	s.dlqMutex.RLock()
	defer s.dlqMutex.RUnlock()

	copyDLQ := make([]EmailJob, len(s.dlq))
	copy(copyDLQ, s.dlq)
	return copyDLQ
}

func (s *EmailService) GetQueueLength() int {
	return s.queue.Length()
}

func (s *EmailService) worker(id int, jobs <-chan EmailJob) {
	defer s.workers.Done()
	log.Printf("worker %d started", id)

	for {
		select {
		case job, ok := <-jobs:
			if !ok {
				log.Printf("worker %d stopping — queue closed", id)
				return
			}
			s.processJob(id, job)

		case <-s.shutdown:
			log.Printf("worker %d stopping — shutdown signal", id)
			return
		}
	}
}

func (s *EmailService) processJob(workerID int, job EmailJob) {
	attempt := job.Retries + 1
	log.Printf("worker %d processing %s (attempt %d)", workerID, job.ID, attempt)

	if err := s.sender.Send(job); err != nil {
		log.Printf("worker %d: %s failed: %v", workerID, job.ID, err)

		if job.Retries < s.config.MaxRetries {
			job.Retries++
			backoff := time.Duration(job.Retries) * time.Second
			time.Sleep(backoff)

			if err := s.queue.Publish(job); err != nil {
				log.Printf("failed to requeue %s — %v, moving to DLQ", job.ID, err)
				s.addToDLQ(job)
			} else {
				log.Printf("%s requeued for retry %d", job.ID, job.Retries)
			}
		} else {
			log.Printf("job %s exceeded max retries, moving to DLQ", job.ID)
			s.addToDLQ(job)
		}
	} else {
		log.Printf("worker %d: job %s completed successfully", workerID, job.ID)
	}
}

func (s *EmailService) addToDLQ(job EmailJob) {
	s.dlqMutex.Lock()
	defer s.dlqMutex.Unlock()
	s.dlq = append(s.dlq, job)
}
