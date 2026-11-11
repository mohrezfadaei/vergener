package semver

import "testing"

func TestBump(t *testing.T) {
	v, _ := Parse("1.4.7")
	for kind, want := range map[string]string{"Major": "2.0.0", "Minor": "1.5.0", "Patch": "1.4.8", "None": "1.4.7"} {
		got, err := v.Bump(kind)
		if err != nil || got.String() != want {
			t.Errorf("%s: %v %v", kind, got, err)
		}
	}
	if _, err := v.Bump("Inherit"); err == nil {
		t.Fatal("Inherit must resolve as policy")
	}
}
func TestOverflow(t *testing.T) {
	max := int(^uint(0) >> 1)
	for _, kind := range []string{"Major", "Minor", "Patch"} {
		if _, err := (Core{max, max, max}).Bump(kind); err == nil {
			t.Fatalf("%s overflowed", kind)
		}
	}
}

func TestParse(t *testing.T) {
	for _, s := range []string{"1.2.3-beta.4", "01.2.3", "garbage", "9999999999999999999999999999999999999.0.0"} {
		if _, err := Parse(s); err == nil {
			t.Errorf("accepted %s", s)
		}
	}
}
