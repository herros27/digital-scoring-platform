package handler

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOk_NilSliceDikirimSebagaiArrayKosong(t *testing.T) {
	type item struct{ ID string }
	var nilList []item
	for name, data := range map[string]any{"nil slice": nilList, "slice kosong": []item{}} {
		w := httptest.NewRecorder()
		Ok(w, data)
		if body := strings.TrimSpace(w.Body.String()); !strings.Contains(body, `"data":[]`) {
			t.Errorf("%s: body = %s, want data:[]", name, body)
		}
	}
	w := httptest.NewRecorder()
	Ok(w, map[string]any{"a": 1})
	if !strings.Contains(w.Body.String(), `"data":{"a":1}`) {
		t.Errorf("objek harus tetap objek: %s", w.Body.String())
	}
	w = httptest.NewRecorder()
	Ok(w, nil)
	if !strings.Contains(w.Body.String(), `"data":null`) {
		t.Errorf("data nil biasa tetap null: %s", w.Body.String())
	}
}
