package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

const idempotencyTTL = 5 * time.Minute

type cachedResponse struct {
	StatusCode int             `json:"status_code"`
	Body       json.RawMessage `json:"body"`
}

// bodyWriter wraps gin.ResponseWriter to capture the response body
type bodyWriter struct {
	gin.ResponseWriter
	body   *bytes.Buffer
	status int
	wrote  bool
}

func (w *bodyWriter) Write(b []byte) (int, error) {
	if !w.wrote {
		w.wrote = true
		if w.status == 0 {
			w.status = http.StatusOK
		}
	}
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w *bodyWriter) WriteHeader(code int) {
	if !w.wrote {
		w.wrote = true
	}
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *bodyWriter) WriteHeaderNow() {
	if !w.wrote {
		w.wrote = true
		w.status = w.ResponseWriter.Status()
	}
	w.ResponseWriter.WriteHeaderNow()
}

// WriteString writes a string to the response (used by c.String internally)
func (w *bodyWriter) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}

func IdempotencyMiddleware(rd *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Solo aplica a métodos que crean/modifican recursos
		if c.Request.Method != http.MethodPost &&
			c.Request.Method != http.MethodPut &&
			c.Request.Method != http.MethodPatch {
			c.Next()
			return
		}

		key := c.GetHeader("X-Idempotency-Key")
		if key == "" {
			c.Next()
			return
		}

		ctx := context.Background()
		redisKey := fmt.Sprintf("idemp:%s", key)

		// Verificar si ya existe
		val, err := rd.Get(ctx, redisKey).Result()
		if err == nil {
			// Key existe — verificar si tiene contenido real
			var cached cachedResponse
			if json.Unmarshal([]byte(val), &cached) == nil && cached.StatusCode > 0 {
				// Ya procesado, retornar respuesta cacheada directo al cliente
				c.Header("Content-Type", "application/json")
				c.Header("X-Idempotency-Replayed", "true")
				c.Writer.WriteHeader(cached.StatusCode)
				c.Writer.Write(cached.Body)
				c.Abort()
				return
			}
			// Key existe pero está "processing" — otro request en curso
			c.AbortWithStatusJSON(http.StatusConflict, gin.H{"message": "Request ya en proceso"})
			return
		}

		// Marcar como "processing" con TTL corto por si el handler falla
		rd.Set(ctx, redisKey, `{"status_code":0}`, 30*time.Second)

		// Wrap response writer para capturar la respuesta
		bw := &bodyWriter{
			ResponseWriter: c.Writer,
			body:           bytes.NewBufferString(""),
			status:         0,
		}
		c.Writer = bw

		c.Next()

		// Después del handler: cachear o limpiar
		finalStatus := bw.status
		if finalStatus == 0 {
			finalStatus = http.StatusOK
		}

		if finalStatus >= 200 && finalStatus < 400 {
			// Éxito — cachear la respuesta
			cached := cachedResponse{
				StatusCode: finalStatus,
				Body:       json.RawMessage(bw.body.Bytes()),
			}
			data, _ := json.Marshal(cached)
			rd.Set(ctx, redisKey, string(data), idempotencyTTL)
		} else {
			// Error del handler — eliminar key para permitir reintento
			rd.Del(ctx, redisKey)
		}
	}
}
