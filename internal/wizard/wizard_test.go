package wizard

import "testing"

// ---------- Subdomain tests ----------

func TestSubdomain_Preview(t *testing.T) {
	got := Subdomain("acme.preview.pixelandprocess.de", "pixelandprocess.de")
	if got != "acme.preview" {
		t.Errorf("expected acme.preview, got %s", got)
	}
}

func TestSubdomain_Staging(t *testing.T) {
	got := Subdomain("acme.staging.pixelandprocess.de", "pixelandprocess.de")
	if got != "acme.staging" {
		t.Errorf("expected acme.staging, got %s", got)
	}
}

func TestSubdomain_Production(t *testing.T) {
	// Domain does not end with baseDomain, so the full domain is returned.
	got := Subdomain("acme-corp.de", "pixelandprocess.de")
	if got != "acme-corp.de" {
		t.Errorf("expected acme-corp.de, got %s", got)
	}
}

func TestSubdomain_EmptyBase(t *testing.T) {
	got := Subdomain("anything.example.com", "")
	if got != "anything.example.com" {
		t.Errorf("expected anything.example.com, got %s", got)
	}
}

func TestSubdomain_ExactMatch(t *testing.T) {
	// When domain equals baseDomain, the suffix check requires domain to be
	// strictly longer than "."+baseDomain, so the full domain is returned.
	got := Subdomain("pixelandprocess.de", "pixelandprocess.de")
	if got != "pixelandprocess.de" {
		t.Errorf("expected pixelandprocess.de, got %s", got)
	}
}

// ---------- defaultDomain tests ----------
// defaultDomain is unexported but accessible from within the same package.

func TestDefaultDomain_Preview(t *testing.T) {
	got := defaultDomain("myapp", "preview", "base.de")
	if got != "myapp.preview.base.de" {
		t.Errorf("expected myapp.preview.base.de, got %s", got)
	}
}

func TestDefaultDomain_Staging(t *testing.T) {
	got := defaultDomain("myapp", "staging", "base.de")
	if got != "myapp.staging.base.de" {
		t.Errorf("expected myapp.staging.base.de, got %s", got)
	}
}

func TestDefaultDomain_Production(t *testing.T) {
	got := defaultDomain("myapp", "production", "base.de")
	if got != "myapp.base.de" {
		t.Errorf("expected myapp.base.de, got %s", got)
	}
}
