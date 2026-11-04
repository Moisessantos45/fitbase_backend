package middleware

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"golang.org/x/time/rate"
)

// tokenBucketScript implementa un token bucket de forma atómica en Redis.
// Usa el reloj de Redis (TIME) para que varias instancias de la API no dependan de sus relojes.
//
//	KEYS[1] = clave del cliente
//	ARGV[1] = tokens por segundo, ARGV[2] = capacidad (burst)
//
// Devuelve {permitido (1/0), milisegundos a esperar}.
var tokenBucketScript = redis.NewScript(`
local rate  = tonumber(ARGV[1])
local burst = tonumber(ARGV[2])

local t   = redis.call('TIME')
local now = t[1] * 1000 + math.floor(t[2] / 1000)

local data   = redis.call('HMGET', KEYS[1], 'tokens', 'ts')
local tokens = tonumber(data[1])
local ts     = tonumber(data[2])

if tokens == nil then
  tokens = burst
  ts = now
end

local delta = math.max(0, now - ts)
tokens = math.min(burst, tokens + (delta * rate / 1000))

local allowed = 0
local retry = 0
if tokens >= 1 then
  tokens = tokens - 1
  allowed = 1
else
  retry = math.ceil((1 - tokens) / rate * 1000)
end

redis.call('HSET', KEYS[1], 'tokens', tokens, 'ts', now)
redis.call('PEXPIRE', KEYS[1], math.ceil(burst / rate * 1000) + 1000)

return {allowed, retry}
`)

const (
	redisRateTimeout = 300 * time.Millisecond
	// redisRetryAfterFailure: tras un fallo de Redis se usa el limiter en memoria durante este tiempo
	// para no añadir latencia a cada petición mientras Redis esté caído.
	redisRetryAfterFailure = 5 * time.Second
)

// RateLimiterRedis limita las peticiones por IP usando Redis, de modo que el límite se comparte
// entre todas las instancias de la API. `name` separa los contadores de cada limiter
// (p. ej. "global", "auth"). Si Redis falla, usa temporalmente un limiter en memoria con los mismos
// parámetros (no se deja la API sin protección ni se bloquea a los usuarios).
func RateLimiterRedis(rd *redis.Client, name string, lmt rate.Limit, burst int) gin.HandlerFunc {
	if rd == nil || lmt <= 0 || burst <= 0 {
		return RateLimiterMiddleware(lmt, burst)
	}

	fallback := newLimiterStore(lmt, burst)
	rateStr := strconv.FormatFloat(float64(lmt), 'f', -1, 64)
	burstStr := strconv.Itoa(burst)
	prefix := "ratelimit:" + name + ":"

	var downUntil atomic.Int64 // unix nano hasta el que no se intenta Redis

	return func(c *gin.Context) {
		if c.Request.Method == http.MethodOptions {
			c.Next()
			return
		}

		ip := c.ClientIP()
		if ip == "" {
			ip = "127.0.0.1"
		}

		allowed, wait := true, time.Duration(0)

		if time.Now().UnixNano() < downUntil.Load() {
			allowed, wait = fallback.allow(ip)
		} else {
			ctx, cancel := context.WithTimeout(c.Request.Context(), redisRateTimeout)
			res, err := tokenBucketScript.Run(ctx, rd, []string{prefix + ip}, rateStr, burstStr).Slice()
			cancel()

			if err != nil || len(res) != 2 {
				log.Printf("[rate-limit] Redis no disponible, usando limiter en memoria: %v", err)
				downUntil.Store(time.Now().Add(redisRetryAfterFailure).UnixNano())
				allowed, wait = fallback.allow(ip)
			} else {
				ok, _ := res[0].(int64)
				ms, _ := res[1].(int64)
				allowed = ok == 1
				wait = time.Duration(ms) * time.Millisecond
			}
		}

		if !allowed {
			c.Header("Retry-After", strconv.Itoa(int(wait.Seconds())+1))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "rate limit exceeded",
			})
			return
		}

		c.Next()
	}
}
