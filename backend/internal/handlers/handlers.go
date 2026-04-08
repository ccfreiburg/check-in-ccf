package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/ccf/check-in/backend/internal/auth"
	"github.com/ccf/check-in/backend/internal/ct"
	"github.com/ccf/check-in/backend/internal/ctsync"
	localdb "github.com/ccf/check-in/backend/internal/db"
	"github.com/go-chi/chi/v5"
	qrcode "github.com/skip2/go-qrcode"
	"gorm.io/gorm"
)

type Handler struct {
	ct                 *ct.Client
	db                 *gorm.DB
	syncSvc            *ctsync.Service
	jwtSecret          []byte
	frontendBase       string
	adminPassword      string
	superAdminPassword string
}

func New(ctClient *ct.Client, database *gorm.DB, syncSvc *ctsync.Service, jwtSecret []byte, frontendBase string) *Handler {
	return &Handler{
		ct:                 ctClient,
		db:                 database,
		syncSvc:            syncSvc,
		jwtSecret:          jwtSecret,
		frontendBase:       frontendBase,
		adminPassword:      os.Getenv("ADMIN_PASSWORD"),
		superAdminPassword: os.Getenv("SUPER_ADMIN_PASSWORD"),
	}
}

// ── Auth ──────────────────────────────────────────────────────────────────

func (h *Handler) AdminLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Password == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if h.superAdminPassword != "" && req.Password == h.superAdminPassword {
		token, err := auth.NewSuperAdminToken(h.jwtSecret)
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"token": token, "role": "super_admin"})
		return
	}
	if req.Password != h.adminPassword {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	token, err := auth.NewAdminToken(h.jwtSecret)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"token": token, "role": "admin"})
}

// ── Admin: ChurchTools views ──────────────────────────────────────────────

// ListChildren returns synced children who have at least one parent relationship.
// Children without a linked parent cannot receive a QR code and are omitted.
func (h *Handler) ListChildren(w http.ResponseWriter, r *http.Request) {
	type row struct {
		CTID      int    `json:"id"`
		FirstName string `json:"firstName"`
		LastName  string `json:"lastName"`
		Birthdate string `json:"birthdate,omitempty"`
		GroupID   int    `json:"groupId"`
		GroupName string `json:"groupName"`
		HasFather bool   `json:"hasFather"`
		HasMother bool   `json:"hasMother"`
	}

	// Subquery: child CT IDs that have at least one parent relationship.
	var childIDs []int
	h.db.Model(&localdb.SyncedRelationship{}).
		Distinct("child_ct_id").
		Pluck("child_ct_id", &childIDs)

	if len(childIDs) == 0 {
		writeJSON(w, http.StatusOK, []row{})
		return
	}

	// Load persons and their primary group membership.
	var persons []localdb.SyncedPerson
	h.db.Where("ct_id IN ? AND is_child = ?", childIDs, true).
		Order("last_name, first_name").
		Find(&persons)

	// Load group memberships for those persons.
	var memberships []localdb.SyncedGroupMembership
	h.db.Where("person_ct_id IN ?", childIDs).Find(&memberships)

	membershipMap := map[int]localdb.SyncedGroupMembership{}
	for _, m := range memberships {
		// Keep first membership per person (stable order).
		if _, exists := membershipMap[m.PersonCTID]; !exists {
			membershipMap[m.PersonCTID] = m
		}
	}

	// Load all relationships and resolve parent sex.
	var rels []localdb.SyncedRelationship
	h.db.Where("child_ct_id IN ?", childIDs).Find(&rels)

	parentIDSet := map[int]struct{}{}
	for _, r := range rels {
		parentIDSet[r.ParentCTID] = struct{}{}
	}
	parentIDSlice := make([]int, 0, len(parentIDSet))
	for id := range parentIDSet {
		parentIDSlice = append(parentIDSlice, id)
	}

	parentSexMap := map[int]string{} // parentCTID → "male"/"female"/""
	if len(parentIDSlice) > 0 {
		var parentPersons []localdb.SyncedPerson
		h.db.Select("ct_id, sex").Where("ct_id IN ?", parentIDSlice).Find(&parentPersons)
		for _, p := range parentPersons {
			parentSexMap[p.CTID] = p.Sex
		}
	}

	type sexFlags struct{ hasFather, hasMother bool }
	childFlags := map[int]sexFlags{}
	for _, rel := range rels {
		sex := parentSexMap[rel.ParentCTID]
		f := childFlags[rel.ChildCTID]
		if sex == "male" {
			f.hasFather = true
		} else if sex == "female" {
			f.hasMother = true
		}
		childFlags[rel.ChildCTID] = f
	}

	result := make([]row, 0, len(persons))
	for _, p := range persons {
		m := membershipMap[p.CTID]
		f := childFlags[p.CTID]
		result = append(result, row{
			CTID:      p.CTID,
			FirstName: p.FirstName,
			LastName:  p.LastName,
			Birthdate: p.Birthdate,
			GroupID:   m.GroupID,
			GroupName: m.GroupName,
			HasFather: f.hasFather,
			HasMother: f.hasMother,
		})
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) GetParentDetail(w http.ResponseWriter, r *http.Request) {
	childID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	parentIDs, err := h.ct.GetParentsForChild(childID)
	if err != nil {
		slog.Warn("GetParentDetail: could not get parents", "childId", childID, "err", err)
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	if len(parentIDs) == 0 {
		slog.Warn("GetParentDetail: no parents found, using id as parent", "id", childID)
		parentIDs = []int{childID}
	}
	parentID := parentIDs[0]
	parent, err := h.ct.GetPerson(parentID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	children, err := h.ct.GetChildrenForParent(parentID)
	if err != nil {
		slog.Warn("GetParentDetail: could not get children", "parentId", parentID, "err", err)
		children = nil
	}
	writeJSON(w, http.StatusOK, map[string]any{"parent": parent, "children": children})
}

// GetParentDetailByParentID returns parent + children from the local DB, keyed by parent CT ID.
func (h *Handler) GetParentDetailByParentID(w http.ResponseWriter, r *http.Request) {
	parentID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var parent localdb.SyncedPerson
	h.db.Where("ct_id = ? AND is_parent = ?", parentID, true).Limit(1).Find(&parent)
	if parent.ID == 0 {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	// Load child IDs for this parent.
	var rels []localdb.SyncedRelationship
	h.db.Where("parent_ct_id = ?", parentID).Find(&rels)

	type childRow struct {
		ID        int    `json:"id"`
		FirstName string `json:"firstName"`
		LastName  string `json:"lastName"`
		Birthdate string `json:"birthdate,omitempty"`
		GroupID   int    `json:"groupId"`
		GroupName string `json:"groupName"`
	}

	var children []childRow
	for _, rel := range rels {
		var child localdb.SyncedPerson
		h.db.Where("ct_id = ? AND is_child = ?", rel.ChildCTID, true).Limit(1).Find(&child)
		if child.ID == 0 {
			continue
		}
		var m localdb.SyncedGroupMembership
		h.db.Where("person_ct_id = ?", rel.ChildCTID).Limit(1).Find(&m)
		children = append(children, childRow{
			ID:        child.CTID,
			FirstName: child.FirstName,
			LastName:  child.LastName,
			Birthdate: child.Birthdate,
			GroupID:   m.GroupID,
			GroupName: m.GroupName,
		})
	}
	if children == nil {
		children = []childRow{}
	}

	type parentRow struct {
		ID          int    `json:"id"`
		FirstName   string `json:"firstName"`
		LastName    string `json:"lastName"`
		Email       string `json:"email"`
		PhoneNumber string `json:"phoneNumber"`
		Mobile      string `json:"mobile"`
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"parent": parentRow{
			ID:          parent.CTID,
			FirstName:   parent.FirstName,
			LastName:    parent.LastName,
			Email:       parent.Email,
			PhoneNumber: parent.PhoneNumber,
			Mobile:      parent.Mobile,
		},
		"children": children,
	})
}

func (h *Handler) GenerateQR(w http.ResponseWriter, r *http.Request) {
	childID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	parentID := childID
	if parentIDs, err := h.ct.GetParentsForChild(childID); err == nil && len(parentIDs) > 0 {
		parentID = parentIDs[0]
	}
	token, err := auth.NewParentToken(h.jwtSecret, parentID)
	if err != nil {
		http.Error(w, "could not create token", http.StatusInternalServerError)
		return
	}
	url := fmt.Sprintf("%s/checkin/%s", h.frontendBase, token)
	png, err := qrcode.Encode(url, qrcode.Medium, 512)
	if err != nil {
		http.Error(w, "qr generation failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Access-Control-Expose-Headers", "X-Checkin-Url")
	w.Header().Set("X-Checkin-Url", url)
	w.Header().Set("Content-Type", "image/png")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(png)
}

// SyncCT triggers a full CT → local DB sync synchronously.
// The client waits for completion; the spinner in the frontend signals progress.
func (h *Handler) SyncCT(w http.ResponseWriter, r *http.Request) {
	if err := h.syncSvc.Run(context.Background()); err != nil {
		slog.Warn("SyncCT failed", "err", err)
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ListGroups returns the configured child groups (ID + name from CT).
func (h *Handler) ListGroups(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.syncSvc.Groups())
}

// ListParents returns synced parent records from the local DB.
// Each parent includes a deduplicated list of groups their children belong to.
// Optional query param: ?sex=male  ?sex=female  (omit for all)
func (h *Handler) ListParents(w http.ResponseWriter, r *http.Request) {
	type groupEntry struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	type row struct {
		CTID        int          `json:"id"`
		FirstName   string       `json:"firstName"`
		LastName    string       `json:"lastName"`
		Sex         string       `json:"sex"`
		Email       string       `json:"email"`
		PhoneNumber string       `json:"phoneNumber"`
		Mobile      string       `json:"mobile"`
		Groups      []groupEntry `json:"groups"`
	}

	q := h.db.Model(&localdb.SyncedPerson{}).Where("is_parent = ?", true)
	if sex := r.URL.Query().Get("sex"); sex != "" {
		q = q.Where("sex = ?", sex)
	}

	var persons []localdb.SyncedPerson
	q.Order("last_name, first_name").Find(&persons)

	if len(persons) == 0 {
		writeJSON(w, http.StatusOK, []row{})
		return
	}

	// Collect all parent CT IDs.
	parentIDs := make([]int, len(persons))
	for i, p := range persons {
		parentIDs[i] = p.CTID
	}

	// Load relationships: parent → children.
	var rels []localdb.SyncedRelationship
	h.db.Where("parent_ct_id IN ?", parentIDs).Find(&rels)

	childIDSet := map[int]struct{}{}
	parentToChildren := map[int][]int{} // parentCTID → []childCTID
	for _, rel := range rels {
		parentToChildren[rel.ParentCTID] = append(parentToChildren[rel.ParentCTID], rel.ChildCTID)
		childIDSet[rel.ChildCTID] = struct{}{}
	}

	// Load group memberships for all those children.
	childIDs := make([]int, 0, len(childIDSet))
	for id := range childIDSet {
		childIDs = append(childIDs, id)
	}

	// childGroupMap: childCTID → []SyncedGroupMembership
	childGroupMap := map[int][]localdb.SyncedGroupMembership{}
	if len(childIDs) > 0 {
		var memberships []localdb.SyncedGroupMembership
		h.db.Where("person_ct_id IN ?", childIDs).Find(&memberships)
		for _, m := range memberships {
			childGroupMap[m.PersonCTID] = append(childGroupMap[m.PersonCTID], m)
		}
	}

	result := make([]row, 0, len(persons))
	for _, p := range persons {
		// Collect unique groups across all children of this parent.
		seen := map[int]struct{}{}
		var groups []groupEntry
		for _, childID := range parentToChildren[p.CTID] {
			for _, m := range childGroupMap[childID] {
				if _, exists := seen[m.GroupID]; !exists {
					seen[m.GroupID] = struct{}{}
					groups = append(groups, groupEntry{ID: m.GroupID, Name: m.GroupName})
				}
			}
		}
		if groups == nil {
			groups = []groupEntry{}
		}
		result = append(result, row{
			CTID:        p.CTID,
			FirstName:   p.FirstName,
			LastName:    p.LastName,
			Sex:         p.Sex,
			Email:       p.Email,
			PhoneNumber: p.PhoneNumber,
			Mobile:      p.Mobile,
			Groups:      groups,
		})
	}
	writeJSON(w, http.StatusOK, result)
}

// ── Admin: check-in management ────────────────────────────────────────────

// ListCheckins returns today's check-in records.
// Optional query params: ?status=pending|registered|checked_in  ?groupId=N
func (h *Handler) ListCheckins(w http.ResponseWriter, r *http.Request) {
	q := h.db.Where("event_date = ?", localdb.Today())
	if s := r.URL.Query().Get("status"); s != "" {
		q = q.Where("status = ?", s)
	}
	if g := r.URL.Query().Get("groupId"); g != "" {
		if gid, err := strconv.Atoi(g); err == nil && gid > 0 {
			q = q.Where("group_id = ?", gid)
		}
	}
	var records []localdb.CheckIn
	if err := q.Order("created_at asc").Find(&records).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, records)
}

// ConfirmTagHandout moves a check-in from "pending" to "registered".
// Called by the door volunteer after handing out the name tag.
func (h *Handler) ConfirmTagHandout(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var record localdb.CheckIn
	if err := h.db.First(&record, id).Error; err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if record.Status != localdb.StatusPending {
		http.Error(w, fmt.Sprintf("cannot confirm: status is %q", record.Status), http.StatusConflict)
		return
	}
	now := time.Now()
	record.Status = localdb.StatusRegistered
	record.RegisteredAt = &now
	if err := h.db.Save(&record).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, record)
}

// CheckInAtGroup moves a check-in to "checked_in" and syncs to ChurchTools.
// Accepts both "pending" and "registered" – tag handout is not a required step.
func (h *Handler) CheckInAtGroup(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var record localdb.CheckIn
	if err := h.db.First(&record, id).Error; err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if record.Status != localdb.StatusRegistered && record.Status != localdb.StatusPending {
		http.Error(w, fmt.Sprintf("cannot check in: status is %q", record.Status), http.StatusConflict)
		return
	}
	now := time.Now()
	record.Status = localdb.StatusCheckedIn
	record.CheckedInAt = &now
	if err := h.db.Save(&record).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Best-effort sync to ChurchTools.
	if err := h.ct.CheckIn(record.ChildID, record.GroupID); err != nil {
		slog.Warn("CheckInAtGroup: CT sync failed", "childId", record.ChildID, "err", err)
	}
	writeJSON(w, http.StatusOK, record)
}

// ── Parent-facing endpoints ───────────────────────────────────────────────

type childWithStatus struct {
	ct.Child
	// Status is "" (not registered today), "pending", "registered", or "checked_in".
	Status string `json:"status"`
}

// GetParentQR returns a QR code PNG for the parent's own check-in URL.
// Security: the token is validated; only that parent's URL is encoded.
func (h *Handler) GetParentQR(w http.ResponseWriter, r *http.Request) {
	tokenStr := chi.URLParam(r, "token")
	claims, err := auth.ParseToken(h.jwtSecret, tokenStr)
	if err != nil || claims.Role != "parent" {
		http.Error(w, "invalid or expired token", http.StatusUnauthorized)
		return
	}
	url := fmt.Sprintf("%s/checkin/%s", h.frontendBase, tokenStr)
	png, err := qrcode.Encode(url, qrcode.Medium, 400)
	if err != nil {
		http.Error(w, "qr generation failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "max-age=3600")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(png)
}

// GetParentCheckinPage returns the parent and their children with today's local status.
func (h *Handler) GetParentCheckinPage(w http.ResponseWriter, r *http.Request) {
	tokenStr := chi.URLParam(r, "token")
	claims, err := auth.ParseToken(h.jwtSecret, tokenStr)
	if err != nil || claims.Role != "parent" {
		http.Error(w, "invalid or expired token", http.StatusUnauthorized)
		return
	}
	parent, err := h.ct.GetPerson(claims.ParentID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	children, err := h.ct.GetChildrenForParent(claims.ParentID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	// Overwrite group info from local DB (CT client returns hardcoded group).
	for i := range children {
		var m localdb.SyncedGroupMembership
		if err := h.db.Where("person_ct_id = ?", children[i].ID).Limit(1).Find(&m).Error; err == nil && m.GroupID != 0 {
			children[i].GroupID = m.GroupID
			children[i].GroupName = m.GroupName
		}
	}

	var withStatus []childWithStatus
	for _, child := range children {
		var record localdb.CheckIn
		status := ""
		if err := h.db.Where("event_date = ? AND child_id = ?", localdb.Today(), child.ID).
			First(&record).Error; err == nil {
			status = record.Status
		}
		withStatus = append(withStatus, childWithStatus{Child: child, Status: status})
	}
	writeJSON(w, http.StatusOK, map[string]any{"parent": parent, "children": withStatus})
}

// RegisterChild creates (or resets to pending) a check-in record.
// Step 1 of the 2-step flow: parent taps "Anmelden" at the entrance.
func (h *Handler) RegisterChild(w http.ResponseWriter, r *http.Request) {
	tokenStr := chi.URLParam(r, "token")
	claims, err := auth.ParseToken(h.jwtSecret, tokenStr)
	if err != nil || claims.Role != "parent" {
		http.Error(w, "invalid or expired token", http.StatusUnauthorized)
		return
	}
	childID, err := strconv.Atoi(chi.URLParam(r, "childId"))
	if err != nil {
		http.Error(w, "invalid childId", http.StatusBadRequest)
		return
	}

	// Verify the child belongs to this parent.
	children, err := h.ct.GetChildrenForParent(claims.ParentID)
	if err != nil {
		http.Error(w, "could not verify child", http.StatusBadGateway)
		return
	}
	var child *ct.Child
	for i := range children {
		if children[i].ID == childID {
			child = &children[i]
			break
		}
	}
	if child == nil {
		http.Error(w, "child not found for this parent", http.StatusForbidden)
		return
	}

	today := localdb.Today()
	var record localdb.CheckIn
	// Find existing record for today or create a new one.
	h.db.Where(localdb.CheckIn{EventDate: today, ChildID: childID}).First(&record)

	record.EventDate = today
	record.ChildID = childID
	record.FirstName = child.FirstName
	record.LastName = child.LastName
	record.Birthdate = child.Birthdate
	record.GroupID = child.GroupID
	record.GroupName = child.GroupName
	record.ParentID = claims.ParentID
	record.Status = localdb.StatusPending

	if err := h.db.Save(&record).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": record.Status, "id": record.ID})
}

// SetCheckInStatus overrides a check-in record's status to any valid value.
// Sending status="" deletes the record entirely (full reset).
// Only accessible to super_admin tokens.
func (h *Handler) SetCheckInStatus(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var req struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	switch req.Status {
	case "", localdb.StatusPending, localdb.StatusRegistered, localdb.StatusCheckedIn:
		// valid
	default:
		http.Error(w, "invalid status", http.StatusBadRequest)
		return
	}

	var record localdb.CheckIn
	if err := h.db.First(&record, id).Error; err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	if req.Status == "" {
		if err := h.db.Unscoped().Delete(&record).Error; err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
		return
	}

	record.Status = req.Status
	now := time.Now()
	switch req.Status {
	case localdb.StatusPending:
		// Full reset: clear both timestamps
		record.RegisteredAt = nil
		record.CheckedInAt = nil
	case localdb.StatusRegistered:
		// Namensschild is independent – set RegisteredAt if not already set,
		// clear CheckedInAt (checked_in → registered rollback)
		if record.RegisteredAt == nil {
			record.RegisteredAt = &now
		}
		record.CheckedInAt = nil
	case localdb.StatusCheckedIn:
		// Stamp CheckedInAt; leave RegisteredAt as-is (may be nil if tag was skipped)
		if record.CheckedInAt == nil {
			record.CheckedInAt = &now
		}
	}

	if err := h.db.Save(&record).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, record)
}

// ── helpers ───────────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
