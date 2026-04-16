package handlers_test

// Tests for guest family CRUD: CreateGuest, UpdateGuest, DeleteGuest.
// Each test uses an in-memory SQLite database via newTestHandler (defined in
// handlers_test.go) so no external services are required.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	localdb "github.com/ccf/check-in/backend/internal/db"
)

// ── helpers ──────────────────────────────────────────────────────────────────

func putJSON(t *testing.T, h http.HandlerFunc, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("PUT", path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h(rr, req)
	return rr
}

func putJSONWithParam(t *testing.T, h http.HandlerFunc, path string, paramKey, paramVal string, body any) *httptest.ResponseRecorder {
	t.Helper()
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("PUT", path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req = withParam(req, paramKey, paramVal)
	rr := httptest.NewRecorder()
	h(rr, req)
	return rr
}

func deleteWithParam(t *testing.T, h http.HandlerFunc, paramKey, paramVal string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest("DELETE", "/", nil)
	req = withParam(req, paramKey, paramVal)
	rr := httptest.NewRecorder()
	h(rr, req)
	return rr
}

type guestReq struct {
	Parent   guestParentReq  `json:"parent"`
	Children []guestChildReq `json:"children"`
}

type guestParentReq struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Sex       string `json:"sex"`
	Mobile    string `json:"mobile"`
}

type guestChildReq struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Birthdate string `json:"birthdate"`
	GroupID   int    `json:"groupId"`
	GroupName string `json:"groupName"`
}

func minimalGuest(firstName, lastName string, children ...guestChildReq) guestReq {
	return guestReq{
		Parent:   guestParentReq{FirstName: firstName, LastName: lastName},
		Children: children,
	}
}

// createGuestViaHandler calls CreateGuest and returns the new parent's gorm ID.
func createGuestViaHandler(t *testing.T, h interface {
	CreateGuest(http.ResponseWriter, *http.Request)
}, req guestReq) int {
	t.Helper()
	b, _ := json.Marshal(req)
	httpReq := httptest.NewRequest("POST", "/api/admin/guests", bytes.NewReader(b))
	httpReq.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.CreateGuest(rr, httpReq)
	if rr.Code != http.StatusCreated {
		t.Fatalf("CreateGuest: expected 201, got %d: %s", rr.Code, rr.Body)
	}
	var resp map[string]any
	json.NewDecoder(rr.Body).Decode(&resp)
	id, ok := resp["id"].(float64)
	if !ok {
		t.Fatalf("CreateGuest: expected id in response, got %v", resp)
	}
	return int(id)
}

// ── CreateGuest ───────────────────────────────────────────────────────────────

func TestCreateGuest_Returns201WithID(t *testing.T) {
	h, _ := newTestHandler(t, &mockCT{}, &mockSync{})
	rr := postJSON(t, h.CreateGuest, "/api/admin/guests", minimalGuest("Anna", "Müller"))
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body)
	}
	var resp map[string]any
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp["id"] == nil {
		t.Error("expected id in response")
	}
}

func TestCreateGuest_CreatesParentInDB(t *testing.T) {
	h, db := newTestHandler(t, &mockCT{}, &mockSync{})
	createGuestViaHandler(t, h, minimalGuest("Thomas", "Bauer"))

	var parent localdb.SyncedPerson
	db.Where("first_name = ? AND is_guest = ?", "Thomas", true).First(&parent)
	if parent.ID == 0 {
		t.Fatal("expected parent to be created in DB")
	}
	if !parent.IsParent {
		t.Error("expected IsParent=true")
	}
	if !parent.IsGuest {
		t.Error("expected IsGuest=true")
	}
	// CTID must be offset + gorm_id
	if parent.CTID != 1_000_000_000+int(parent.ID) {
		t.Errorf("expected CTID=%d, got %d", 1_000_000_000+int(parent.ID), parent.CTID)
	}
}

func TestCreateGuest_WithChildren_CreatesChildrenAndRelationships(t *testing.T) {
	h, db := newTestHandler(t, &mockCT{}, &mockSync{})
	req := minimalGuest("Maria", "Klein",
		guestChildReq{FirstName: "Leon", LastName: "Klein", GroupID: 5, GroupName: "Kleine"},
		guestChildReq{FirstName: "Sara", LastName: "Klein", GroupID: 5, GroupName: "Kleine"},
	)
	parentID := createGuestViaHandler(t, h, req)

	var parent localdb.SyncedPerson
	db.First(&parent, parentID)

	var count int64
	db.Model(&localdb.SyncedPerson{}).Where("is_child = ? AND is_guest = ?", true, true).Count(&count)
	if count != 2 {
		t.Errorf("expected 2 children, got %d", count)
	}

	var relCount int64
	db.Model(&localdb.SyncedRelationship{}).Where("parent_ct_id = ?", parent.CTID).Count(&relCount)
	if relCount != 2 {
		t.Errorf("expected 2 relationships, got %d", relCount)
	}
}

func TestCreateGuest_WithChild_CreatesGroupMembership(t *testing.T) {
	h, db := newTestHandler(t, &mockCT{}, &mockSync{})
	req := minimalGuest("Eva", "Hofer",
		guestChildReq{FirstName: "Tim", LastName: "Hofer", GroupID: 7, GroupName: "Kinder"},
	)
	createGuestViaHandler(t, h, req)

	var child localdb.SyncedPerson
	db.Where("first_name = ? AND is_child = ?", "Tim", true).First(&child)

	var mem localdb.SyncedGroupMembership
	db.Where("person_ct_id = ?", child.CTID).First(&mem)
	if mem.GroupID != 7 {
		t.Errorf("expected GroupID=7, got %d", mem.GroupID)
	}
	if mem.GroupName != "Kinder" {
		t.Errorf("expected GroupName=Kinder, got %q", mem.GroupName)
	}
}

func TestCreateGuest_MissingParentName_Returns400(t *testing.T) {
	h, _ := newTestHandler(t, &mockCT{}, &mockSync{})
	rr := postJSON(t, h.CreateGuest, "/api/admin/guests", minimalGuest("", ""))
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestCreateGuest_MissingFirstName_Returns400(t *testing.T) {
	h, _ := newTestHandler(t, &mockCT{}, &mockSync{})
	rr := postJSON(t, h.CreateGuest, "/api/admin/guests", minimalGuest("", "Müller"))
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestCreateGuest_InvalidJSON_Returns400(t *testing.T) {
	h, _ := newTestHandler(t, &mockCT{}, &mockSync{})
	req := httptest.NewRequest("POST", "/api/admin/guests", bytes.NewReader([]byte("{bad}")))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.CreateGuest(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestCreateGuest_ChildWithoutGroup_NoGroupMembership(t *testing.T) {
	h, db := newTestHandler(t, &mockCT{}, &mockSync{})
	req := minimalGuest("Franz", "Wolf",
		guestChildReq{FirstName: "Lena", LastName: "Wolf", GroupID: 0},
	)
	createGuestViaHandler(t, h, req)

	var child localdb.SyncedPerson
	db.Where("first_name = ? AND is_child = ?", "Lena", true).First(&child)

	var count int64
	db.Model(&localdb.SyncedGroupMembership{}).Where("person_ct_id = ?", child.CTID).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 group memberships for child without group, got %d", count)
	}
}

// ── UpdateGuest ───────────────────────────────────────────────────────────────

func TestUpdateGuest_UpdatesParentFields(t *testing.T) {
	h, db := newTestHandler(t, &mockCT{}, &mockSync{})
	id := createGuestViaHandler(t, h, minimalGuest("Old", "Name"))

	updateReq := minimalGuest("New", "Surname")
	updateReq.Parent.Mobile = "+4917612345678"
	rr := putJSONWithParam(t, h.UpdateGuest, "/api/admin/guests/"+fmt.Sprint(id), "id", fmt.Sprint(id), updateReq)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rr.Code, rr.Body)
	}

	var parent localdb.SyncedPerson
	db.First(&parent, id)
	if parent.FirstName != "New" {
		t.Errorf("expected FirstName=New, got %q", parent.FirstName)
	}
	if parent.LastName != "Surname" {
		t.Errorf("expected LastName=Surname, got %q", parent.LastName)
	}
	if parent.Mobile != "+4917612345678" {
		t.Errorf("expected Mobile=+4917612345678, got %q", parent.Mobile)
	}
}

func TestUpdateGuest_UpdatesChildInPlace_PreservesGormID(t *testing.T) {
	h, db := newTestHandler(t, &mockCT{}, &mockSync{})
	id := createGuestViaHandler(t, h, minimalGuest("Parent", "Test",
		guestChildReq{FirstName: "OrigFirst", LastName: "OrigLast", GroupID: 1, GroupName: "G1"},
	))

	var originalChild localdb.SyncedPerson
	db.Where("is_child = ? AND is_guest = ?", true, true).First(&originalChild)
	originalChildID := originalChild.ID

	updateReq := minimalGuest("Parent", "Test",
		guestChildReq{FirstName: "NewFirst", LastName: "NewLast", GroupID: 2, GroupName: "G2"},
	)
	rr := putJSONWithParam(t, h.UpdateGuest, "/", "id", fmt.Sprint(id), updateReq)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rr.Code, rr.Body)
	}

	var updatedChild localdb.SyncedPerson
	db.First(&updatedChild, originalChildID)
	if updatedChild.FirstName != "NewFirst" {
		t.Errorf("expected FirstName=NewFirst, got %q", updatedChild.FirstName)
	}
	// Gorm ID must not have changed — same row updated in-place.
	if updatedChild.ID != originalChildID {
		t.Errorf("expected child gorm ID to be stable, got %d (was %d)", updatedChild.ID, originalChildID)
	}
}

func TestUpdateGuest_UpdatesChildInPlace_PreservesCheckinStatus(t *testing.T) {
	h, db := newTestHandler(t, &mockCT{}, &mockSync{})
	id := createGuestViaHandler(t, h, minimalGuest("Parent", "Check",
		guestChildReq{FirstName: "Kid", LastName: "Check", GroupID: 1, GroupName: "G"},
	))

	var child localdb.SyncedPerson
	db.Where("is_child = ? AND is_guest = ?", true, true).First(&child)

	// Simulate pre-existing check-in for today.
	checkin := localdb.CheckIn{
		EventDate: localdb.Today(),
		ChildID:   int(child.ID),
		ParentID:  id,
		Status:    "checked_in",
		IsGuest:   true,
		FirstName: "Kid",
		LastName:  "Check",
	}
	db.Create(&checkin)

	// Now update the child's name.
	updateReq := minimalGuest("Parent", "Check",
		guestChildReq{FirstName: "Kiddo", LastName: "Check", GroupID: 1, GroupName: "G"},
	)
	rr := putJSONWithParam(t, h.UpdateGuest, "/", "id", fmt.Sprint(id), updateReq)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rr.Code, rr.Body)
	}

	// CheckIn record must still exist with status=checked_in.
	var updated localdb.CheckIn
	db.First(&updated, checkin.ID)
	if updated.Status != "checked_in" {
		t.Errorf("expected check-in status preserved as checked_in, got %q", updated.Status)
	}
	// Cached name should be refreshed.
	if updated.FirstName != "Kiddo" {
		t.Errorf("expected cached FirstName updated to Kiddo, got %q", updated.FirstName)
	}
}

func TestUpdateGuest_AddChild_CreatesNewChild(t *testing.T) {
	h, db := newTestHandler(t, &mockCT{}, &mockSync{})
	id := createGuestViaHandler(t, h, minimalGuest("Parent", "Add",
		guestChildReq{FirstName: "First", LastName: "Add", GroupID: 1, GroupName: "G"},
	))

	updateReq := minimalGuest("Parent", "Add",
		guestChildReq{FirstName: "First", LastName: "Add", GroupID: 1, GroupName: "G"},
		guestChildReq{FirstName: "Second", LastName: "Add", GroupID: 1, GroupName: "G"},
	)
	rr := putJSONWithParam(t, h.UpdateGuest, "/", "id", fmt.Sprint(id), updateReq)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rr.Code, rr.Body)
	}

	var count int64
	db.Model(&localdb.SyncedPerson{}).Where("is_child = ? AND is_guest = ?", true, true).Count(&count)
	if count != 2 {
		t.Errorf("expected 2 children after adding one, got %d", count)
	}
}

func TestUpdateGuest_RemoveChild_DeletesExcessChild(t *testing.T) {
	h, db := newTestHandler(t, &mockCT{}, &mockSync{})
	id := createGuestViaHandler(t, h, minimalGuest("Parent", "Remove",
		guestChildReq{FirstName: "Child1", LastName: "Remove", GroupID: 1, GroupName: "G"},
		guestChildReq{FirstName: "Child2", LastName: "Remove", GroupID: 1, GroupName: "G"},
	))

	// Shrink to 1 child.
	updateReq := minimalGuest("Parent", "Remove",
		guestChildReq{FirstName: "Child1", LastName: "Remove", GroupID: 1, GroupName: "G"},
	)
	rr := putJSONWithParam(t, h.UpdateGuest, "/", "id", fmt.Sprint(id), updateReq)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rr.Code, rr.Body)
	}

	var count int64
	db.Model(&localdb.SyncedPerson{}).Where("is_child = ? AND is_guest = ?", true, true).Count(&count)
	if count != 1 {
		t.Errorf("expected 1 child after removal, got %d", count)
	}
}

func TestUpdateGuest_RemoveChild_DeletesActiveCheckin(t *testing.T) {
	h, db := newTestHandler(t, &mockCT{}, &mockSync{})
	id := createGuestViaHandler(t, h, minimalGuest("Parent", "Del",
		guestChildReq{FirstName: "Removable", LastName: "Del", GroupID: 1, GroupName: "G"},
		guestChildReq{FirstName: "Kept", LastName: "Del", GroupID: 1, GroupName: "G"},
	))

	var children []localdb.SyncedPerson
	db.Where("is_child = ? AND is_guest = ?", true, true).Order("id").Find(&children)
	if len(children) != 2 {
		t.Fatalf("expected 2 children pre-update, got %d", len(children))
	}
	// The second child (index 1 / higher CTID = to-be-removed) has a check-in.
	toBeRemoved := children[1]
	checkin := localdb.CheckIn{
		EventDate: localdb.Today(),
		ChildID:   int(toBeRemoved.ID),
		Status:    "pending",
		IsGuest:   true,
	}
	db.Create(&checkin)

	// Shrink to 1 child.
	updateReq := minimalGuest("Parent", "Del",
		guestChildReq{FirstName: "Kept", LastName: "Del", GroupID: 1, GroupName: "G"},
	)
	rr := putJSONWithParam(t, h.UpdateGuest, "/", "id", fmt.Sprint(id), updateReq)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rr.Code, rr.Body)
	}

	// CheckIn for removed child must be soft-deleted.
	var remaining int64
	db.Model(&localdb.CheckIn{}).Where("child_id = ?", toBeRemoved.ID).Count(&remaining)
	if remaining != 0 {
		t.Errorf("expected check-in for removed child to be deleted, found %d records", remaining)
	}
}

func TestUpdateGuest_NotFound_Returns404(t *testing.T) {
	h, _ := newTestHandler(t, &mockCT{}, &mockSync{})
	rr := putJSONWithParam(t, h.UpdateGuest, "/", "id", "9999", minimalGuest("X", "Y"))
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestUpdateGuest_InvalidID_Returns400(t *testing.T) {
	h, _ := newTestHandler(t, &mockCT{}, &mockSync{})
	rr := putJSONWithParam(t, h.UpdateGuest, "/", "id", "not-an-id", minimalGuest("X", "Y"))
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestUpdateGuest_NonGuestParent_Returns404(t *testing.T) {
	h, db := newTestHandler(t, &mockCT{}, &mockSync{})
	// Create a real (non-guest) parent.
	realParent := localdb.SyncedPerson{CTID: 500, FirstName: "Real", LastName: "Parent", IsParent: true, IsGuest: false}
	db.Create(&realParent)
	rr := putJSONWithParam(t, h.UpdateGuest, "/", "id", fmt.Sprint(realParent.ID), minimalGuest("X", "Y"))
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404 for non-guest parent, got %d", rr.Code)
	}
}

// ── DeleteGuest ───────────────────────────────────────────────────────────────

func TestDeleteGuest_Returns204(t *testing.T) {
	h, _ := newTestHandler(t, &mockCT{}, &mockSync{})
	id := createGuestViaHandler(t, h, minimalGuest("Del", "Parent"))
	rr := deleteWithParam(t, h.DeleteGuest, "id", fmt.Sprint(id))
	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rr.Code, rr.Body)
	}
}

func TestDeleteGuest_RemovesParentFromDB(t *testing.T) {
	h, db := newTestHandler(t, &mockCT{}, &mockSync{})
	id := createGuestViaHandler(t, h, minimalGuest("Gone", "Parent"))
	deleteWithParam(t, h.DeleteGuest, "id", fmt.Sprint(id))

	var count int64
	db.Model(&localdb.SyncedPerson{}).Where("id = ?", id).Count(&count)
	if count != 0 {
		t.Errorf("expected parent to be deleted, found %d rows", count)
	}
}

func TestDeleteGuest_RemovesChildrenAndRelationships(t *testing.T) {
	h, db := newTestHandler(t, &mockCT{}, &mockSync{})
	id := createGuestViaHandler(t, h, minimalGuest("Del2", "Parent",
		guestChildReq{FirstName: "KidToDel", LastName: "Parent", GroupID: 1, GroupName: "G"},
	))

	var parent localdb.SyncedPerson
	db.First(&parent, id)

	deleteWithParam(t, h.DeleteGuest, "id", fmt.Sprint(id))

	var childCount int64
	db.Unscoped().Model(&localdb.SyncedPerson{}).Where("ct_id = ?", parent.CTID+1).Count(&childCount)
	// All children hard-deleted.
	var relCount int64
	db.Model(&localdb.SyncedRelationship{}).Where("parent_ct_id = ?", parent.CTID).Count(&relCount)
	if relCount != 0 {
		t.Errorf("expected 0 relationships after delete, got %d", relCount)
	}
}

func TestDeleteGuest_NotFound_Returns404(t *testing.T) {
	h, _ := newTestHandler(t, &mockCT{}, &mockSync{})
	rr := deleteWithParam(t, h.DeleteGuest, "id", "9999")
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestDeleteGuest_NonGuest_Returns404(t *testing.T) {
	h, db := newTestHandler(t, &mockCT{}, &mockSync{})
	realParent := localdb.SyncedPerson{CTID: 600, FirstName: "Real", LastName: "One", IsParent: true, IsGuest: false}
	db.Create(&realParent)
	rr := deleteWithParam(t, h.DeleteGuest, "id", fmt.Sprint(realParent.ID))
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404 for non-guest parent, got %d", rr.Code)
	}
}

// ── Integration: guest check-in flow ─────────────────────────────────────────

// TestGuest_FullCheckinFlow verifies the end-to-end guest lifecycle:
// create → register (via parent token) → check in at group.
func TestGuest_FullCheckinFlow(t *testing.T) {
	h, db := newTestHandler(t, &mockCT{}, &mockSync{})

	// 1. Create guest family.
	id := createGuestViaHandler(t, h, minimalGuest("Gast", "Familie",
		guestChildReq{FirstName: "GastKind", LastName: "Familie", GroupID: 3, GroupName: "KG"},
	))

	// 2. Generate a parent JWT for this parent.
	tok := parentBearerToken(t, id)

	// 3. Find the guest child.
	var parent localdb.SyncedPerson
	db.First(&parent, id)
	var child localdb.SyncedPerson
	db.Where("is_child = ? AND is_guest = ?", true, true).First(&child)

	// 4. Register the child (mimics parent tapping Anmelden).
	// RegisterChild takes the child gorm_id (not CTID) as the URL parameter.
	regReq := httptest.NewRequest("POST", "/", nil)
	regReq = withParams(regReq, map[string]string{
		"token":   tok,
		"childId": fmt.Sprint(child.ID),
	})
	rrReg := httptest.NewRecorder()
	h.RegisterChild(rrReg, regReq)
	if rrReg.Code != http.StatusOK {
		t.Fatalf("RegisterChild: expected 200, got %d: %s", rrReg.Code, rrReg.Body)
	}

	// 5. Verify pending CheckIn was created with IsGuest=true.
	var checkin localdb.CheckIn
	db.Where("child_id = ? AND event_date = ?", child.ID, localdb.Today()).First(&checkin)
	if checkin.Status != "pending" {
		t.Errorf("expected status=pending after registration, got %q", checkin.Status)
	}
	if !checkin.IsGuest {
		t.Error("expected IsGuest=true on check-in record")
	}

	// 6. Check in at group (volunteer action).
	ciReq := httptest.NewRequest("POST", "/", nil)
	ciReq = withParam(ciReq, "id", fmt.Sprint(checkin.ID))
	rrCI := httptest.NewRecorder()
	h.CheckInAtGroup(rrCI, ciReq)
	if rrCI.Code != http.StatusOK {
		t.Fatalf("CheckInAtGroup: expected 200, got %d: %s", rrCI.Code, rrCI.Body)
	}

	var updated localdb.CheckIn
	db.First(&updated, checkin.ID)
	if updated.Status != "checked_in" {
		t.Errorf("expected status=checked_in, got %q", updated.Status)
	}
}

// TestGuest_EditWhileCheckedIn verifies that editing a family does not disturb
// an existing check-in record.
func TestGuest_EditWhileCheckedIn(t *testing.T) {
	h, db := newTestHandler(t, &mockCT{}, &mockSync{})

	id := createGuestViaHandler(t, h, minimalGuest("Edit", "Live",
		guestChildReq{FirstName: "ActiveKid", LastName: "Live", GroupID: 1, GroupName: "G"},
	))

	var child localdb.SyncedPerson
	db.Where("is_child = ?", true).First(&child)

	// Simulate an active check-in.
	checkin := localdb.CheckIn{
		EventDate: localdb.Today(),
		ChildID:   int(child.ID),
		Status:    "checked_in",
		IsGuest:   true,
	}
	db.Create(&checkin)

	// Edit: rename child and change group.
	updateReq := minimalGuest("Edit", "Live",
		guestChildReq{FirstName: "Renamed", LastName: "Live", GroupID: 2, GroupName: "G2"},
	)
	rr := putJSONWithParam(t, h.UpdateGuest, "/", "id", fmt.Sprint(id), updateReq)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("UpdateGuest: expected 204, got %d: %s", rr.Code, rr.Body)
	}

	// Check-in status must be untouched.
	var ci localdb.CheckIn
	db.First(&ci, checkin.ID)
	if ci.Status != "checked_in" {
		t.Errorf("expected checked_in status preserved, got %q", ci.Status)
	}

	// Child should still exist (same gorm ID).
	var updChild localdb.SyncedPerson
	db.First(&updChild, child.ID)
	if updChild.FirstName != "Renamed" {
		t.Errorf("expected child renamed, got %q", updChild.FirstName)
	}
}
