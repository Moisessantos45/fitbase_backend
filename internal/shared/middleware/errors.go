package middleware

import (
	"log"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
)

// SafeRecovery captura panics: el stack trace va al log y el cliente solo recibe un 500 genérico.
func SafeRecovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("[PANIC] %s %s: %v\n%s", c.Request.Method, c.Request.URL.Path, rec, debug.Stack())
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Error interno del servidor"})
			}
		}()
		c.Next()
	}
}

// NotFound responde un JSON genérico para rutas inexistentes.
func NotFound(c *gin.Context) {
	c.JSON(http.StatusNotFound, gin.H{"message": "Recurso no encontrado"})
}

// MethodNotAllowed responde un JSON genérico para métodos no permitidos.
func MethodNotAllowed(c *gin.Context) {
	c.JSON(http.StatusMethodNotAllowed, gin.H{"message": "Método no permitido"})
}

// InternalError registra el error real y responde un mensaje genérico (500).
func InternalError(c *gin.Context, err error) {
	log.Printf("[ERROR] %s %s: %v", c.Request.Method, c.Request.URL.Path, err)
	c.JSON(http.StatusInternalServerError, gin.H{"message": "Error interno del servidor"})
}

// BadRequest registra el error real (binding/validación) y responde un mensaje genérico (400),
// evitando exponer nombres de structs o campos internos.
func BadRequest(c *gin.Context, err error) {
	log.Printf("[WARN] %s %s: bad request: %v", c.Request.Method, c.Request.URL.Path, err)
	c.JSON(http.StatusBadRequest, gin.H{"message": "Solicitud inválida"})
}
