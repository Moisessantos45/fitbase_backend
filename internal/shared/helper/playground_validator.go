package helper

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

// RegisterJSONTagName hace que los errores de validación usen el nombre del campo
// definido en el tag `json` (p. ej. "first_name") en vez del nombre del campo en Go.
// Llamar una sola vez al iniciar la app, antes de registrar rutas.
func RegisterJSONTagName() {
	v, ok := binding.Validator.Engine().(*validator.Validate)
	if !ok {
		return
	}
	v.RegisterTagNameFunc(func(f reflect.StructField) string {
		name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
		if name == "" {
			name, _, _ = strings.Cut(f.Tag.Get("form"), ",")
		}
		if name == "-" || name == "" {
			return f.Name
		}
		return name
	})
}

// isNumeric indica si el campo es numérico, para ajustar el mensaje de min/max/gte/lte.
func isNumeric(k reflect.Kind) bool {
	switch k {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	}
	return false
}

// sizeMsg construye mensajes de límites según el tipo del campo.
func sizeMsg(fe validator.FieldError, field, textual, numeric string) string {
	switch {
	case isNumeric(fe.Kind()):
		return fmt.Sprintf("El campo %s debe ser %s %s", field, numeric, fe.Param())
	case fe.Kind() == reflect.Slice || fe.Kind() == reflect.Map || fe.Kind() == reflect.Array:
		return fmt.Sprintf("El campo %s debe tener %s %s elementos", field, textual, fe.Param())
	default:
		return fmt.Sprintf("El campo %s debe tener %s %s caracteres", field, textual, fe.Param())
	}
}

// HumanValidationErrors convierte errores de validator en mensajes seguros para el cliente,
// sin exponer nombres de structs ni detalles internos.
func HumanValidationErrors(err error) []string {
	var ve validator.ValidationErrors
	if !errors.As(err, &ve) {
		return []string{"Body inválido"}
	}

	msgs := make([]string, 0, len(ve))
	for _, fe := range ve {
		field := fe.Field()

		switch fe.Tag() {
		case "required", "required_with", "required_without":
			msgs = append(msgs, fmt.Sprintf("El campo %s es obligatorio", field))
		case "email":
			msgs = append(msgs, fmt.Sprintf("El campo %s debe ser un correo válido", field))
		case "min":
			if isNumeric(fe.Kind()) {
				msgs = append(msgs, fmt.Sprintf("El campo %s debe ser como mínimo %s", field, fe.Param()))
			} else {
				msgs = append(msgs, sizeMsg(fe, field, "al menos", ""))
			}
		case "max":
			if isNumeric(fe.Kind()) {
				msgs = append(msgs, fmt.Sprintf("El campo %s debe ser como máximo %s", field, fe.Param()))
			} else {
				msgs = append(msgs, sizeMsg(fe, field, "máximo", ""))
			}
		case "gte":
			msgs = append(msgs, fmt.Sprintf("El campo %s debe ser mayor o igual a %s", field, fe.Param()))
		case "lte":
			msgs = append(msgs, fmt.Sprintf("El campo %s debe ser menor o igual a %s", field, fe.Param()))
		case "gt":
			msgs = append(msgs, fmt.Sprintf("El campo %s debe ser mayor a %s", field, fe.Param()))
		case "lt":
			msgs = append(msgs, fmt.Sprintf("El campo %s debe ser menor a %s", field, fe.Param()))
		case "len":
			msgs = append(msgs, sizeMsg(fe, field, "exactamente", "exactamente"))
		case "oneof":
			msgs = append(msgs, fmt.Sprintf("El campo %s debe ser uno de: %s", field, strings.ReplaceAll(fe.Param(), " ", ", ")))
		case "uuid", "uuid4":
			msgs = append(msgs, fmt.Sprintf("El campo %s debe ser un UUID válido", field))
		case "url", "http_url":
			msgs = append(msgs, fmt.Sprintf("El campo %s debe ser una URL válida", field))
		case "numeric", "number":
			msgs = append(msgs, fmt.Sprintf("El campo %s debe ser numérico", field))
		case "alphanum":
			msgs = append(msgs, fmt.Sprintf("El campo %s solo puede contener letras y números", field))
		default:
			msgs = append(msgs, fmt.Sprintf("El campo %s no es válido", field))
		}
	}

	return msgs
}
