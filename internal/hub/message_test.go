package hub

import "testing"

func TestRoleFromClaim(t *testing.T) {
	cases := map[string]Role{
		"wasit":    RoleReferee,
		"dewan":    RoleReferee,
		"referee":  RoleReferee,
		"juri":     RoleJudge,
		"judge":    RoleJudge,
		"admin":    RoleAdmin,
		"operator": Role("operator"),
	}
	for in, want := range cases {
		if got := RoleFromClaim(in); got != want {
			t.Errorf("RoleFromClaim(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestIsOperatorClaim(t *testing.T) {
	for _, r := range []string{"admin", "dewan", "wasit"} {
		if !IsOperatorClaim(r) {
			t.Errorf("%q harus operator", r)
		}
	}
	for _, r := range []string{"judge", "juri", "contingent", "operator", ""} {
		if IsOperatorClaim(r) {
			t.Errorf("%q tidak boleh operator", r)
		}
	}
}
