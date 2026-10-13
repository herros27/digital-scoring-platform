package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type sampleTestPayload struct {
	Name   string `json:"name" validate:"required,min=2,max=50"`
	Email  string `json:"email" validate:"required,email"`
	Role   string `json:"role" validate:"required,oneof=admin wasit juri"`
	Age    int    `json:"age" validate:"omitempty,min=17,max=100"`
	Status string `json:"status" validate:"required,oneof=active inactive"`
}

func TestValidator_ValidPayload(t *testing.T) {
	body := `{"name":"Budi Santoso","email":"budi@silat.id","role":"wasit","age":25,"status":"active"}`
	req := httptest.NewRequest("POST", "/test", strings.NewReader(body))

	var payload sampleTestPayload
	err := Decode(req, &payload)
	if err != nil {
		t.Fatalf("expected valid decode, got error: %v", err)
	}
	if payload.Name != "Budi Santoso" || payload.Role != "wasit" {
		t.Fatalf("unexpected decoded payload: %+v", payload)
	}
}

func TestValidator_RequiredFieldMissing(t *testing.T) {
	// Name and email missing
	body := `{"role":"wasit","status":"active"}`
	req := httptest.NewRequest("POST", "/test", strings.NewReader(body))

	var payload sampleTestPayload
	err := Decode(req, &payload)
	if err == nil {
		t.Fatal("expected validation error for missing name and email, got nil")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "name") || !strings.Contains(errMsg, "required") {
		t.Errorf("expected error message to mention 'name' and 'required', got: %s", errMsg)
	}
	if !strings.Contains(errMsg, "email") || !strings.Contains(errMsg, "required") {
		t.Errorf("expected error message to mention 'email' and 'required', got: %s", errMsg)
	}
}

func TestValidator_InvalidEmailFormat(t *testing.T) {
	body := `{"name":"Budi","email":"not-an-email","role":"wasit","status":"active"}`
	req := httptest.NewRequest("POST", "/test", strings.NewReader(body))

	var payload sampleTestPayload
	err := Decode(req, &payload)
	if err == nil {
		t.Fatal("expected validation error for invalid email format")
	}

	if !strings.Contains(err.Error(), "email") {
		t.Errorf("expected error message to mention email, got: %s", err.Error())
	}
}

func TestValidator_EnumOneOfValidation(t *testing.T) {
	body := `{"name":"Budi","email":"budi@silat.id","role":"hacker","status":"active"}`
	req := httptest.NewRequest("POST", "/test", strings.NewReader(body))

	var payload sampleTestPayload
	err := Decode(req, &payload)
	if err == nil {
		t.Fatal("expected validation error for invalid role enum")
	}

	if !strings.Contains(err.Error(), "role") || !strings.Contains(err.Error(), "one of") {
		t.Errorf("expected error message to mention role enum failure, got: %s", err.Error())
	}
}

func TestValidator_BindHelper(t *testing.T) {
	// Invalid payload tested via Bind helper
	body := `{"name":"A","email":"invalid","role":"invalid_role","status":"bad"}`
	req := httptest.NewRequest("POST", "/test", strings.NewReader(body))
	w := httptest.NewRecorder()

	var payload sampleTestPayload
	ok := Bind(w, req, &payload)
	if ok {
		t.Fatal("expected Bind to return false on invalid payload")
	}

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d", w.Code)
	}

	var res struct {
		Success bool `json:"success"`
		Error   struct {
			Code    string       `json:"code"`
			Message string       `json:"message"`
			Details []FieldError `json:"details"`
		} `json:"error"`
	}

	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if res.Success {
		t.Errorf("expected success=false, got true")
	}
	if res.Error.Code != "VALIDATION_ERROR" {
		t.Errorf("expected error code VALIDATION_ERROR, got %s", res.Error.Code)
	}
	if len(res.Error.Details) == 0 {
		t.Errorf("expected validation details to contain field errors")
	}
}

func TestValidator_NonStructSafeBypass(t *testing.T) {
	// Map or slice payloads should safely bypass struct validation without panicking
	body := `{"key":"value"}`
	req := httptest.NewRequest("POST", "/test", bytes.NewReader([]byte(body)))

	var m map[string]string
	err := Decode(req, &m)
	if err != nil {
		t.Fatalf("expected map decode to succeed without validation error, got: %v", err)
	}
	if m["key"] != "value" {
		t.Fatalf("unexpected map value: %v", m)
	}
}

func TestAuthHandler_Logout(t *testing.T) {
	h := &AuthHandler{}
	req := httptest.NewRequest("POST", "/api/v1/auth/logout", nil)
	rec := httptest.NewRecorder()

	h.Logout(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got: %d", rec.Code)
	}

	var res struct {
		Success bool `json:"success"`
		Data    struct {
			Message string `json:"message"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !res.Success || res.Data.Message != "successfully logged out" {
		t.Fatalf("unexpected logout response: %+v", res)
	}
}

// Isi permintaan di atas batas ditolak (tidak dibaca tanpa akhir); isi normal tetap diterima.
func TestDecode_RejectsOversizedBody(t *testing.T) {
	var dst struct {
		Note string `json:"note"`
	}
	big := `{"note":"` + strings.Repeat("a", maxBodyBytes) + `"}`
	if err := Decode(httptest.NewRequest("POST", "/", strings.NewReader(big)), &dst); err == nil {
		t.Fatal("isi lebih dari batas harus ditolak")
	}
	if err := Decode(httptest.NewRequest("POST", "/", strings.NewReader(`{"note":"ok"}`)), &dst); err != nil || dst.Note != "ok" {
		t.Fatalf("isi normal harus diterima: %v %q", err, dst.Note)
	}
}

// PIN dengan role admin/dewan/operator akan menghasilkan token admin dari 6 digit angka; hanya juri dan wasit.
func TestCreatePIN_RejectsPrivilegedRoles(t *testing.T) {
	h := &AuthHandler{} // ditolak oleh validasi sebelum menyentuh database
	for _, role := range []string{"admin", "dewan", "operator", "referee"} {
		body := `{"competition_id":"c","arena_id":"a","role":"` + role + `"}`
		rec := httptest.NewRecorder()
		h.CreatePIN(rec, httptest.NewRequest("POST", "/api/v1/auth/create-pin", strings.NewReader(body)))
		if rec.Code != http.StatusBadRequest {
			t.Errorf("role %q: status = %d, want 400", role, rec.Code)
		}
	}
}

// Daftar kontingen memakai batas password yang sama dengan reset/ganti password (8 karakter).
func TestContingentRegister_RejectsShortPassword(t *testing.T) {
	h := &ContingentHandler{} // ditolak oleh validasi sebelum menyentuh service
	body := `{"name":"Tim","pic_name":"PIC","email":"a@b.id","phone":"081234","password":"1234567"}`
	rec := httptest.NewRecorder()
	h.Register(rec, httptest.NewRequest("POST", "/api/v1/auth/contingent/register", strings.NewReader(body)))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("password 7 karakter: status = %d, want 400", rec.Code)
	}
}
