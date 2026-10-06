package service

import "testing"

func TestClassLabel(t *testing.T) {
	cases := []struct{ wc, g, want string }{
		{"A", "female", "Kelas A Putri"},
		{"B", "putra", "Kelas B Putra"},
		{"C", "Male", "Kelas C Putra"},
		{"D", "", "Kelas D"},
		{"E", "lainnya", "Kelas E lainnya"},
	}
	for _, c := range cases {
		if got := classLabel(c.wc, c.g); got != c.want {
			t.Errorf("classLabel(%q,%q) = %q, want %q", c.wc, c.g, got, c.want)
		}
	}
}
