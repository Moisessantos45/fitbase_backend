package helper

import (
	"errors"
	"fitbase/internal/shared/errorsx"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// RespondError traduce un error de la capa de servicio/repositorio a una respuesta HTTP segura.
//
//   - errorsx.PublicError  → su status y mensaje (errores de negocio propios, se muestran tal cual).
//   - sentinelas errorsx.* → status y mensaje genérico en español.
//   - cualquier otro error → 500 genérico; el detalle real solo se escribe en el log.
func RespondError(c *gin.Context, err error) {
	if err == nil {
		return
	}

	if pub, ok := errors.AsType[*errorsx.PublicError](err); ok {
		body := gin.H{"message": pub.Message}
		if pub.Code != "" {
			body["code"] = pub.Code
		}
		if pub.Status >= http.StatusInternalServerError {
			if pub.Err != nil {
				logInternal(c, pub.Err)
			} else {
				logInternal(c, err)
			}
		}
		c.JSON(pub.Status, body)
		return
	}

	switch {
	case errors.Is(err, errorsx.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"message": "Recurso no encontrado"})
	case errors.Is(err, errorsx.ErrAlreadyExists):
		c.JSON(http.StatusConflict, gin.H{"message": "El recurso ya existe"})
	case errors.Is(err, errorsx.ErrInvalidReference):
		c.JSON(http.StatusBadRequest, gin.H{"message": "Referencia inválida"})
	case errors.Is(err, errorsx.ErrInvalidData):
		c.JSON(http.StatusBadRequest, gin.H{"message": "Datos inválidos"})
	case errors.Is(err, errorsx.ErrUnauthorized):
		c.JSON(http.StatusUnauthorized, gin.H{"message": "No autorizado"})
	case errors.Is(err, errorsx.ErrInvalidCredentials):
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Correo o contraseña incorrectos"})
	case errors.Is(err, errorsx.ErrForbidden):
		c.JSON(http.StatusForbidden, gin.H{"message": "Acceso denegado"})
	case errors.Is(err, errorsx.ErrTimeout):
		logInternal(c, err)
		c.JSON(http.StatusGatewayTimeout, gin.H{"message": "La operación tardó demasiado"})
	case errors.Is(err, errorsx.ErrCanceled):
		// el cliente cerró la conexión; no hay nada útil que responder
		c.AbortWithStatus(499)
	default:
		logInternal(c, err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Error interno del servidor"})
	}
}

func logInternal(c *gin.Context, err error) {
	log.Printf("[ERROR] %s %s: %v", c.Request.Method, c.Request.URL.Path, err)
}
