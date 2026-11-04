package middleware

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GuardConfig define los límites del middleware RequestGuard.
type GuardConfig struct {
	MaxJSONBytes      int64 // tamaño máximo de un body JSON
	MaxMultipartBytes int64 // tamaño máximo total de un body multipart
	MaxJSONDepth      int   // profundidad máxima de anidamiento JSON
}

// DefaultGuardConfig: JSON 1MB, multipart 10MB, profundidad 20.
var DefaultGuardConfig = GuardConfig{
	MaxJSONBytes:      1 << 20,
	MaxMultipartBytes: 10 << 20,
	MaxJSONDepth:      20,
}

func hasBody(r *http.Request) bool {
	switch r.Method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return r.ContentLength != 0 // -1 = chunked/desconocido
	}
	return false
}

// RequestGuard solo permite application/json y multipart/form-data,
// limita el tamaño del body y valida que el JSON esté bien formado.
func RequestGuard(cfg GuardConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		r := c.Request
		if !hasBody(r) {
			c.Next()
			return
		}

		mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnsupportedMediaType, gin.H{"error": "Content-Type inválido o ausente"})
			return
		}

		switch mt {
		case "application/json":
			if r.ContentLength > cfg.MaxJSONBytes {
				c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{"error": "body demasiado grande"})
				return
			}
			body, err := io.ReadAll(http.MaxBytesReader(c.Writer, r.Body, cfg.MaxJSONBytes))
			if err != nil {
				var mbe *http.MaxBytesError
				if errors.As(err, &mbe) {
					c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{"error": "body demasiado grande"})
				} else {
					c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "no se pudo leer el body"})
				}
				return
			}
			if len(bytes.TrimSpace(body)) == 0 {
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "body JSON vacío"})
				return
			}
			if !json.Valid(body) {
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "JSON mal formado"})
				return
			}
			if !jsonDepthOK(body, cfg.MaxJSONDepth) {
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "JSON demasiado anidado"})
				return
			}
			// restaurar el body para los handlers
			r.Body = io.NopCloser(bytes.NewReader(body))
			r.ContentLength = int64(len(body))

		case "multipart/form-data":
			if r.ContentLength > cfg.MaxMultipartBytes {
				c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{"error": "body demasiado grande"})
				return
			}
			// cubre también chunked (ContentLength = -1)
			r.Body = http.MaxBytesReader(c.Writer, r.Body, cfg.MaxMultipartBytes)

		default:
			c.AbortWithStatusJSON(http.StatusUnsupportedMediaType, gin.H{
				"error": "Content-Type no soportado, usa application/json o multipart/form-data",
			})
			return
		}

		c.Next()
	}
}

// jsonDepthOK verifica que el anidamiento no exceda max.
func jsonDepthOK(b []byte, max int) bool {
	dec := json.NewDecoder(bytes.NewReader(b))
	depth := 0
	for {
		tok, err := dec.Token()
		if err != nil {
			return err == io.EOF
		}
		if d, ok := tok.(json.Delim); ok {
			if d == '{' || d == '[' {
				depth++
				if depth > max {
					return false
				}
			} else {
				depth--
			}
		}
	}
}
