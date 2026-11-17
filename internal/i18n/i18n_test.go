package i18n

import "testing"

func TestNormalize(t *testing.T) {
	for in, want := range map[string]Lang{"fr": FR, "fr_FR.UTF-8": FR, "FR-ca": FR, "en_US.UTF-8": EN, "en": EN} {
		if got, ok := Normalize(in); !ok || got != want {
			t.Errorf("Normalize(%q) = %v,%v", in, got, ok)
		}
	}
	for _, in := range []string{"", "C", "de_DE"} {
		if _, ok := Normalize(in); ok {
			t.Errorf("Normalize(%q) should be unsupported", in)
		}
	}
}

func TestDetect(t *testing.T) {
	for _, k := range []string{"WTUI_LANG", "LC_ALL", "LC_MESSAGES"} {
		t.Setenv(k, "")
	}
	t.Setenv("LANG", "fr_FR.UTF-8")
	if Detect() != FR {
		t.Error("expected fr")
	}
	t.Setenv("LC_ALL", "en_US.UTF-8")
	if Detect() != EN {
		t.Error("LC_ALL should win")
	}
	t.Setenv("WTUI_LANG", "fr")
	if Detect() != FR {
		t.Error("WTUI_LANG should win")
	}
}

func TestCatalogsMatch(t *testing.T) {
	for k := range catalog[EN] {
		if _, ok := catalog[FR][k]; !ok {
			t.Errorf("key %q missing in fr", k)
		}
	}
	for k := range catalog[FR] {
		if _, ok := catalog[EN][k]; !ok {
			t.Errorf("key %q missing in en", k)
		}
	}
}

func TestT(t *testing.T) {
	defer Set(EN)
	Set(FR)
	if T("help.quit") != "quitter" {
		t.Error("fr translation")
	}
	if T("nope") != "nope" {
		t.Error("key fallback")
	}
	Set(EN)
	if T("err.lang", "x") != `unsupported language "x" (use en or fr)` {
		t.Error("format")
	}
}
