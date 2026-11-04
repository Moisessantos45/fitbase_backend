package templates

import (
	"bytes"

	"embed"
	"html/template"
)

//go:embed *.html
var templateFiles embed.FS

const (
	CONFIRM_ACCOUNT_PATH = "confirm-account"
	FORGOT_PASSWORD_PATH = "forgot-password"
)

// EmailRenderer es el encargado de cargar y renderizar las plantillas HTML
type EmailRenderer struct {
	templates *template.Template
}

// NewEmailRenderer inicializa el renderizador leyendo los archivos incrustados
func NewEmailRenderer() (*EmailRenderer, error) {
	tmpl, err := template.ParseFS(templateFiles, "*.html")
	if err != nil {
		return nil, err
	}
	return &EmailRenderer{templates: tmpl}, nil
}

// Estructuras de datos para cada tipo de correo

type PasswordResetData struct {
	Name      string
	ResetLink string
}

type ConfirmAccountData struct {
	Name        string
	ConfirmLink string
}

type WelcomeData struct {
	Name      string
	LoginLink string
}

type SubscriptionCreatedData struct {
	Name       string
	GymName    string
	PlanName   string
	PlanPrice  float64
	StartDate  string
	EndDate    string
	TicketCode string
	QRCodeURL  string
	AutoRenew  bool
	IsRenewal  bool
}

type DayPassTicketData struct {
	Name       string
	GymName    string
	PlanName   string
	PlanPrice  float64
	VisitDate  string
	TicketCode string
	QRCodeURL  string
	IsPaid     bool
}

type DayPassPaymentConfirmedData struct {
	Name              string
	GymName           string
	PlanName          string
	Amount            float64
	PaymentMethodName string
	TicketCode        string
	QRCodeURL         string
	VisitDate         string
}

func (r *EmailRenderer) RenderPasswordReset(data PasswordResetData) (string, error) {
	var body bytes.Buffer
	err := r.templates.ExecuteTemplate(&body, "password_reset.html", data)
	return body.String(), err
}

func (r *EmailRenderer) RenderConfirmAccount(data ConfirmAccountData) (string, error) {
	var body bytes.Buffer
	err := r.templates.ExecuteTemplate(&body, "confirm_account.html", data)
	return body.String(), err
}

func (r *EmailRenderer) RenderWelcome(data WelcomeData) (string, error) {
	var body bytes.Buffer
	err := r.templates.ExecuteTemplate(&body, "welcome.html", data)
	return body.String(), err
}

func (r *EmailRenderer) RenderSubscriptionCreated(data SubscriptionCreatedData) (string, error) {
	var body bytes.Buffer
	err := r.templates.ExecuteTemplate(&body, "subscription_created.html", data)
	return body.String(), err
}

func (r *EmailRenderer) RenderDayPassTicket(data DayPassTicketData) (string, error) {
	var body bytes.Buffer
	err := r.templates.ExecuteTemplate(&body, "day_pass_ticket.html", data)
	return body.String(), err
}

func (r *EmailRenderer) RenderDayPassPaymentConfirmed(data DayPassPaymentConfirmedData) (string, error) {
	var body bytes.Buffer
	err := r.templates.ExecuteTemplate(&body, "day_pass_payment_confirmed.html", data)
	return body.String(), err
}
