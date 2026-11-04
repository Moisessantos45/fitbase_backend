package utils

import (
	"context"
	"log"
	"sync"
	"time"
)

type EmailJob struct {
	To      []string
	Subject string
	Body    string
	ReplyTo string
}

type Mailer struct {
	jobs    chan EmailJob
	workers int
	wg      sync.WaitGroup
}

var DefaultMailer *Mailer

func InitMailer(workers int) {
	DefaultMailer = &Mailer{
		jobs:    make(chan EmailJob, 100),
		workers: workers,
	}

	for i := range workers {
		DefaultMailer.wg.Add(1)
		go DefaultMailer.worker(i + 1)
	}
}

func (m *Mailer) worker(id int) {
	defer m.wg.Done()
	for job := range m.jobs {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)

		err := SendEmailSyncWithReplyTo(ctx, job.To, job.Subject, job.Body, job.ReplyTo)
		if err != nil {
			log.Printf("Worker %d: Error enviando correo a %v: %v", id, job.To, err)
		} else {
			log.Printf("Worker %d: Correo enviado a %v exitosamente", id, job.To)
		}

		cancel()
	}
}

func ShutdownMailer(timeout time.Duration) {
	if DefaultMailer != nil {
		close(DefaultMailer.jobs)

		c := make(chan struct{})
		go func() {
			DefaultMailer.wg.Wait()
			close(c)
		}()

		select {
		case <-c:
			log.Println("Mailer shutdown completado")
		case <-time.After(timeout):
			log.Println("Mailer shutdown timeout: algunos correos pendientes pueden no haberse enviado")
		}
	}
}

func EnqueueEmail(to []string, subject string, htmlBody string) error {
	return EnqueueEmailWithReplyTo(to, subject, htmlBody, "")
}

func EnqueueEmailWithReplyTo(to []string, subject string, htmlBody string, replyTo string) error {
	job := EmailJob{
		To:      to,
		Subject: subject,
		Body:    htmlBody,
		ReplyTo: replyTo,
	}

	if DefaultMailer == nil {
		log.Println("Advertencia: DefaultMailer no inicializado. Enviando síncronamente...")
		return SendEmailSyncWithReplyTo(context.Background(), to, subject, htmlBody, replyTo)
	}

	DefaultMailer.jobs <- job
	return nil
}
