package handler

import (
    "encoding/json"
    "net/http"
    "strconv"
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