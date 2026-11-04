package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/gin-gonic/gin"
)

type QueryInt interface {
	~int64 | ~uint64 | ~int | ~uint
}

var URLRegex = regexp.MustCompile(`^https?:\/\/.+\..+`)

var EmailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

var PhoneRegex = regexp.MustCompile(`^\+?[0-9\s\-\(\)]{7,20}$`)

func NormalizeSearch(input string) (string, error) {
	s := strings.TrimSpace(input)
	if s == "" {
		return "", nil
	}

	s = strings.Join(strings.Fields(s), " ")

	if len([]rune(s)) > 100 {
		return "", errors.New("search demasiado largo")
	}

	for _, r := range s {
		if unicode.IsControl(r) {
			return "", errors.New("search contiene caracteres no permitidos")
		}
	}

	return s, nil
}

func ValidateURL(link string, fieldName string) error {
	if strings.TrimSpace(link) == "" {
		return nil
	}

	if !URLRegex.MatchString(link) {
		return fmt.Errorf("el campo %s tiene un formato de URL inválido", fieldName)
	}

	_, err := url.ParseRequestURI(link)
	if err != nil {
		return fmt.Errorf("el campo %s tiene un formato de URL inválido: %s", fieldName, err.Error())
	}

	return nil
}

func ValidateJSONFormat(data string, fieldName string) error {
	if strings.TrimSpace(data) == "" {
		return fmt.Errorf("el campo %s es obligatorio y debe tener un formato JSON válido", fieldName)
	}

	var js json.RawMessage
	if err := json.Unmarshal([]byte(data), &js); err != nil {
		return fmt.Errorf("el campo %s tiene un formato JSON inválido: %s", fieldName, err.Error())
	}

	return nil
}

func ValidateParamsId(c *gin.Context, params string) (uint64, error) {
	newParams := params
	if len(strings.TrimSpace(params)) == 0 {
		newParams = "id"
	}

	idStr := c.Param(newParams)
	if len(strings.TrimSpace(idStr)) == 0 {
		return 0, errors.New("ID de producto no proporcionado")
	}

	id, newErr := strconv.ParseUint(idStr, 10, 64)
	if newErr != nil {
		return 0, errors.New("ID de producto inválido: " + newErr.Error())
	}

	return id, nil
}

func ExtractedParamsJwt(c *gin.Context) (string, uint64, error) {
	userIDIface, exists := c.Get("userID")
	if !exists {
		return "", 0, fmt.Errorf("userID no existe en el contexto")
	}

	userIDStr, ok := userIDIface.(string)
	if !ok {
		return "", 0, fmt.Errorf("userID no es string")
	}

	userID, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil {
		log.Printf("Error al convertir userID '%s' a uint64: %v", userIDStr, err)
		return "", 0, fmt.Errorf("userID inválido")
	}

	tokenIface, exists := c.Get("token")
	if !exists {
		return "", 0, fmt.Errorf("token no existe en el contexto")
	}

	tokenStr, ok := tokenIface.(string)
	if !ok {
		return "", 0, fmt.Errorf("token no es string")
	}

	return tokenStr, userID, nil
}
