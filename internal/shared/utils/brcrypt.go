package utils

import (
	"fmt"
	"sync"

	"golang.org/x/crypto/bcrypt"
)

var (
	dummyHashOnce sync.Once
	dummyHash     []byte
)

// BurnPasswordCheck ejecuta una comparación bcrypt contra un hash falso para que el login
// de un usuario inexistente tarde lo mismo que el de uno existente (evita enumeración por tiempo).
func BurnPasswordCheck(password string) {
	dummyHashOnce.Do(func() {
		dummyHash, _ = bcrypt.GenerateFromPassword([]byte("fitbase-dummy-password"), bcrypt.DefaultCost)
	})
	_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
}

// HashPassword genera un hash bcrypt de una contraseña.
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("error al generar el hash de la contraseña: %w", err)
	}
	return string(bytes), nil
}

// CheckPasswordHash compara una contraseña en texto plano con su hash.
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
