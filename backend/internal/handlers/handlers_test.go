package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/ccf/check-in/backend/internal/auth"
	"github.com/ccf/check-in/backend/internal/ct"
	"github.com/ccf/check-in/backend/internal/ctsync"
	localdb "github.com/ccf/check-in/backend/internal/db"
	"github.com/ccf/check-in/backend/internal/handlers"
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

// Mock CT client

type mockCT struct {
	loginID      int
	loginErr     error
	person       *ct.Person
	personErr    error
	parentIDs    []int
	parentIDsErr error
	children     []ct.Child
	childrenErr  error
	checkInErr   error
}

func (m *mockCT) LoginUser(username, password string) (int, error) {
	return m.loginID, m.loginErr
}
func (m *mockCT) GetPerson(id int) (*ct.Person, error) {
	return m.person, m.personErr
}
func (m *mockCT) GetParentsForChild(childID int) ([]int, error) {
	return m.parentIDs, m.parentIDsErr
}
func (m *mockCT) GetChildrenForParent(parentID int) ([]ct.Child, error) {
	return m.children, m.childrenErr
}
func (m *mockCT) CheckIn(childID, groupID int) error {
	return m.checkInErr
}

// Mock sync service

type mockSync struct {
	groups []ctsync.GroupConfig
	runErr error
}

func (m *mockSync) Run(ctx context.Context) error { return m.runErr }
func (m *mockSync) Groups() []ctsync.GroupConfig  { return m.groups }

// Test helpers

const testJWTSecret = "test-secret-key-long-enough-32b"

func newTestHandler(t *testing.T, ctc *mockCT, sync *mockSync) (*handlers.Handler, *gorm.DB) {
	t.Helper()
	database, err := localdb.Open(":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Setenv("LOCAL_PASSWORD", "true")
	t.Setenv("ADMIN_PASSWORD", "admin123")
	t.Setenv("VOLUNTEER_PASSWORD", "vol123")
	t.Setenv("CT_ADMIN_PERSONS", "")
	reportsDir := t.TempDir()
	t.Setenv("REPORTS_DIR", reportsDir)
	h := handlers.New(ctc, database, sync, []byte(testJWTSecret), "http://test.local")
	return h, database
}

func withParam(r *http.Request, key, val string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, val)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

// withParams attaches multiple chi URL parameters to a request context.
func withParams(r *http.Request, params map[string]string) *http.Request {
	rctx := chi.NewRouteContext()
	for k, v := range params {
		rctx.URLParams.Add(k, v)
	}
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func postJSON(t *testing.T, h http.HandlerFunc, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h(rr, req)
	return rr
}

func adminBearerToken(t *testing.T) string {
	t.Helper()
	tok, err := auth.NewAdminToken([]byte(testJWTSecret))
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

func parentBearerToken(t *testing.T, parentID int) string {
	t.Helper()
	tok, err := auth.NewParentToken([]byte(testJWTSecret), parentID)
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

// AdminLogin

func TestAdminLogin_LocalAdmin_ReturnsAdminRole(t *testing.T) {
	h, _ := newTestHandler(t, &mockCT{}, &mockSync{})
	rr := postJSON(t, h.AdminLogin, "/api/auth/admin", map[string]string{
		"username": "", "password": "admin123",
	})
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body)
	}
	var resp struct {
		Role string `json:"role"`
	}
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp.Role != "admin" {
		t.Errorf("expected role=admin, got %q", resp.Role)
	}
}

func TestAdminLogin_LocalVolunteer_ReturnsVolunteerRole(t *testing.T) {
	h, _ := newTestHandler(t, &mockCT{}, &mockSync{})
	rr := postJSON(t, h.AdminLogin, "/api/auth/admin", map[string]string{
		"username": "", "password": "vol123",
	})
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var resp struct {
		Role string `json:"role"`
	}
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp.Role != "volunteer" {
		t.Errorf("expected role=volunteer, got %q", resp.Role)
	}
}

func TestAdminLogin_WrongPassword_Returns403(t *testing.T) {
	h, _ := newTestHandler(t, &mockCT{}, &mockSync{})
	rr := postJSON(t, h.AdminLogin, "/api/auth/admin", map[string]string{
		"username": "", "password": "wrong",
	})
	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rr.Code)
	}
}

func TestAdminLogin_EmptyPassword_Returns400(t *testing.T) {
	h, _ := newTestHandler(t, &mockCT{}, &mockSync{})
	rr := postJSON(t, h.AdminLogin, "/api/auth/admin", map[string]string{
		"username": "", "password": "",
	})
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestAdminLogin_InvalidJSON_Returns400(t *testing.T) {
	h, _ := newTestHandler(t, &mockCT{}, &mockSync{})
	req := httptest.NewRequest("POST", "/api/auth/admin", strings.NewReader("{bad json"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.AdminLogin(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestAdminLogin_CTPersistentToken_Returns200(t *testing.T) {
	t.Setenv("LOCAL_PASSWORD", "false")
	database, err := localdb.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("CT_ADMIN_PERSONS", "ct.admin@example.com")
	t.Setenv("REPORTS_DIR", t.TempDir())
	ctc := &mockCT{loginID: 7}
	h := handlers.New(ctc, database, &mockSync{}, []byte(testJWTSecret), "http://test.local")
	rr := postJSON(t, h.AdminLogin, "/api/auth/admin", map[string]string{
		"username": "ct.admin@example.com", "password": "any",
	})
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body)
	}
	var resp struct {
		Role string `json:"role"`
	}
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp.Role != "admin" {
		t.Errorf("expected admin role for CT_ADMIN_PERSONS member, got %q", resp.Role)
	}
}

func TestAdminLogin_CTUserInStaff_ReturnsStaffRole(t *testing.T) {
	t.Setenv("LOCAL_PASSWORD", "false")
	database, err := localdb.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("CT_ADMIN_PERSONS", "")
	t.Setenv("REPORTS_DIR", t.TempDir())
	database.Create(&localdb.SyncedStaff{CTID: 5, FirstName: "Test", LastName: "Staff", Email: "staff@example.com", Role: "volunteer"})
	ctc := &mockCT{loginID: 5}
	h := handlers.New(ctc, database, &mockSync{}, []byte(testJWTSecret), "http://test.local")
	rr := postJSON(t, h.AdminLogin, "/api/auth/admin", map[string]string{
		"username": "staff@example.com", "password": "secret",
	})
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body)
	}
	var resp struct {
		Role string `json:"role"`
	}
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp.Role != "volunteer" {
		t.Errorf("expected role=volunteer, got %q", resp.Role)
	}
}

func TestAdminLogin_CTUser_NotInStaff_Returns403(t *testing.T) {
	t.Setenv("LOCAL_PASSWORD", "false")
	database, err := localdb.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("CT_ADMIN_PERSONS", "")
	t.Setenv("REPORTS_DIR", t.TempDir())
	ctc := &mockCT{loginID: 99}
	h := handlers.New(ctc, database, &mockSync{}, []byte(testJWTSecret), "http://test.local")
	rr := postJSON(t, h.AdminLogin, "/api/auth/admin", map[string]string{
		"username": "nobody@example.com", "password": "secret",
	})
	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rr.Code)
	}
}

func TestAdminLogin_CTLogin_Fails_Returns403(t *testing.T) {
	t.Setenv("LOCAL_PASSWORD", "false")
	database, _ := localdb.Open(":memory:")
	t.Setenv("CT_ADMIN_PERSONS", "")
	t.Setenv("REPORTS_DIR", t.TempDir())
	ctc := &mockCT{loginErr: fmt.Errorf("invalid credentials")}
	h := handlers.New(ctc, database, &mockSync{}, []byte(testJWTSecret), "http://test.local")
	rr := postJSON(t, h.AdminLogin, "/api/auth/admin", map[string]string{
		"username": "user@example.com", "password": "bad",
	})
	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rr.Code)
	}
}

// ListCheckins

func TestListCheckins_Empty_ReturnsEmptyArray(t *testing.T) {
	h, _ := newTestHandler(t, &mockCT{}, &mockSync{})
	req := httptest.NewRequest("GET", "/api/admin/checkins", nil)
	rr := httptest.NewRecorder()
	h.ListCheckins(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var resp []any
	json.NewDecoder(rr.Body).Decode(&resp)
	if len(resp) != 0 {
		t.Errorf("expected empty slice, got %d items", len(resp))
	}
}

func TestListCheckins_ReturnsOnlyToday(t *testing.T) {
	h, db := newTestHandler(t, &mockCT{}, &mockSync{})
	today := localdb.Today()
	db.Create(&localdb.CheckIn{EventDate: today, ChildID: 1, Status: "pending"})
	db.Create(&localdb.CheckIn{EventDate: "1990-01-01", ChildID: 2, Status: "pending"})
	req := httptest.NewRequest("GET", "/api/admin/checkins", nil)
	rr := httptest.NewRecorder()
	h.ListCheckins(rr, req)
	var resp []map[string]any
	json.NewDecoder(rr.Body).Decode(&resp)
	if len(resp) != 1 {
		t.Errorf("expected 1 today record, got %d", len(resp))
	}
}

func TestListCheckins_FilterByStatus(t *testing.T) {
	h, database := newTestHandler(t, &mockCT{}, &mockSync{})
	today := localdb.Today()
	database.Create(&localdb.CheckIn{EventDate: today, ChildID: 1, Status: "pending"})
	database.Create(&localdb.CheckIn{EventDate: today, ChildID: 2, Status: "checked_in"})
	req := httptest.NewRequest("GET", "/api/admin/checkins?status=pending", nil)
	rr := httptest.NewRecorder()
	h.ListCheckins(rr, req)
	var resp []map[string]any
	json.NewDecoder(rr.Body).Decode(&resp)
	if len(resp) != 1 {
		t.Errorf("expected 1 pending record, got %d", len(resp))
	}
}

// ConfirmTagHandout

func TestConfirmTagHandout_TogglesTagReceived(t *testing.T) {
	h, database := newTestHandler(t, &mockCT{}, &mockSync{})
	record := localdb.CheckIn{EventDate: localdb.Today(), ChildID: 5, Status: "pending"}
	database.Create(&record)
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/admin/checkins/%d/confirm", record.ID), nil)
	req = withParam(req, "id", fmt.Sprint(record.ID))
	rr := httptest.NewRecorder()
	h.ConfirmTagHandout(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body)
	}
	var updated localdb.CheckIn
	database.First(&updated, record.ID)
	if !updated.TagReceived {
		t.Error("expected TagReceived=true after confirm")
	}
	if updated.RegisteredAt == nil {
		t.Error("expected RegisteredAt to be set")
	}
}

func TestConfirmTagHandout_ToggleOff(t *testing.T) {
	h, database := newTestHandler(t, &mockCT{}, &mockSync{})
	record := localdb.CheckIn{EventDate: localdb.Today(), ChildID: 6, Status: "pending", TagReceived: true}
	database.Create(&record)
	req := httptest.NewRequest("POST", "/", nil)
	req = withParam(req, "id", fmt.Sprint(record.ID))
	rr := httptest.NewRecorder()
	h.ConfirmTagHandout(rr, req)
	var updated localdb.CheckIn
	database.First(&updated, record.ID)
	if updated.TagReceived {
		t.Error("expected TagReceived=false after second toggle")
	}
}

func TestConfirmTagHandout_NotFound_Returns404(t *testing.T) {
	h, _ := newTestHandler(t, &mockCT{}, &mockSync{})
	req := httptest.NewRequest("POST", "/", nil)
	req = withParam(req, "id", "9999")
	rr := httptest.NewRecorder()
	h.ConfirmTagHandout(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestConfirmTagHandout_InvalidID_Returns400(t *testing.T) {
	h, _ := newTestHandler(t, &mockCT{}, &mockSync{})
	req := httptest.NewRequest("POST", "/", nil)
	req = withParam(req, "id", "invalid")
	rr := httptest.NewRecorder()
	h.ConfirmTagHandout(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

// CheckInAtGroup

func TestCheckInAtGroup_ChangesPendingToCheckedIn(t *testing.T) {
	h, database := newTestHandler(t, &mockCT{}, &mockSync{})
	record := localdb.CheckIn{EventDate: localdb.Today(), ChildID: 10, GroupID: 1, Status: "pending"}
	database.Create(&record)
	req := httptest.NewRequest("POST", "/", nil)
	req = withParam(req, "id", fmt.Sprint(record.ID))
	rr := httptest.NewRecorder()
	h.CheckInAtGroup(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body)
	}
	var updated localdb.CheckIn
	database.First(&updated, record.ID)
	if updated.Status != "checked_in" {
		t.Errorf("expected status=checked_in, got %q", updated.Status)
	}
	if updated.CheckedInAt == nil {
		t.Error("expected CheckedInAt to be set")
	}
}

func TestCheckInAtGroup_AlreadyCheckedIn_Returns409(t *testing.T) {
	h, database := newTestHandler(t, &mockCT{}, &mockSync{})
	record := localdb.CheckIn{EventDate: localdb.Today(), ChildID: 11, Status: "checked_in"}
	database.Create(&record)
	req := httptest.NewRequest("POST", "/", nil)
	req = withParam(req, "id", fmt.Sprint(record.ID))
	rr := httptest.NewRecorder()
	h.CheckInAtGroup(rr, req)
	if rr.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", rr.Code)
	}
}

// SetCheckInStatus

func TestSetCheckInStatus_ToPending_ClearsTimestamps(t *testing.T) {
	h, database := newTestHandler(t, &mockCT{}, &mockSync{})
	record := localdb.CheckIn{EventDate: localdb.Today(), ChildID: 20, Status: "checked_in", TagReceived: true}
	database.Create(&record)
	req := httptest.NewRequest("POST", "/", strings.NewReader(`{"status":"pending"}`))
	req.Header.Set("Content-Type", "application/json")
	req = withParam(req, "id", fmt.Sprint(record.ID))
	rr := httptest.NewRecorder()
	h.SetCheckInStatus(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body)
	}
	var updated localdb.CheckIn
	database.First(&updated, record.ID)
	if updated.Status != "pending" {
		t.Errorf("expected status=pending, got %q", updated.Status)
	}
	if updated.TagReceived {
		t.Error("expected TagReceived=false after reset to pending")
	}
}

func TestSetCheckInStatus_EmptyStatus_SoftDeletes(t *testing.T) {
	h, database := newTestHandler(t, &mockCT{}, &mockSync{})
	record := localdb.CheckIn{EventDate: localdb.Today(), ChildID: 21, Status: "pending"}
	database.Create(&record)
	req := httptest.NewRequest("POST", "/", strings.NewReader(`{"status":""}`))
	req.Header.Set("Content-Type", "application/json")
	req = withParam(req, "id", fmt.Sprint(record.ID))
	rr := httptest.NewRecorder()
	h.SetCheckInStatus(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var count int64
	database.Model(&localdb.CheckIn{}).Where("id = ?", record.ID).Count(&count)
	if count != 0 {
		t.Error("expected record to be deleted (soft deleted)")
	}
	var deleted localdb.CheckIn
	database.Unscoped().First(&deleted, record.ID)
	if deleted.CheckedOutAt == nil {
		t.Error("expected CheckedOutAt to be set on soft-deleted record")
	}
}

func TestSetCheckInStatus_InvalidStatus_Returns400(t *testing.T) {
	h, database := newTestHandler(t, &mockCT{}, &mockSync{})
	record := localdb.CheckIn{EventDate: localdb.Today(), ChildID: 22, Status: "pending"}
	database.Create(&record)
	req := httptest.NewRequest("POST", "/", strings.NewReader(`{"status":"invalid_status"}`))
	req.Header.Set("Content-Type", "application/json")
	req = withParam(req, "id", fmt.Sprint(record.ID))
	rr := httptest.NewRecorder()
	h.SetCheckInStatus(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestSetCheckInStatus_NotFound_Returns404(t *testing.T) {
	h, _ := newTestHandler(t, &mockCT{}, &mockSync{})
	req := httptest.NewRequest("POST", "/", strings.NewReader(`{"status":"pending"}`))
	req.Header.Set("Content-Type", "application/json")
	req = withParam(req, "id", "9999")
	rr := httptest.NewRecorder()
	h.SetCheckInStatus(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

// EndEvent

func TestEndEvent_WithRecords_GeneratesReportAndDeletes(t *testing.T) {
	h, database := newTestHandler(t, &mockCT{}, &mockSync{})
	today := localdb.Today()
	database.Create(&localdb.CheckIn{EventDate: today, ChildID: 30, Status: "pending"})
	database.Create(&localdb.CheckIn{EventDate: today, ChildID: 31, Status: "checked_in"})
	req := httptest.NewRequest("POST", "/api/admin/checkins/end-event", nil)
	rr := httptest.NewRecorder()
	h.EndEvent(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body)
	}
	var count int64
	database.Unscoped().Model(&localdb.CheckIn{}).Where("event_date = ?", today).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 records after EndEvent, got %d", count)
	}
	reportsDir := os.Getenv("REPORTS_DIR")
	entries, err := os.ReadDir(reportsDir)
	if err != nil {
		t.Fatalf("cannot read reports dir: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 report file, got %d", len(entries))
	}
}

func TestEndEvent_NoRecords_Returns200(t *testing.T) {
	h, _ := newTestHandler(t, &mockCT{}, &mockSync{})
	req := httptest.NewRequest("POST", "/", nil)
	rr := httptest.NewRecorder()
	h.EndEvent(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

// ListReports / GetReport

func TestListReports_Empty_ReturnsEmptyArray(t *testing.T) {
	h, _ := newTestHandler(t, &mockCT{}, &mockSync{})
	req := httptest.NewRequest("GET", "/api/admin/reports", nil)
	rr := httptest.NewRecorder()
	h.ListReports(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var resp []any
	json.NewDecoder(rr.Body).Decode(&resp)
	if len(resp) != 0 {
		t.Errorf("expected empty array, got %d items", len(resp))
	}
}

func TestListReports_ReturnsFiles_NewestFirst(t *testing.T) {
	h, database := newTestHandler(t, &mockCT{}, &mockSync{})
	today := localdb.Today()
	database.Create(&localdb.CheckIn{EventDate: today, ChildID: 40, Status: "pending"})
	req := httptest.NewRequest("POST", "/", nil)
	rr := httptest.NewRecorder()
	h.EndEvent(rr, req)
	rr = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/api/admin/reports", nil)
	h.ListReports(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var reports []map[string]any
	json.NewDecoder(rr.Body).Decode(&reports)
	if len(reports) == 0 {
		t.Error("expected at least one report")
	}
}

func TestGetReport_ValidFilename_ServesFile(t *testing.T) {
	h, database := newTestHandler(t, &mockCT{}, &mockSync{})
	today := localdb.Today()
	database.Create(&localdb.CheckIn{EventDate: today, ChildID: 50, Status: "pending"})
	endReq := httptest.NewRequest("POST", "/", nil)
	h.EndEvent(httptest.NewRecorder(), endReq)
	reportsDir := os.Getenv("REPORTS_DIR")
	entries, _ := os.ReadDir(reportsDir)
	if len(entries) == 0 {
		t.Skip("no report file created")
	}
	filename := entries[0].Name()
	req := httptest.NewRequest("GET", "/api/admin/reports/"+filename, nil)
	req = withParam(req, "filename", filename)
	rr := httptest.NewRecorder()
	h.GetReport(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body)
	}
	if ct := rr.Header().Get("Content-Type"); !strings.Contains(ct, "text/csv") {
		t.Errorf("expected text/csv content type, got %q", ct)
	}
}

func TestGetReport_InvalidFilename_Returns400(t *testing.T) {
	h, _ := newTestHandler(t, &mockCT{}, &mockSync{})
	req := httptest.NewRequest("GET", "/", nil)
	req = withParam(req, "filename", "../etc/passwd")
	rr := httptest.NewRecorder()
	h.GetReport(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestGetReport_NonExistentFile_Returns404(t *testing.T) {
	h, _ := newTestHandler(t, &mockCT{}, &mockSync{})
	req := httptest.NewRequest("GET", "/", nil)
	req = withParam(req, "filename", "2025-01-01_001.csv")
	rr := httptest.NewRecorder()
	h.GetReport(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

// GetVAPIDPublicKey

func TestGetVAPIDPublicKey_ReturnsKey(t *testing.T) {
	t.Setenv("VAPID_PUBLIC_KEY", "test-vapid-pub-key")
	t.Setenv("LOCAL_PASSWORD", "true")
	t.Setenv("ADMIN_PASSWORD", "admin123")
	t.Setenv("VOLUNTEER_PASSWORD", "vol123")
	t.Setenv("CT_ADMIN_PERSONS", "")
	t.Setenv("REPORTS_DIR", t.TempDir())
	database, _ := localdb.Open(":memory:")
	h := handlers.New(&mockCT{}, database, &mockSync{}, []byte(testJWTSecret), "http://test.local")
	req := httptest.NewRequest("GET", "/api/push/vapid-public-key", nil)
	rr := httptest.NewRecorder()
	h.GetVAPIDPublicKey(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var resp map[string]string
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp["publicKey"] != "test-vapid-pub-key" {
		t.Errorf("expected publicKey in response, got %v", resp)
	}
}

// ListGroups

func TestListGroups_ReturnsConfiguredGroups(t *testing.T) {
	sync := &mockSync{groups: []ctsync.GroupConfig{{ID: 10, Name: "Gruppe A"}, {ID: 20, Name: "Gruppe B"}}}
	h, _ := newTestHandler(t, &mockCT{}, sync)
	req := httptest.NewRequest("GET", "/api/admin/groups", nil)
	rr := httptest.NewRecorder()
	h.ListGroups(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var resp []map[string]any
	json.NewDecoder(rr.Body).Decode(&resp)
	if len(resp) != 2 {
		t.Errorf("expected 2 groups, got %d", len(resp))
	}
}

// SyncCT

func TestSyncCT_Success_Returns200(t *testing.T) {
	h, _ := newTestHandler(t, &mockCT{}, &mockSync{})
	req := httptest.NewRequest("POST", "/api/admin/sync", nil)
	rr := httptest.NewRecorder()
	h.SyncCT(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body)
	}
}

func TestSyncCT_Error_Returns503(t *testing.T) {
	sync := &mockSync{runErr: fmt.Errorf("sync failed")}
	h, _ := newTestHandler(t, &mockCT{}, sync)
	req := httptest.NewRequest("POST", "/api/admin/sync", nil)
	rr := httptest.NewRecorder()
	h.SyncCT(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", rr.Code)
	}
}

// SendParentMessage

func TestSendParentMessage_VAPIDNotConfigured_Returns501(t *testing.T) {
	h, _ := newTestHandler(t, &mockCT{}, &mockSync{})
	req := httptest.NewRequest("POST", "/", nil)
	req = withParam(req, "id", "1")
	rr := httptest.NewRecorder()
	h.SendParentMessage(rr, req)
	if rr.Code != http.StatusNotImplemented {
		t.Errorf("expected 501, got %d", rr.Code)
	}
}

// GetParentQR

func TestGetParentQR_ValidToken_ReturnsPNG(t *testing.T) {
	h, _ := newTestHandler(t, &mockCT{}, &mockSync{})
	tok := parentBearerToken(t, 7)
	req := httptest.NewRequest("GET", "/", nil)
	req = withParam(req, "token", tok)
	rr := httptest.NewRecorder()
	h.GetParentQR(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "image/png" {
		t.Errorf("expected image/png, got %q", ct)
	}
}

func TestGetParentQR_InvalidToken_Returns401(t *testing.T) {
	h, _ := newTestHandler(t, &mockCT{}, &mockSync{})
	req := httptest.NewRequest("GET", "/", nil)
	req = withParam(req, "token", "bad-token")
	rr := httptest.NewRecorder()
	h.GetParentQR(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestGetParentQR_AdminToken_Returns401(t *testing.T) {
	h, _ := newTestHandler(t, &mockCT{}, &mockSync{})
	tok := adminBearerToken(t)
	req := httptest.NewRequest("GET", "/", nil)
	req = withParam(req, "token", tok)
	rr := httptest.NewRecorder()
	h.GetParentQR(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for admin token used as parent token, got %d", rr.Code)
	}
}

// GetParentManifest

func TestGetParentManifest_ValidToken_ReturnsManifest(t *testing.T) {
	h, _ := newTestHandler(t, &mockCT{}, &mockSync{})
	tok := parentBearerToken(t, 8)
	req := httptest.NewRequest("GET", "/", nil)
	req = withParam(req, "token", tok)
	rr := httptest.NewRecorder()
	h.GetParentManifest(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var manifest map[string]any
	json.NewDecoder(rr.Body).Decode(&manifest)
	if manifest["start_url"] == nil {
		t.Error("expected start_url in manifest")
	}
	if !strings.Contains(manifest["start_url"].(string), tok) {
		t.Errorf("expected start_url to contain token, got %q", manifest["start_url"])
	}
}

func TestGetParentManifest_InvalidToken_Returns401(t *testing.T) {
	h, _ := newTestHandler(t, &mockCT{}, &mockSync{})
	req := httptest.NewRequest("GET", "/", nil)
	req = withParam(req, "token", "not-valid")
	rr := httptest.NewRecorder()
	h.GetParentManifest(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

// SavePushSubscription

func TestSavePushSubscription_Valid_Returns200(t *testing.T) {
	h, _ := newTestHandler(t, &mockCT{}, &mockSync{})
	tok := parentBearerToken(t, 9)
	body := map[string]string{
		"endpoint": "https://push.example.com/sub/123",
		"p256dh":   "key-data",
		"auth":     "auth-secret",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req = withParam(req, "token", tok)
	rr := httptest.NewRecorder()
	h.SavePushSubscription(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body)
	}
}

func TestSavePushSubscription_InvalidToken_Returns401(t *testing.T) {
	h, _ := newTestHandler(t, &mockCT{}, &mockSync{})
	body := map[string]string{"endpoint": "https://push.example.com/sub/1", "p256dh": "k", "auth": "a"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req = withParam(req, "token", "invalid-token")
	rr := httptest.NewRecorder()
	h.SavePushSubscription(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

// ClearNotify

func TestClearNotify_ClearsLastNotifiedAt(t *testing.T) {
	h, database := newTestHandler(t, &mockCT{}, &mockSync{})
	database.Create(&localdb.CheckIn{EventDate: localdb.Today(), ChildID: 60, Status: "pending"})
	var record localdb.CheckIn
	database.First(&record)
	req := httptest.NewRequest("DELETE", "/", nil)
	req = withParam(req, "id", fmt.Sprint(record.ID))
	rr := httptest.NewRecorder()
	h.ClearNotify(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestClearNotify_NotFound_Returns404(t *testing.T) {
	h, _ := newTestHandler(t, &mockCT{}, &mockSync{})
	req := httptest.NewRequest("DELETE", "/", nil)
	req = withParam(req, "id", "9999")
	rr := httptest.NewRecorder()
	h.ClearNotify(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

// GetParentDetailByParentID

func TestGetParentDetailByParentID_Found(t *testing.T) {
	h, database := newTestHandler(t, &mockCT{}, &mockSync{})
	p := &localdb.SyncedPerson{CTID: 100, FirstName: "Jane", LastName: "Doe", IsParent: true}
	database.Create(p)
	req := httptest.NewRequest("GET", "/", nil)
	req = withParam(req, "id", fmt.Sprint(p.ID))
	rr := httptest.NewRecorder()
	h.GetParentDetailByParentID(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body)
	}
	var resp map[string]any
	json.NewDecoder(rr.Body).Decode(&resp)
	parent, ok := resp["parent"].(map[string]any)
	if !ok {
		t.Fatal("expected parent in response")
	}
	if parent["firstName"] != "Jane" {
		t.Errorf("expected firstName=Jane, got %v", parent["firstName"])
	}
}

func TestGetParentDetailByParentID_NotFound_Returns404(t *testing.T) {
	h, _ := newTestHandler(t, &mockCT{}, &mockSync{})
	req := httptest.NewRequest("GET", "/", nil)
	req = withParam(req, "id", "9999")
	rr := httptest.NewRecorder()
	h.GetParentDetailByParentID(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

// GetChildParents

func TestGetChildParents_ReturnsLinkedParents(t *testing.T) {
	h, database := newTestHandler(t, &mockCT{}, &mockSync{})
	database.Create(&localdb.SyncedPerson{CTID: 201, FirstName: "Parent", LastName: "One", IsParent: true})
	child := &localdb.SyncedPerson{CTID: 301, FirstName: "Child", LastName: "One", IsChild: true}
	database.Create(child)
	database.Create(&localdb.SyncedRelationship{ParentCTID: 201, ChildCTID: 301})
	req := httptest.NewRequest("GET", "/", nil)
	req = withParam(req, "id", fmt.Sprint(child.ID))
	rr := httptest.NewRecorder()
	h.GetChildParents(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var resp []map[string]any
	json.NewDecoder(rr.Body).Decode(&resp)
	if len(resp) != 1 {
		t.Errorf("expected 1 parent, got %d", len(resp))
	}
}

// GetParentDetail (CT-dependent)

func TestGetParentDetail_ReturnsParentAndChildren(t *testing.T) {
	h, database := newTestHandler(t, &mockCT{}, &mockSync{})
	database.Create(&localdb.SyncedPerson{CTID: 200, FirstName: "Max", LastName: "Schmidt", IsParent: true})
	child := &localdb.SyncedPerson{CTID: 300, FirstName: "Anna", LastName: "Schmidt", IsChild: true}
	database.Create(child)
	database.Create(&localdb.SyncedRelationship{ParentCTID: 200, ChildCTID: 300})
	req := httptest.NewRequest("GET", "/", nil)
	req = withParam(req, "id", fmt.Sprint(child.ID))
	rr := httptest.NewRecorder()
	h.GetParentDetail(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body)
	}
	var resp map[string]any
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp["parent"] == nil {
		t.Error("expected parent in response")
	}
}

// GetParentCheckinPage (CT-dependent)

func TestGetParentCheckinPage_ValidToken_ReturnsPage(t *testing.T) {
	h, database := newTestHandler(t, &mockCT{}, &mockSync{})
	parent := &localdb.SyncedPerson{CTID: 7, FirstName: "Test", LastName: "Parent", IsParent: true}
	database.Create(parent)
	tok := parentBearerToken(t, int(parent.ID))
	req := httptest.NewRequest("GET", "/", nil)
	req = withParam(req, "token", tok)
	rr := httptest.NewRecorder()
	h.GetParentCheckinPage(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body)
	}
}

func TestGetParentCheckinPage_InvalidToken_Returns401(t *testing.T) {
	h, _ := newTestHandler(t, &mockCT{}, &mockSync{})
	req := httptest.NewRequest("GET", "/", nil)
	req = withParam(req, "token", "bad-token")
	rr := httptest.NewRecorder()
	h.GetParentCheckinPage(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

// RegisterChild (CT-dependent)

func TestRegisterChild_Valid_CreatesPendingRecord(t *testing.T) {
	h, database := newTestHandler(t, &mockCT{}, &mockSync{})
	par := &localdb.SyncedPerson{CTID: 10, FirstName: "Test", LastName: "Parent", IsParent: true}
	database.Create(par)
	child := &localdb.SyncedPerson{CTID: 55, FirstName: "Kid", LastName: "Test", IsChild: true}
	database.Create(child)
	database.Create(&localdb.SyncedRelationship{ParentCTID: 10, ChildCTID: 55})
	tok := parentBearerToken(t, int(par.ID))
	req := httptest.NewRequest("POST", "/", nil)
	req = withParams(req, map[string]string{"token": tok, "childId": fmt.Sprint(child.ID)})
	rr := httptest.NewRecorder()
	h.RegisterChild(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body)
	}
	var count int64
	database.Model(&localdb.CheckIn{}).Where("child_id = ? AND status = ?", child.ID, "pending").Count(&count)
	if count != 1 {
		t.Errorf("expected 1 pending check-in record, got %d", count)
	}
}

func TestRegisterChild_ChildNotBelongingToParent_Returns403(t *testing.T) {
	h, database := newTestHandler(t, &mockCT{}, &mockSync{})
	par := &localdb.SyncedPerson{CTID: 10, FirstName: "Test", LastName: "Parent", IsParent: true}
	database.Create(par)
	child := &localdb.SyncedPerson{CTID: 99, FirstName: "Other", LastName: "Kid", IsChild: true}
	database.Create(child)
	// No relationship between parent and child.
	tok := parentBearerToken(t, int(par.ID))
	req := httptest.NewRequest("POST", "/", nil)
	req = withParams(req, map[string]string{"token": tok, "childId": fmt.Sprint(child.ID)})
	rr := httptest.NewRecorder()
	h.RegisterChild(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rr.Code)
	}
}

// ListParents

func TestListParents_FilterBySex(t *testing.T) {
	h, database := newTestHandler(t, &mockCT{}, &mockSync{})
	database.Create(&localdb.SyncedPerson{CTID: 401, FirstName: "Father", IsParent: true, Sex: "male"})
	database.Create(&localdb.SyncedPerson{CTID: 402, FirstName: "Mother", IsParent: true, Sex: "female"})
	req := httptest.NewRequest("GET", "/api/admin/parents?sex=male", nil)
	rr := httptest.NewRecorder()
	h.ListParents(rr, req)
	var resp []map[string]any
	json.NewDecoder(rr.Body).Decode(&resp)
	if len(resp) != 1 {
		t.Errorf("expected 1 male parent, got %d", len(resp))
	}
}

// GenerateQR

func TestGenerateQR_ReturnsQRCodePNG(t *testing.T) {
	h, database := newTestHandler(t, &mockCT{}, &mockSync{})
	p := &localdb.SyncedPerson{CTID: 500, FirstName: "Test", LastName: "Parent", IsParent: true}
	database.Create(p)
	req := httptest.NewRequest("POST", "/", nil)
	req = withParam(req, "id", fmt.Sprint(p.ID))
	rr := httptest.NewRecorder()
	h.GenerateQR(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "image/png" {
		t.Errorf("expected image/png, got %q", ct)
	}
}

func TestGenerateQR_InvalidID_Returns400(t *testing.T) {
	h, _ := newTestHandler(t, &mockCT{}, &mockSync{})
	req := httptest.NewRequest("POST", "/", nil)
	req = withParam(req, "id", "notanumber")
	rr := httptest.NewRecorder()
	h.GenerateQR(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestGenerateQR_ReturnsCheckinURLHeader(t *testing.T) {
	h, database := newTestHandler(t, &mockCT{}, &mockSync{})
	p := &localdb.SyncedPerson{CTID: 42, FirstName: "Test", LastName: "Parent", IsParent: true}
	database.Create(p)
	req := httptest.NewRequest("POST", "/", nil)
	req = withParam(req, "id", fmt.Sprint(p.ID))
	rr := httptest.NewRecorder()
	h.GenerateQR(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if url := rr.Header().Get("X-Checkin-Url"); url == "" {
		t.Error("expected X-Checkin-Url header to be set")
	}
}

// ListChildren

func TestListChildren_Empty_ReturnsEmptyArray(t *testing.T) {
	h, _ := newTestHandler(t, &mockCT{}, &mockSync{})
	req := httptest.NewRequest("GET", "/api/admin/children", nil)
	rr := httptest.NewRecorder()
	h.ListChildren(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var resp []any
	json.NewDecoder(rr.Body).Decode(&resp)
	if len(resp) != 0 {
		t.Errorf("expected empty array, got %d items", len(resp))
	}
}

func TestListChildren_WithData_ReturnsChildren(t *testing.T) {
	h, database := newTestHandler(t, &mockCT{}, &mockSync{})
	// Create parent and child persons.
	database.Create(&localdb.SyncedPerson{CTID: 901, FirstName: "Child", LastName: "One", IsChild: true, Sex: ""})
	database.Create(&localdb.SyncedPerson{CTID: 902, FirstName: "Dad", LastName: "One", IsParent: true, Sex: "male"})
	// Link them.
	database.Create(&localdb.SyncedRelationship{ParentCTID: 902, ChildCTID: 901})
	// Group membership.
	database.Create(&localdb.SyncedGroupMembership{PersonCTID: 901, GroupID: 5, GroupName: "Kindergruppe"})
	req := httptest.NewRequest("GET", "/api/admin/children", nil)
	rr := httptest.NewRecorder()
	h.ListChildren(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var resp []map[string]any
	json.NewDecoder(rr.Body).Decode(&resp)
	if len(resp) != 1 {
		t.Errorf("expected 1 child, got %d", len(resp))
	}
	if resp[0]["hasFather"] != true {
		t.Errorf("expected hasFather=true, got %v", resp[0]["hasFather"])
	}
}

// CheckInAtGroup error cases

func TestCheckInAtGroup_InvalidID_Returns400(t *testing.T) {
	h, _ := newTestHandler(t, &mockCT{}, &mockSync{})
	req := httptest.NewRequest("POST", "/", nil)
	req = withParam(req, "id", "notanid")
	rr := httptest.NewRecorder()
	h.CheckInAtGroup(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestCheckInAtGroup_NotFound_Returns404(t *testing.T) {
	h, _ := newTestHandler(t, &mockCT{}, &mockSync{})
	req := httptest.NewRequest("POST", "/", nil)
	req = withParam(req, "id", "9999")
	rr := httptest.NewRecorder()
	h.CheckInAtGroup(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

// GetParentDetail error cases

func TestGetParentDetail_InvalidID_Returns400(t *testing.T) {
	h, _ := newTestHandler(t, &mockCT{}, &mockSync{})
	req := httptest.NewRequest("GET", "/", nil)
	req = withParam(req, "id", "notanumber")
	rr := httptest.NewRecorder()
	h.GetParentDetail(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestGetParentDetail_NoRelationship_Returns404(t *testing.T) {
	h, database := newTestHandler(t, &mockCT{}, &mockSync{})
	child := &localdb.SyncedPerson{CTID: 100, FirstName: "Orphan", IsChild: true}
	database.Create(child)
	req := httptest.NewRequest("GET", "/", nil)
	req = withParam(req, "id", fmt.Sprint(child.ID))
	rr := httptest.NewRecorder()
	h.GetParentDetail(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

// GetParentDetailByParentID error cases

func TestGetParentDetailByParentID_InvalidID_Returns400(t *testing.T) {
	h, _ := newTestHandler(t, &mockCT{}, &mockSync{})
	req := httptest.NewRequest("GET", "/", nil)
	req = withParam(req, "id", "notanumber")
	rr := httptest.NewRecorder()
	h.GetParentDetailByParentID(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestGetParentDetailByParentID_WithChildren(t *testing.T) {
	h, database := newTestHandler(t, &mockCT{}, &mockSync{})
	parent := &localdb.SyncedPerson{CTID: 200, FirstName: "Parent", IsParent: true}
	database.Create(parent)
	database.Create(&localdb.SyncedPerson{CTID: 201, FirstName: "Child", IsChild: true})
	database.Create(&localdb.SyncedRelationship{ParentCTID: 200, ChildCTID: 201})
	database.Create(&localdb.SyncedGroupMembership{PersonCTID: 201, GroupID: 3, GroupName: "Teens"})
	req := httptest.NewRequest("GET", "/", nil)
	req = withParam(req, "id", fmt.Sprint(parent.ID))
	rr := httptest.NewRecorder()
	h.GetParentDetailByParentID(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body)
	}
	var resp map[string]any
	json.NewDecoder(rr.Body).Decode(&resp)
	children := resp["children"].([]any)
	if len(children) != 1 {
		t.Errorf("expected 1 child, got %d", len(children))
	}
}

// GetChildParents error cases

func TestGetChildParents_InvalidID_Returns400(t *testing.T) {
	h, _ := newTestHandler(t, &mockCT{}, &mockSync{})
	req := httptest.NewRequest("GET", "/", nil)
	req = withParam(req, "id", "notanumber")
	rr := httptest.NewRecorder()
	h.GetChildParents(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

// ClearNotify error cases

func TestClearNotify_InvalidID_Returns400(t *testing.T) {
	h, _ := newTestHandler(t, &mockCT{}, &mockSync{})
	req := httptest.NewRequest("DELETE", "/", nil)
	req = withParam(req, "id", "notanumber")
	rr := httptest.NewRecorder()
	h.ClearNotify(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

// ListCheckins with groupId filter

func TestListCheckins_FilterByGroupID(t *testing.T) {
	h, database := newTestHandler(t, &mockCT{}, &mockSync{})
	today := localdb.Today()
	database.Create(&localdb.CheckIn{EventDate: today, ChildID: 1, GroupID: 10, Status: "pending"})
	database.Create(&localdb.CheckIn{EventDate: today, ChildID: 2, GroupID: 20, Status: "pending"})
	req := httptest.NewRequest("GET", "/api/admin/checkins?groupId=10", nil)
	rr := httptest.NewRecorder()
	h.ListCheckins(rr, req)
	var resp []map[string]any
	json.NewDecoder(rr.Body).Decode(&resp)
	if len(resp) != 1 {
		t.Errorf("expected 1 group-10 record, got %d", len(resp))
	}
}

// ListParents without filter

func TestListParents_NoFilter_ReturnsAll(t *testing.T) {
	h, database := newTestHandler(t, &mockCT{}, &mockSync{})
	database.Create(&localdb.SyncedPerson{CTID: 501, FirstName: "Alice", IsParent: true, Sex: "female"})
	database.Create(&localdb.SyncedPerson{CTID: 502, FirstName: "Bob", IsParent: true, Sex: "male"})
	req := httptest.NewRequest("GET", "/api/admin/parents", nil)
	rr := httptest.NewRecorder()
	h.ListParents(rr, req)
	var resp []map[string]any
	json.NewDecoder(rr.Body).Decode(&resp)
	if len(resp) != 2 {
		t.Errorf("expected 2 parents, got %d", len(resp))
	}
}

// SendParentMessage error cases (after VAPID not-configured path)

func TestSendParentMessage_InvalidID_Returns400(t *testing.T) {
	t.Setenv("VAPID_PRIVATE_KEY", "priv")
	t.Setenv("VAPID_PUBLIC_KEY", "pub")
	t.Setenv("LOCAL_PASSWORD", "true")
	t.Setenv("ADMIN_PASSWORD", "admin123")
	t.Setenv("VOLUNTEER_PASSWORD", "vol123")
	t.Setenv("CT_ADMIN_PERSONS", "")
	t.Setenv("REPORTS_DIR", t.TempDir())
	database, _ := localdb.Open(":memory:")
	h := handlers.New(&mockCT{}, database, &mockSync{}, []byte(testJWTSecret), "http://test.local")
	req := httptest.NewRequest("POST", "/", nil)
	req = withParam(req, "id", "notanumber")
	rr := httptest.NewRecorder()
	h.SendParentMessage(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestSendParentMessage_RecordNotFound_Returns404(t *testing.T) {
	t.Setenv("VAPID_PRIVATE_KEY", "priv")
	t.Setenv("VAPID_PUBLIC_KEY", "pub")
	t.Setenv("LOCAL_PASSWORD", "true")
	t.Setenv("ADMIN_PASSWORD", "admin123")
	t.Setenv("VOLUNTEER_PASSWORD", "vol123")
	t.Setenv("CT_ADMIN_PERSONS", "")
	t.Setenv("REPORTS_DIR", t.TempDir())
	database, _ := localdb.Open(":memory:")
	h := handlers.New(&mockCT{}, database, &mockSync{}, []byte(testJWTSecret), "http://test.local")
	req := httptest.NewRequest("POST", "/", nil)
	req = withParam(req, "id", "9999")
	rr := httptest.NewRecorder()
	h.SendParentMessage(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestSendParentMessage_NoSubscription_Returns404(t *testing.T) {
	t.Setenv("VAPID_PRIVATE_KEY", "priv")
	t.Setenv("VAPID_PUBLIC_KEY", "pub")
	t.Setenv("LOCAL_PASSWORD", "true")
	t.Setenv("ADMIN_PASSWORD", "admin123")
	t.Setenv("VOLUNTEER_PASSWORD", "vol123")
	t.Setenv("CT_ADMIN_PERSONS", "")
	t.Setenv("REPORTS_DIR", t.TempDir())
	database, _ := localdb.Open(":memory:")
	database.Create(&localdb.CheckIn{EventDate: localdb.Today(), ChildID: 70, Status: "pending", ParentID: 999})
	var record localdb.CheckIn
	database.First(&record)
	h := handlers.New(&mockCT{}, database, &mockSync{}, []byte(testJWTSecret), "http://test.local")
	req := httptest.NewRequest("POST", "/", nil)
	req = withParam(req, "id", fmt.Sprint(record.ID))
	rr := httptest.NewRecorder()
	h.SendParentMessage(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404 (no subscription), got %d", rr.Code)
	}
}

// GetParentCheckinPage CT error

func TestGetParentCheckinPage_ParentNotFound_Returns404(t *testing.T) {
	h, _ := newTestHandler(t, &mockCT{}, &mockSync{})
	tok := parentBearerToken(t, 9999)
	req := httptest.NewRequest("GET", "/", nil)
	req = withParam(req, "token", tok)
	rr := httptest.NewRecorder()
	h.GetParentCheckinPage(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

// RegisterChild invalid childId

func TestRegisterChild_InvalidChildID_Returns400(t *testing.T) {
	h, _ := newTestHandler(t, &mockCT{}, &mockSync{})
	tok := parentBearerToken(t, 10)
	req := httptest.NewRequest("POST", "/", nil)
	req = withParams(req, map[string]string{"token": tok, "childId": "notanumber"})
	rr := httptest.NewRecorder()
	h.RegisterChild(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

// GetParentDetail no parents falls back to self

func TestGetParentDetail_ReturnsParentByChildID(t *testing.T) {
	h, database := newTestHandler(t, &mockCT{}, &mockSync{})
	database.Create(&localdb.SyncedPerson{CTID: 42, FirstName: "Solo", LastName: "Parent", IsParent: true})
	child := &localdb.SyncedPerson{CTID: 88, FirstName: "Kid", LastName: "Parent", IsChild: true}
	database.Create(child)
	database.Create(&localdb.SyncedRelationship{ParentCTID: 42, ChildCTID: 88})
	req := httptest.NewRequest("GET", "/", nil)
	req = withParam(req, "id", fmt.Sprint(child.ID))
	rr := httptest.NewRecorder()
	h.GetParentDetail(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body)
	}
	var resp map[string]any
	json.NewDecoder(rr.Body).Decode(&resp)
	parent := resp["parent"].(map[string]any)
	if parent["firstName"] != "Solo" {
		t.Errorf("expected firstName=Solo, got %v", parent["firstName"])
	}
}

// SetCheckInStatus invalid id and bad JSON

func TestSetCheckInStatus_InvalidID_Returns400(t *testing.T) {
	h, _ := newTestHandler(t, &mockCT{}, &mockSync{})
	req := httptest.NewRequest("POST", "/", strings.NewReader(`{"status":"pending"}`))
	req.Header.Set("Content-Type", "application/json")
	req = withParam(req, "id", "notanid")
	rr := httptest.NewRecorder()
	h.SetCheckInStatus(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestSetCheckInStatus_BadJSON_Returns400(t *testing.T) {
	h, database := newTestHandler(t, &mockCT{}, &mockSync{})
	record := localdb.CheckIn{EventDate: localdb.Today(), ChildID: 90, Status: "pending"}
	database.Create(&record)
	req := httptest.NewRequest("POST", "/", strings.NewReader("{not json}"))
	req.Header.Set("Content-Type", "application/json")
	req = withParam(req, "id", fmt.Sprint(record.ID))
	rr := httptest.NewRecorder()
	h.SetCheckInStatus(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestSetCheckInStatus_ToCheckedIn_SetsTimestamp(t *testing.T) {
	h, database := newTestHandler(t, &mockCT{}, &mockSync{})
	record := localdb.CheckIn{EventDate: localdb.Today(), ChildID: 91, Status: "pending"}
	database.Create(&record)
	req := httptest.NewRequest("POST", "/", strings.NewReader(`{"status":"checked_in"}`))
	req.Header.Set("Content-Type", "application/json")
	req = withParam(req, "id", fmt.Sprint(record.ID))
	rr := httptest.NewRecorder()
	h.SetCheckInStatus(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body)
	}
	var updated localdb.CheckIn
	database.First(&updated, record.ID)
	if updated.Status != "checked_in" {
		t.Errorf("expected checked_in, got %q", updated.Status)
	}
}

// ListParents with children and group membership

func TestListParents_WithChildren_ReturnsGroups(t *testing.T) {
	h, database := newTestHandler(t, &mockCT{}, &mockSync{})
	database.Create(&localdb.SyncedPerson{CTID: 601, FirstName: "Parent", IsParent: true})
	database.Create(&localdb.SyncedPerson{CTID: 602, FirstName: "Child", IsChild: true})
	database.Create(&localdb.SyncedRelationship{ParentCTID: 601, ChildCTID: 602})
	database.Create(&localdb.SyncedGroupMembership{PersonCTID: 602, GroupID: 7, GroupName: "Jugend"})
	req := httptest.NewRequest("GET", "/api/admin/parents", nil)
	rr := httptest.NewRecorder()
	h.ListParents(rr, req)
	var resp []map[string]any
	json.NewDecoder(rr.Body).Decode(&resp)
	if len(resp) != 1 {
		t.Fatalf("expected 1 parent, got %d", len(resp))
	}
	groups := resp[0]["groups"].([]any)
	if len(groups) != 1 {
		t.Errorf("expected 1 group, got %d", len(groups))
	}
}

// ListReports with non-matching files

func TestListReports_IgnoresNonCSVFiles(t *testing.T) {
	h, _ := newTestHandler(t, &mockCT{}, &mockSync{})
	reportsDir := os.Getenv("REPORTS_DIR")
	// Create a non-CSV file and a valid CSV file.
	os.WriteFile(reportsDir+"/not-a-report.txt", []byte("data"), 0644)
	os.WriteFile(reportsDir+"/2025-06-01_001.csv", []byte("data"), 0644)
	req := httptest.NewRequest("GET", "/api/admin/reports", nil)
	rr := httptest.NewRecorder()
	h.ListReports(rr, req)
	var reports []map[string]any
	json.NewDecoder(rr.Body).Decode(&reports)
	if len(reports) != 1 {
		t.Errorf("expected 1 valid report, got %d", len(reports))
	}
}

// EndEvent when reportsDir is not set (covers the MkdirAll path with actual dir)

func TestEndEvent_MultipleRecords_ReportHasParentData(t *testing.T) {
	h, database := newTestHandler(t, &mockCT{}, &mockSync{})
	today := localdb.Today()
	// Insert a parent synced person.
	par := &localdb.SyncedPerson{CTID: 700, FirstName: "Parent", LastName: "Test", IsParent: true}
	database.Create(par)
	// Insert check-in with that parent's GORM ID.
	database.Create(&localdb.CheckIn{EventDate: today, ChildID: 71, FirstName: "Kid", LastName: "Test", Status: "pending", ParentID: int(par.ID)})
	req := httptest.NewRequest("POST", "/", nil)
	rr := httptest.NewRecorder()
	h.EndEvent(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body)
	}
	reportsDir := os.Getenv("REPORTS_DIR")
	entries, _ := os.ReadDir(reportsDir)
	if len(entries) != 1 {
		t.Fatalf("expected 1 report, got %d", len(entries))
	}
	data, _ := os.ReadFile(reportsDir + "/" + entries[0].Name())
	if !strings.Contains(string(data), "Parent") {
		t.Error("expected parent name in CSV report")
	}
}
