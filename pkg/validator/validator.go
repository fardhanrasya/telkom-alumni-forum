package validator

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

func FormatValidationError(err error) string {
	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		var messages []string
		for _, fieldError := range validationErrors {
			message := getFieldErrorMessage(fieldError)
			messages = append(messages, message)
		}
		return strings.Join(messages, "; ")
	}
	return err.Error()
}

func getFieldErrorMessage(fe validator.FieldError) string {
	field := getFieldName(fe.Field())

	switch fe.Tag() {
	case "required":
		return fmt.Sprintf("%s wajib diisi", field)
	case "email":
		return fmt.Sprintf("%s harus berupa email yang valid", field)
	case "min":
		if fe.Type().String() == "string" {
			return fmt.Sprintf("%s minimal %s karakter", field, fe.Param())
		}
		return fmt.Sprintf("%s minimal %s", field, fe.Param())
	case "max":
		if fe.Type().String() == "string" {
			return fmt.Sprintf("%s maksimal %s karakter", field, fe.Param())
		}
		return fmt.Sprintf("%s maksimal %s", field, fe.Param())
	default:
		return fmt.Sprintf("%s tidak valid", field)
	}
}

func getFieldName(field string) string {
	fieldNames := map[string]string{
		"Username":       "Username",
		"Email":          "Email",
		"Password":       "Password",
		"Role":           "Role",
		"FullName":       "Nama lengkap",
		"IdentityNumber": "Nomor identitas",
		"Angkatan":       "Angkatan",
		"Bio":            "Bio",
	}

	if name, ok := fieldNames[field]; ok {
		return name
	}
	return field
}
