package utils

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"time"

	"github.com/o1egl/paseto"
)

type Payload struct {
	ID       string    `json:"id"`                 // jti único anti-replay
	UserID   string    `json:"sub"`                // tu user_id
	Platform string    `json:"platform,omitempty"` // "web" o "mobile"
	Exp      time.Time `json:"exp"`                // expiración
}

type PasetoMaker struct {
	v2 *paseto.V2
}

func NewPasetoMaker() *PasetoMaker {
	return &PasetoMaker{
		v2: paseto.NewV2(),
	}
}

func (maker *PasetoMaker) NewToken(userID string, duration time.Duration) (string, error) {
	return maker.NewTokenWithPlatform(userID, "", duration)
}

func (maker *PasetoMaker) NewTokenWithPlatform(userID string, platform string, duration time.Duration) (string, error) {
	jti := make([]byte, 16)
	var secretKey = []byte(os.Getenv("SECRET_KEY_JWT"))
	if _, err := rand.Read(jti); err != nil {
		return "", err
	}

	payload := Payload{
		ID:       hex.EncodeToString(jti),
		UserID:   userID,
		Platform: platform,
		Exp:      time.Now().Add(duration),
	}

	token, err := maker.v2.Encrypt(secretKey, payload, nil)
	return token, err
}

func (maker *PasetoMaker) NewInviteToken(userID string, duration time.Duration, role string, appIDs string) (string, *Payload, error) {
	jti := make([]byte, 16)
	secretKey := []byte(os.Getenv("SECRET_KEY_JWT"))

	if _, err := rand.Read(jti); err != nil {
		return "", nil, err
	}

	payload := &Payload{
		ID:     hex.EncodeToString(jti),
		UserID: userID,
		Exp:    time.Now().Add(duration),
	}

	token, err := maker.v2.Encrypt(secretKey, payload, nil)
	if err != nil {
		return "", nil, err
	}

	return token, payload, nil
}

func (maker *PasetoMaker) VerifyToken(token string) (*Payload, error) {
	var payload Payload
	var secretKey = []byte(os.Getenv("SECRET_KEY_JWT"))

	err := maker.v2.Decrypt(token, secretKey, &payload, nil)
	if err != nil {
		return nil, err
	}

	if time.Now().After(payload.Exp) {
		return nil, errors.New("token expirado")
	}

	// Opcional: validar jti vs cache Redis (anti-replay)
	// redis.Set(ctx, "used:"+payload.ID, "1", 24*time.Hour)

	return &payload, nil
}
