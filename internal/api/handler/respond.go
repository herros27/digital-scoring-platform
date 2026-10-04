package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/herros27/digital-scoring/internal/logger"
)

var validate *validator.Validate

func init() {
	validate = validator.New()
	// ponytail: register tag name to read 'json' tag rather than struct field name for cleaner API error messages
	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})
}

// FieldError represents a specific field validation error.
type FieldError struct {
	Field   string `json:"field"`
	Tag     string `json:"tag"`
	Message string `json:"message"`
}

// ValidationError aggregates all field errors for a failed struct validation.
type ValidationError struct {
	Errors []FieldError `json:"errors"`
}

func (v *ValidationError) Error() string {
	if len(v.Errors) == 0 {
		return "validation error"
	}
	var msgs []string
	for _, e := range v.Errors {
		msgs = append(msgs, e.Message)
	}
	return strings.Join(msgs, "; ")
}

// Validate executes struct validation rules using go-playground/validator/v10.
// ponytail: skips validation without error if dst is not a struct (e.g. map, slice).
func Validate(dst any) error {
	if dst == nil {
		return nil
	}
	val := reflect.ValueOf(dst)
	if val.Kind() == reflect.Pointer {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return nil
	}

	err := validate.Struct(dst)
	if err == nil {
		return nil
	}

	var valErrs validator.ValidationErrors
	if errors.As(err, &valErrs) {
		var fieldErrors []FieldError
		for _, fe := range valErrs {
			field := fe.Field()
			if field == "" {
				field = fe.StructField()
			}
			var msg string
			switch fe.Tag() {
			case "required":
				msg = fmt.Sprintf("field '%s' is required", field)
			case "email":
				msg = fmt.Sprintf("field '%s' must be a valid email", field)
			case "min":
				msg = fmt.Sprintf("field '%s' must be at least %s characters/value", field, fe.Param())
			case "max":
				msg = fmt.Sprintf("field '%s' must be at most %s characters/value", field, fe.Param())
			case "oneof":
				msg = fmt.Sprintf("field '%s' must be one of [%s]", field, fe.Param())
			case "uuid":
				msg = fmt.Sprintf("field '%s' must be a valid UUID", field)
			default:
				msg = fmt.Sprintf("field '%s' failed on '%s' validation", field, fe.Tag())
			}
			fieldErrors = append(fieldErrors, FieldError{
				Field:   field,
				Tag:     fe.Tag(),
				Message: msg,
			})
		}
		return &ValidationError{Errors: fieldErrors}
	}

	return err
}

// Decode decodes the JSON request body and validates the destination struct.
func Decode(r *http.Request, dst any) error {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		return err
	}
	return Validate(dst)
}

// Bind decodes and validates the request body in one call, automatically returning 400 Bad Request if invalid.
func Bind(w http.ResponseWriter, r *http.Request, dst any) bool {
	if err := Decode(r, dst); err != nil {
		var valErr *ValidationError
		if errors.As(err, &valErr) {
			Fail(w, http.StatusBadRequest, "VALIDATION_ERROR", valErr.Error(), valErr.Errors)
			return false
		}
		Fail(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), nil)
		return false
	}
	return true
}

func Ok(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"success": true, "data": data})
}

func Fail(w http.ResponseWriter, status int, code, message string, details any) {
	if logger.Log != nil {
		logger.Log.Warn("API Error", "status", status, "code", code, "message", message)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{
		"success": false,
		"error": map[string]any{
			"code":    code,
			"message": message,
			"details": details,
		},
	})
}

// FailMsg — shorthand tanpa details
func FailMsg(w http.ResponseWriter, status int, code, message string) {
	Fail(w, status, code, message, nil)
}

type Pagination struct {
	Page  int `json:"page"`
	Limit int `json:"limit"`
	Total int `json:"total"`
}

func ParsePagination(r *http.Request) (page, limit, offset int) {
	page, _ = strconv.Atoi(r.URL.Query().Get("page"))
	if page <= 0 {
		page = 1
	}
	limit, _ = strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 20
	} else if limit > 100 {
		limit = 100
	}
	offset = (page - 1) * limit
	return page, limit, offset
}

func OkPaged(w http.ResponseWriter, data any, page, limit, total int) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"data":    data,
		"pagination": Pagination{
			Page:  page,
			Limit: limit,
			Total: total,
		},
	})
}
