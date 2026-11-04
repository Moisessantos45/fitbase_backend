package errorsx

import (
	"errors"
	"net/http"
)

// PublicError es un error de negocio cuyo mensaje es seguro de mostrar al cliente.
// Los errores que NO sean PublicError ni uno de los sentinelas de este paquete
// se responden como 500 genérico (el detalle solo va al log).
type PublicError struct {
	Status  int    // código HTTP
	Code    string // código opcional para el frontend (p. ej. "PLAN_LIMIT_REACHED")
	Message string // mensaje seguro para el cliente
	Err     error  // causa interna opcional (solo para logs)
}

func (e *PublicError) Error() string { return e.Message }
func (e *PublicError) Unwrap() error { return e.Err }

// Public crea un error de negocio con status y mensaje visibles al cliente.
func Public(status int, message string) *PublicError {
	return &PublicError{Status: status, Message: message}
}

// PublicCode igual que Public pero con un código estable para el frontend.
func PublicCode(status int, code, message string) *PublicError {
	return &PublicError{Status: status, Code: code, Message: message}
}

// Atajos comunes.
func BadRequest(message string) *PublicError   { return Public(http.StatusBadRequest, message) }
func Conflict(message string) *PublicError     { return Public(http.StatusConflict, message) }
func NotFound(message string) *PublicError     { return Public(http.StatusNotFound, message) }
func Unauthorized(message string) *PublicError { return Public(http.StatusUnauthorized, message) }
func Forbidden(message string) *PublicError    { return Public(http.StatusForbidden, message) }

// Wrap crea un error público conservando la causa interna (solo se escribe en el log).
func Wrap(status int, message string, cause error) *PublicError {
	return &PublicError{Status: status, Message: message, Err: cause}
}

// Internal es un fallo del servidor (500) con mensaje propio visible y causa solo para logs.
func Internal(message string, cause error) *PublicError {
	return Wrap(http.StatusInternalServerError, message, cause)
}

var (
	ErrCodeAccountInactive  = errors.New("ACCOUNT_INACTIVE")
	ErrCodeEmailNotVerified = errors.New("EMAIL_NOT_VERIFIED")
	ErrEmailNotVerified     = errors.New("email no verificado")
	ErrPending2FA           = errors.New("PENDING_2FA")
	ErrNotFound             = errors.New("resource not found")
	ErrAlreadyExists        = errors.New("resource already exists")
	ErrInvalidReference     = errors.New("invalid reference")
	ErrInvalidData          = errors.New("invalid data")
	ErrTimeout              = errors.New("operation timed out")
	ErrCanceled             = errors.New("operation canceled")
	ErrUnauthorized         = errors.New("unauthorized")
	ErrInvalidCredentials   = errors.New("invalid credentials")
	ErrForbidden            = errors.New("forbidden")
)
