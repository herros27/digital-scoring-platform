package handler

import (
    "encoding/json"
    "net/http"
)

func Ok(w http.ResponseWriter, data any) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]any{"success": true, "data": data})
}

func Fail(w http.ResponseWriter, status int, code, message string) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(map[string]any{
        "success": false,
        "error":   map[string]string{"code": code, "message": message},
    })
}

func Decode(r *http.Request, dst any) error {
    return json.NewDecoder(r.Body).Decode(dst)
}