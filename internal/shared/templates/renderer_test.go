package templates

import (
	"strings"
	"testing"
)

func TestRenderer(t *testing.T) {
	r, err := NewEmailRenderer()
	if err != nil {
		t.Fatalf("failed to create renderer: %v", err)
	}

	// 1. Confirm Account
	confirmHtml, err := r.RenderConfirmAccount(ConfirmAccountData{
		Name:        "Test User",
		ConfirmLink: "https://fitbase.test/confirm",
	})
	if err != nil {
		t.Errorf("RenderConfirmAccount error: %v", err)
	}
	if !strings.Contains(confirmHtml, "https://fitbase.test/confirm") || !strings.Contains(confirmHtml, "FITBASE") {
		t.Errorf("ConfirmAccount missing content")
	}

	// 2. Subscription Created
	subHtml, err := r.RenderSubscriptionCreated(SubscriptionCreatedData{
		Name:       "Test User",
		GymName:    "FitNes Planet",
		PlanName:   "Plan VIP",
		StartDate:  "2026-11-01",
		EndDate:    "2026-12-01",
		TicketCode: "SUB-12345",
		AutoRenew:  true,
	})
	if err != nil {
		t.Errorf("RenderSubscriptionCreated error: %v", err)
	}
	if !strings.Contains(subHtml, "SUB-12345") || !strings.Contains(subHtml, "Activada") {
		t.Errorf("SubscriptionCreated missing content")
	}

	// 3. Day Pass Ticket (Pending)
	dpPending, err := r.RenderDayPassTicket(DayPassTicketData{
		Name:       "Guest User",
		GymName:    "FitNes Planet",
		PlanName:   "Pase 1 Dia",
		PlanPrice:  150.00,
		VisitDate:  "2026-11-02",
		TicketCode: "DP-55555",
		IsPaid:     false,
	})
	if err != nil {
		t.Errorf("RenderDayPassTicket error: %v", err)
	}
	if !strings.Contains(dpPending, "DP-55555") || !strings.Contains(dpPending, "Pendiente de Pago") {
		t.Errorf("DayPassTicket pending missing content")
	}

	// 4. Day Pass Payment Confirmed
	dpConfirmed, err := r.RenderDayPassPaymentConfirmed(DayPassPaymentConfirmedData{
		Name:              "Guest User",
		GymName:           "FitNes Planet",
		PlanName:          "Pase 1 Dia",
		Amount:            150.00,
		PaymentMethodName: "Efectivo",
		TicketCode:        "DP-55555",
		VisitDate:         "2026-11-02",
	})
	if err != nil {
		t.Errorf("RenderDayPassPaymentConfirmed error: %v", err)
	}
	if !strings.Contains(dpConfirmed, "DP-55555") || !strings.Contains(dpConfirmed, "Efectivo") {
		t.Errorf("DayPassPaymentConfirmed missing content")
	}
}

