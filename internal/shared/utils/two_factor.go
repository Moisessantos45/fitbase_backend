package utils

import (
	"time"

	"github.com/xlzd/gotp"
)

func GenerateTOTP(email string, name string) (string, string, error) {
	secret := gotp.RandomSecret(16)
	totp := gotp.NewDefaultTOTP(secret)

	uri := totp.ProvisioningUri(email, name)

	return uri, secret, nil
}

func VerifyTOTP(secret string, code string) bool {
	totp := gotp.NewDefaultTOTP(secret)
	return totp.Verify(code, time.Now().Unix())
}
