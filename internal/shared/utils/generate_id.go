package utils

import (
	"fmt"
	"math/rand"
	"time"
)

func GenerateID() string {
	// Genera un ID único basado en el tiempo actual y un número aleatorio
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), rand.Intn(1000000))
}
