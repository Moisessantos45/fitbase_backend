package utils

import (
	"context"
	"fitbase/config"

	"fmt"
	"log"
	"net/smtp"

	"github.com/resend/resend-go/v3"
)

func SendEmailSync(ctx context.Context, to []string, subject string, htmlBody string) error {
	return SendEmailSyncWithReplyTo(ctx, to, subject, htmlBody, "")
}

func SendEmailSyncWithReplyTo(ctx context.Context, to []string, subject string, htmlBody string, replyTo string) error {
	log.Println("Intentando enviar correo vía Google SMTP...")
	err := sendWithGoogle(to, subject, htmlBody, replyTo)
	if err == nil {
		return nil
	}

	log.Printf("Google SMTP falló: %v. Reintentando con Resend...", err)

	err = sendWithResend(ctx, to, subject, htmlBody, replyTo)
	if err != nil {
		log.Printf("Resend también falló: %v", err)
		return fmt.Errorf("todos los servicios de email fallaron: %w", err)
	}

	log.Println("Correo enviado exitosamente vía Resend")
	return nil
}

func sendWithGoogle(to []string, subject string, htmlBody string, replyTo string) error {
	emailConfig := config.GetEmailConfig()

	from := fmt.Sprintf("Formulario de Contacto <%s>", emailConfig.From)

	msg := fmt.Appendf(nil, "To: %s\r\n"+
		"From: %s\r\n"+
		"Subject: %s\r\n"+
		"MIME-Version: 1.0\r\n"+
		"Content-Type: text/html; charset=\"UTF-8\"\r\n",
		to[0], from, subject)

	if replyTo != "" {
		msg = fmt.Appendf(msg, "Reply-To: %s\r\n", replyTo)
	}

	msg = fmt.Appendf(msg, "\r\n%s", htmlBody)

	auth := smtp.PlainAuth("", emailConfig.SMTPUser, emailConfig.SMTPPass, emailConfig.SMTPHost)
	addr := fmt.Sprintf("%s:%s", emailConfig.SMTPHost, emailConfig.SMTPPort)

	return smtp.SendMail(addr, auth, emailConfig.From, to, []byte(msg))
}

func sendWithResend(ctx context.Context, to []string, subject string, htmlBody string, replyTo string) error {
	emailConfig := config.GetEmailConfig()
	client := resend.NewClient(emailConfig.SMTPPassKey)

	from := "Formulario de Contacto <support@mmabitec.me>"

	params := &resend.SendEmailRequest{
		From:    from,
		To:      to,
		Html:    htmlBody,
		Subject: subject,
	}

	if replyTo != "" {
		params.ReplyTo = replyTo
	}

	_, err := client.Emails.Send(params)
	if err != nil {
		fmt.Println(err.Error())
		return err
	}

	return nil
}
