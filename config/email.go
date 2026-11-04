package config

import "os"

type EmailConfig struct {
	SMTPHost    string
	SMTPPort    string
	SMTPUser    string
	SMTPPass    string
	SMTPPassKey string
	From        string
}

func GetEmailConfig() EmailConfig {

	smtpHost := os.Getenv("SMTP_HOST")           // Host SMTP de Brevo
	smtpPort := os.Getenv("SMTP_PORT")           // Puerto SMTP de Brevo
	smtpUser := os.Getenv("SMTP_USER")           // Email de login
	smtpPass := os.Getenv("API_KEY_SMTP")        // SMTP key de dashboard
	smtpPassKey := os.Getenv("API_KEY_SMTP_KEY") // SMTP key de dashboard
	from := "moisessantoshdz45@gmail.com"

	return EmailConfig{
		SMTPHost:    smtpHost,
		SMTPPort:    smtpPort,
		SMTPUser:    smtpUser,
		SMTPPass:    smtpPass,
		SMTPPassKey: smtpPassKey,
		From:        from,
	}
}
