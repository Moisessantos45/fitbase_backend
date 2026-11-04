package utils

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
)

func GenerateURL(path string, params string) string {
	isProduction := os.Getenv("GO_ENV")
	var host = os.Getenv("HOST_API_PROD")
	if isProduction == "dev" {
		host = os.Getenv("HOST_API_DEV")
	}

	// En dev, siempre usar IP dinámica del servidor + puerto de HOST_API_DEV
	if isProduction == "dev" {
		ip := GetOutboundIP()
		u, err := url.Parse(host)
		if err == nil && u.Port() != "" {
			host = fmt.Sprintf("http://%s:%s", ip, u.Port())
		} else {
			host = fmt.Sprintf("http://%s", ip)
		}
	}

	if strings.TrimSpace(params) == "" {
		return fmt.Sprintf("%s/%s", host, path)
	}

	return fmt.Sprintf("%s/%s/%s/%s", host, "api/v1", path, params)
}

// RebuildURL reconstruye una URL guardada con el host actual del servidor
// Ejemplo: input "http://172.16.0.2:4105/api/v1/storage/file/123" → "http://192.168.1.50:4105/api/v1/storage/file/123"
func RebuildURL(storedURL string) string {
	if storedURL == "" {
		return ""
	}

	// Si no es URL absoluta, devolver tal cual
	if !strings.HasPrefix(storedURL, "http://") && !strings.HasPrefix(storedURL, "https://") {
		return storedURL
	}

	// Extraer solo el path: /api/v1/storage/file/123
	u, err := url.Parse(storedURL)
	if err != nil {
		return storedURL
	}

	// En producción devolver tal cual
	if os.Getenv("GO_ENV") != "dev" {
		return storedURL
	}

	currentIP := GetOutboundIP()
	port := u.Port()
	if port != "" {
		return fmt.Sprintf("http://%s:%s%s", currentIP, port, u.Path)
	}
	return fmt.Sprintf("http://%s%s", currentIP, u.Path)
}

func ReplaceHost(host string) (string, error) {
	if strings.TrimSpace(host) == "" {
		return "", nil
	}

	isProduction := os.Getenv("GO_ENV")

	if isProduction != "dev" {
		return host, nil
	}

	u, err := url.Parse(host)
	if err != nil {
		return "", err
	}

	if p := u.Port(); p != "" {
		u.Host = net.JoinHostPort("localhost", p)
	} else {
		u.Host = "localhost"
	}

	return u.String(), nil
}

func BuildEvidenceURL(fileID string) string {
	env := strings.ToLower(strings.TrimSpace(os.Getenv("GO_ENV")))

	var apiBase string
	if env == "prod" {
		apiBase = os.Getenv("PUBLIC_API_BASE")
	} else {
		apiBase = os.Getenv("HOST_API_DEV")
	}

	apiBase = strings.TrimRight(apiBase, "/")

	return fmt.Sprintf("%s/api/v1/storage/public/%s", apiBase, fileID)
}
