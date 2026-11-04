package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

const (
	// maxClientsPerStore evita que un atacante con muchas IPs (p. ej. IPv6) agote la memoria.
	maxClientsPerStore = 100_000
	// idleTTL: tiempo sin actividad tras el cual se descarta el contador de una IP.
	idleTTL = time.Hour
)

type client struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// limiterStore guarda los contadores de UNA instancia del middleware, de modo que
// cada limiter (global, login, registro...) tiene sus propios límites y no comparten estado.
type limiterStore struct {
	mu      sync.Mutex
	clients map[string]*client
	lmt     rate.Limit
	burst   int
}

var (
	storesMu    sync.Mutex
	stores      []*limiterStore
	cleanupOnce sync.Once
)

func newLimiterStore(lmt rate.Limit, burst int) *limiterStore {
	s := &limiterStore{
		clients: make(map[string]*client),
		lmt:     lmt,
		burst:   burst,
	}

	storesMu.Lock()
	stores = append(stores, s)
	storesMu.Unlock()

	return s
}

// allow consume un token de la IP indicada. Devuelve si se permite y cuánto esperar si no.
func (s *limiterStore) allow(ip string) (bool, time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	cl, ok := s.clients[ip]
	if !ok {
		if len(s.clients) >= maxClientsPerStore {
			s.sweepLocked(now, time.Minute)
		}
		cl = &client{limiter: rate.NewLimiter(s.lmt, s.burst)}
		s.clients[ip] = cl
	}
	cl.lastSeen = now

	r := cl.limiter.Reserve()
	if !r.OK() {
		return false, time.Second
	}
	if delay := r.DelayFrom(now); delay > 0 {
		r.CancelAt(now) // no consumir el token si se rechaza
		return false, delay
	}

	return true, 0
}

// sweepLocked elimina las IPs inactivas por más de ttl. Requiere s.mu tomado.
func (s *limiterStore) sweepLocked(now time.Time, ttl time.Duration) {
	for ip, cl := range s.clients {
		if now.Sub(cl.lastSeen) > ttl {
			delete(s.clients, ip)
		}
	}
}

// RateLimiterMiddleware limita las peticiones por IP. Cada llamada crea un limiter
// independiente, por lo que se puede usar uno global y otros más estrictos por ruta.
func RateLimiterMiddleware(lmt rate.Limit, burst int) gin.HandlerFunc {
	store := newLimiterStore(lmt, burst)

	return func(c *gin.Context) {
		if c.Request.Method == http.MethodOptions {
			c.Next()
			return
		}

		ip := c.ClientIP()
		if ip == "" {
			ip = "127.0.0.1"
		}

		allowed, wait := store.allow(ip)
		if !allowed {
			secs := int(wait.Seconds()) + 1
			c.Header("Retry-After", strconv.Itoa(secs))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "rate limit exceeded",
			})
			return
		}

		c.Next()
	}
}

// StartCleanup lanza (una sola vez) la limpieza periódica de todos los limiters registrados.
func StartCleanup() {
	cleanupOnce.Do(func() {
		go func() {
			ticker := time.NewTicker(10 * time.Minute)
			defer ticker.Stop()
			for range ticker.C {
				now := time.Now()

				storesMu.Lock()
				current := make([]*limiterStore, len(stores))
				copy(current, stores)
				storesMu.Unlock()

				for _, s := range current {
					s.mu.Lock()
					s.sweepLocked(now, idleTTL)
					s.mu.Unlock()
				}
			}
		}()
	})
}
