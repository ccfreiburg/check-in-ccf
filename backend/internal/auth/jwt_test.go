package auth_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ccf/check-in/backend/internal/auth"
)

var testSecret = []byte("test-secret-key-long-enough-32b!")

// ── helpers ───────────────────────────────────────────────────────────────

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

// ── NewVolunteerToken ─────────────────────────────────────────────────────

func TestNewVolunteerToken_RoleIsVolunteer(t *testing.T) {
	tok, err := auth.NewVolunteerToken(testSecret)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := auth.ParseToken(testSecret, tok)
	if err != nil {
		t.Fatal(err)
	}
	if claims.Role != "volunteer" {
		t.Errorf("expected role=volunteer, got %q", claims.Role)
	}
	if claims.ParentID != 0 {
		t.Errorf("expected parentId=0, got %d", claims.ParentID)
	}
}

// ── NewAdminToken ─────────────────────────────────────────────────────────

func TestNewAdminToken_RoleIsAdmin(t *testing.T) {
	tok, err := auth.NewAdminToken(testSecret)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := auth.ParseToken(testSecret, tok)
	if err != nil {
		t.Fatal(err)
	}
	if claims.Role != "admin" {
		t.Errorf("expected role=admin, got %q", claims.Role)
	}
}

// ── NewParentToken ────────────────────────────────────────────────────────

func TestNewParentToken_RoleAndID(t *testing.T) {
	tok, err := auth.NewParentToken(testSecret, 42)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := auth.ParseToken(testSecret, tok)
	if err != nil {
		t.Fatal(err)
	}
	if claims.Role != "parent" {
		t.Errorf("expected role=parent, got %q", claims.Role)
	}
	if claims.ParentID != 42 {
		t.Errorf("expected parentId=42, got %d", claims.ParentID)
	}
}

// ── ParseToken ────────────────────────────────────────────────────────────

func TestParseToken_WrongSecret_Fails(t *testing.T) {
	tok, _ := auth.NewAdminToken(testSecret)
	_, err := auth.ParseToken([]byte("wrong-secret-key-long-enough-32b"), tok)
	if err == nil {
		t.Error("expected error for wrong secret, got nil")
	}
}

func TestParseToken_Malformed_Fails(t *testing.T) {
	_, err := auth.ParseToken(testSecret, "not.a.valid.token")
	if err == nil {
		t.Error("expected error for malformed token, got nil")
	}
}

func TestParseToken_Empty_Fails(t *testing.T) {
	_, err := auth.ParseToken(testSecret, "")
	if err == nil {
		t.Error("expected error for empty token, got nil")
	}
}

// ── VolunteerMiddleware ───────────────────────────────────────────────────

func TestVolunteerMiddleware_NoHeader_Returns401(t *testing.T) {
	mw := auth.VolunteerMiddleware(testSecret)
	handler := mw(okHandler())
	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestVolunteerMiddleware_InvalidToken_Returns403(t *testing.T) {
	mw := auth.VolunteerMiddleware(testSecret)
	handler := mw(okHandler())
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer not-a-token")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rr.Code)
	}
}

func TestVolunteerMiddleware_VolunteerToken_Passes(t *testing.T) {
	tok, _ := auth.NewVolunteerToken(testSecret)
	mw := auth.VolunteerMiddleware(testSecret)
	reached := false
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reached = true
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	if !reached {
		t.Error("inner handler was not reached")
	}
}

func TestVolunteerMiddleware_AdminToken_Passes(t *testing.T) {
	tok, _ := auth.NewAdminToken(testSecret)
	mw := auth.VolunteerMiddleware(testSecret)
	handler := mw(okHandler())
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("admin token should also pass volunteer middleware; got %d", rr.Code)
	}
}

func TestVolunteerMiddleware_ParentToken_Returns403(t *testing.T) {
	tok, _ := auth.NewParentToken(testSecret, 1)
	mw := auth.VolunteerMiddleware(testSecret)
	handler := mw(okHandler())
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403 for parent token, got %d", rr.Code)
	}
}

func TestVolunteerMiddleware_MissingBearerPrefix_Returns401(t *testing.T) {
	tok, _ := auth.NewVolunteerToken(testSecret)
	mw := auth.VolunteerMiddleware(testSecret)
	handler := mw(okHandler())
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", tok) // no "Bearer " prefix
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 without Bearer prefix, got %d", rr.Code)
	}
}

// ── AdminMiddleware ───────────────────────────────────────────────────────

func TestAdminMiddleware_NoHeader_Returns401(t *testing.T) {
	mw := auth.AdminMiddleware(testSecret)
	handler := mw(okHandler())
	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestAdminMiddleware_VolunteerToken_Returns403(t *testing.T) {
	tok, _ := auth.NewVolunteerToken(testSecret)
	mw := auth.AdminMiddleware(testSecret)
	handler := mw(okHandler())
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403 for volunteer token, got %d", rr.Code)
	}
}

func TestAdminMiddleware_AdminToken_Passes(t *testing.T) {
	tok, _ := auth.NewAdminToken(testSecret)
	mw := auth.AdminMiddleware(testSecret)
	reached := false
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reached = true
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	if !reached {
		t.Error("inner handler was not reached")
	}
}

func TestAdminMiddleware_ParentToken_Returns403(t *testing.T) {
	tok, _ := auth.NewParentToken(testSecret, 1)
	mw := auth.AdminMiddleware(testSecret)
	handler := mw(okHandler())
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rr.Code)
	}
}

// ── ValidateParentToken ───────────────────────────────────────────────────

func TestValidateParentToken_Valid(t *testing.T) {
	tok, _ := auth.NewParentToken(testSecret, 99)
	claims, err := auth.ValidateParentToken(testSecret, tok)
	if err != nil {
		t.Fatal(err)
	}
	if claims.ParentID != 99 {
		t.Errorf("expected parentId=99, got %d", claims.ParentID)
	}
}

func TestValidateParentToken_AdminToken_Fails(t *testing.T) {
	tok, _ := auth.NewAdminToken(testSecret)
	_, err := auth.ValidateParentToken(testSecret, tok)
	if err == nil {
		t.Error("expected error for admin token passed as parent token")
	}
}

func TestValidateParentToken_VolunteerToken_Fails(t *testing.T) {
	tok, _ := auth.NewVolunteerToken(testSecret)
	_, err := auth.ValidateParentToken(testSecret, tok)
	if err == nil {
		t.Error("expected error for volunteer token passed as parent token")
	}
}

func TestValidateParentToken_WrongSecret_Fails(t *testing.T) {
	tok, _ := auth.NewParentToken(testSecret, 1)
	_, err := auth.ValidateParentToken([]byte("different-secret-key-long-enough-!!"), tok)
	if err == nil {
		t.Error("expected error for wrong secret")
	}
}
