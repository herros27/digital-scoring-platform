package handler

import (
    "encoding/json"
    "net/http"
)

func Ok(w http.ResponseWriter, data any) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]any{"success": true, "data": data})
}

func Fail(w http.ResponseWriter, status int, code, message string, details any) {
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

func Decode(r *http.Request, dst any) error {
    return json.NewDecoder(r.Body).Decode(dst)
}