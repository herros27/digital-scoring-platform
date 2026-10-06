package hub

import "testing"

func TestReleaseOnlyByOwner(t *testing.T) {
	h := NewHub("a1")
	first, second := new(int), new(int)
	h.Attach(first, func(string, *Client, Message) {}, func() {})
	h.Attach(second, func(string, *Client, Message) {}, func() {})

	h.Release(first) // bukan pemilik: tidak boleh melepas callback milik partai berikutnya
	if h.OnMessage == nil {
		t.Fatal("Release oleh non-pemilik melepas callback")
	}
	h.Release(second)
	if h.OnMessage != nil || h.onPresence != nil {
		t.Fatal("Release oleh pemilik harus melepas callback")
	}
}
