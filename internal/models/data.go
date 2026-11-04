package models

const (
	RoleAdminID      uint64 = 1
	RoleOwnerID      uint64 = 2
	RoleGymManagerID uint64 = 3
	RoleUserID       uint64 = 4

	StatusSubscriptionActiveID   uint64 = 1
	StatusSubscriptionInactiveID uint64 = 2
	StatusSubscriptionPendingID  uint64 = 3
	StatusSubscriptionExpiredID  uint64 = 4

	AccessTypeMembershipID uint64 = 1
	AccessTypeDayPassID    uint64 = 2
	AccessTypeGuestID      uint64 = 3

	PaymentMethodCashID         uint64 = 1
	PaymentMethodCreditCardID   uint64 = 2
	PaymentMethodDebitCardID    uint64 = 3
	PaymentMethodBankTransferID uint64 = 4
	PaymentMethodPayPalID       uint64 = 5
	PaymentMethodCryptoID       uint64 = 6

	PaymentStatusPendingID  uint64 = 1
	PaymentStatusPaidID     uint64 = 2
	PaymentStatusCanceledID uint64 = 3
	PaymentStatusRefundedID uint64 = 4
)

var Roles = []Role{
	{ID: RoleAdminID, Name: "Admin"},
	{ID: RoleOwnerID, Name: "Owner"},
	{ID: RoleGymManagerID, Name: "GymManager"},
	{ID: RoleUserID, Name: "User"},
}

var PaymentMethods = []PaymentMethod{
	{ID: 1, Name: "Efectivo", Active: true},
	{ID: 2, Name: "Tarjeta de crédito", Active: true},
	{ID: 3, Name: "Tarjeta de débito", Active: true},
	{ID: 4, Name: "Transferencia bancaria", Active: true},
	{ID: 5, Name: "PayPal", Active: true},
	{ID: 6, Name: "Criptomonedas", Active: true},
}

var StatusSubscriptions = []StatusSubscription{
	{ID: 1, Name: "Activo", Description: "La suscripción está activa y en uso."},
	{ID: 2, Name: "Inactivo", Description: "La suscripción ha sido desactivada o cancelada."},
	{ID: 3, Name: "Pendiente", Description: "La suscripción está pendiente de activación o pago."},
	{ID: 4, Name: "Expirado", Description: "La suscripción ha expirado y ya no es válida."},
}

var PaymentStatuses = []PaymentStatus{
	{ID: 1, Name: "Pendiente", Active: true},
	{ID: 2, Name: "Pagado", Active: true},
	{ID: 3, Name: "Cancelado", Active: true},
	{ID: 4, Name: "Reembolsado", Active: true},
}

var AccessTypes = []AccessType{
	{ID: 1, Name: "Membresía", Active: true},
	{ID: 2, Name: "Día de Pase", Active: true},
	{ID: 3, Name: "Invitado", Active: true},
}
