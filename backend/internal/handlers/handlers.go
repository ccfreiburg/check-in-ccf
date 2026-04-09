package handlers

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/ccf/check-in/backend/internal/auth"
	"github.com/ccf/check-in/backend/internal/ct"
	"github.com/ccf/check-in/backend/internal/ctsync"
	localdb "github.com/ccf/check-in/backend/internal/db"
	"github.com/go-chi/chi/v5"
	qrcode "github.com/skip2/go-qrcode"
	"gorm.io/gorm"
)

// ctClientIface is the subset of ct.Client methods used by handlers.
// It is unexported; any type whose method set covers these signatures satisfies it.
type ctClientIface interface {
	LoginUser(username, password string) (int, error)
	GetPerson(id int) (*ct.Person, error)
	GetParentsForChild(childID int) ([]int, error)
	GetChildrenForParent(parentID int) ([]ct.Child, error)
	CheckIn(childID, groupID int) error
}

// syncServiceIface is the subset of ctsync.Service methods used by handlers.
type syncServiceIface interface {
	Run(ctx context.Context) error
	Groups() []ctsync.GroupConfig
}

type Handler struct {
	ct                ctClientIface
	db                *gorm.DB
	syncSvc           syncServiceIface
	jwtSecret         []byte
	frontendBase      string
	localPassword     bool
	volunteerPassword string
	adminPassword     string
	vapidPrivateKey   string
	vapidPublicKey    string
	reportsDir        string
	adminEmails       map[string]struct{} // emails that always get "admin" role regardless of synced_staff
}

func New(ctClient ctClientIface, database *gorm.DB, syncSvc syncServiceIface, jwtSecret []byte, frontendBase string) *Handler {
	reportsDir := os.Getenv("REPORTS_DIR")
	if reportsDir == "" {
		reportsDir = "./reports"
	}
	adminEmails := map[string]struct{}{}
	for _, raw := range strings.Split(os.Getenv("CT_ADMIN_PERSONS"), ",") {
		raw = strings.TrimSpace(raw)
		if raw != "" {
			adminEmails[strings.ToLower(raw)] = struct{}{}
		}
	}
	return &Handler{
		ct:                ctClient,
		db:                database,
		syncSvc:           syncSvc,
		jwtSecret:         jwtSecret,
		frontendBase:      frontendBase,
		localPassword:     os.Getenv("LOCAL_PASSWORD") == "true",
		volunteerPassword: os.Getenv("VOLUNTEER_PASSWORD"),
		adminPassword:     os.Getenv("ADMIN_PASSWORD"),
		vapidPrivateKey:   os.Getenv("VAPID_PRIVATE_KEY"),
		vapidPublicKey:    os.Getenv("VAPID_PUBLIC_KEY"),
		reportsDir:        reportsDir,
		adminEmails:       adminEmails,
	}
}

// ── Auth ──────────────────────────────────────────────────────────────────

func (h *Handler) AdminLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Password == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	var role string

	if h.localPassword {
		// Local-password mode: match against env vars, ignore username.
		switch {
		case h.adminPassword != "" && req.Password == h.adminPassword:
			role = "admin"
		case h.volunteerPassword != "" && req.Password == h.volunteerPassword:
			role = "volunteer"
		default:
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
	} else {
		// ChurchTools auth mode: verify credentials against CT, then look up role.
		if req.Username == "" {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		ctPersonID, err := h.ct.LoginUser(req.Username, req.Password)
		if err != nil {
			slog.Warn("CT login failed", "username", req.Username, "err", err)
			http.Error(w, "forbidden: CT authentication failed", http.StatusForbidden)
			return
		}
		_ = ctPersonID // person ID not needed; email is used for role lookup

		// CT_ADMIN_PERSONS: emails that always get admin regardless of synced_staff.
		_, emailMatch := h.adminEmails[strings.ToLower(req.Username)]
		if emailMatch {
			role = "admin"
		} else {
			// Look up role by email in synced_staff (populated during CT sync).
			var staff localdb.SyncedStaff
			if err := h.db.Where("LOWER(email) = LOWER(?)", req.Username).First(&staff).Error; err != nil {
				slog.Warn("CT login: authenticated but no staff role assigned", "email", req.Username)
				http.Error(w, "forbidden: no role assigned — not a member of any configured group", http.StatusForbidden)
				return
			}
			role = staff.Role
		}
	}

	var (
		token string
		err   error
	)
	switch role {
	case "admin":
		token, err = auth.NewAdminToken(h.jwtSecret)
	default:
		token, err = auth.NewVolunteerToken(h.jwtSecret)
	}
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"token": token, "role": role})
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
	// Override group info from the local sync DB (CT client returns hardcoded group).
	for i := range children {
		var m localdb.SyncedGroupMembership
		if err := h.db.Where("person_ct_id = ?", children[i].ID).Limit(1).Find(&m).Error; err == nil && m.GroupID != 0 {
			children[i].GroupID = m.GroupID
			children[i].GroupName = m.GroupName
		}
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

// GetChildParents returns all parents linked to a child (by child CT ID) from the local DB.
func (h *Handler) GetChildParents(w http.ResponseWriter, r *http.Request) {
	childID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var rels []localdb.SyncedRelationship
	h.db.Where("child_ct_id = ?", childID).Find(&rels)

	type parentRow struct {
		ID          int    `json:"id"`
		FirstName   string `json:"firstName"`
		LastName    string `json:"lastName"`
		Email       string `json:"email"`
		PhoneNumber string `json:"phoneNumber"`
		Mobile      string `json:"mobile"`
	}

	parents := make([]parentRow, 0, len(rels))
	for _, rel := range rels {
		var p localdb.SyncedPerson
		h.db.Where("ct_id = ? AND is_parent = ?", rel.ParentCTID, true).Limit(1).Find(&p)
		if p.ID == 0 {
			continue
		}
		parents = append(parents, parentRow{
			ID:          p.CTID,
			FirstName:   p.FirstName,
			LastName:    p.LastName,
			Email:       p.Email,
			PhoneNumber: p.PhoneNumber,
			Mobile:      p.Mobile,
		})
	}
	writeJSON(w, http.StatusOK, parents)
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
// EndEvent generates a CSV report for the day, then deletes all today's check-in records.
func (h *Handler) EndEvent(w http.ResponseWriter, r *http.Request) {
	today := localdb.Today()

	// Load all today's records (Unscoped to include soft-deleted checkout entries).
	var records []localdb.CheckIn
	h.db.Unscoped().Where("event_date = ?", today).
		Order("group_name, last_name, first_name").Find(&records)

	if len(records) > 0 {
		if err := h.generateReport(today, records); err != nil {
			slog.Warn("EndEvent: failed to generate report", "err", err)
			// non-fatal: proceed with deletion
		}
	}

	if err := h.db.Unscoped().Where("event_date = ?", today).Delete(&localdb.CheckIn{}).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

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

// ConfirmTagHandout toggles the TagReceived flag on a check-in record.
// Idempotent toggle: sets true if false, false if true.
// Does not affect the check-in status.
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
	now := time.Now()
	record.TagReceived = !record.TagReceived
	if record.TagReceived {
		record.RegisteredAt = &now
	} else {
		record.RegisteredAt = nil
	}
	if err := h.db.Save(&record).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, record)
}

// CheckInAtGroup moves a check-in to "checked_in" and syncs to ChurchTools.
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
	if record.Status != localdb.StatusPending {
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
	Status         string     `json:"status"`
	LastNotifiedAt *time.Time `json:"lastNotifiedAt"`
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
		var lastNotifiedAt *time.Time
		if err := h.db.Where("event_date = ? AND child_id = ?", localdb.Today(), child.ID).
			First(&record).Error; err == nil {
			status = record.Status
			lastNotifiedAt = record.LastNotifiedAt
		}
		withStatus = append(withStatus, childWithStatus{Child: child, Status: status, LastNotifiedAt: lastNotifiedAt})
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
	// Find existing record for today or create a new one (include soft-deleted rows).
	h.db.Unscoped().Where("event_date = ? AND child_id = ?", today, childID).First(&record)
	record.DeletedAt = gorm.DeletedAt{} // restore if soft-deleted

	record.EventDate = today
	record.ChildID = childID
	record.FirstName = child.FirstName
	record.LastName = child.LastName
	record.Birthdate = child.Birthdate
	record.GroupID = child.GroupID
	record.GroupName = child.GroupName
	// Override with the more accurate group info from the local sync DB.
	var membership localdb.SyncedGroupMembership
	if h.db.Where("person_ct_id = ?", childID).Limit(1).Find(&membership).Error == nil && membership.GroupID != 0 {
		record.GroupID = membership.GroupID
		record.GroupName = membership.GroupName
	}
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
// Only accessible to admin tokens.
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
	case "", localdb.StatusPending, localdb.StatusCheckedIn:
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
		now := time.Now()
		record.CheckedOutAt = &now
		h.db.Save(&record) // persist timestamp before soft-delete
		if err := h.db.Delete(&record).Error; err != nil {
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
		// Full reset: clear all timestamps and tag
		record.TagReceived = false
		record.RegisteredAt = nil
		record.CheckedInAt = nil
	case localdb.StatusCheckedIn:
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

// ── Reports ───────────────────────────────────────────────────────────────

// reportFilenameRe validates the safe filename pattern YYYY-MM-DD_NNN.csv.
var reportFilenameRe = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}_[0-9]{3}\.csv$`)

// generateReport writes a CSV file for the given event date and check-in records.
func (h *Handler) generateReport(date string, records []localdb.CheckIn) error {
	if err := os.MkdirAll(h.reportsDir, 0750); err != nil {
		return fmt.Errorf("create reports dir: %w", err)
	}
	seq, err := nextReportSeq(h.reportsDir, date)
	if err != nil {
		return err
	}
	filename := fmt.Sprintf("%s_%03d.csv", date, seq)
	path := filepath.Join(h.reportsDir, filename)
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create report file: %w", err)
	}
	defer f.Close()

	// Build parent name lookup.
	seen := map[int]struct{}{}
	var parentIDs []int
	for _, r := range records {
		if r.ParentID != 0 {
			if _, ok := seen[r.ParentID]; !ok {
				seen[r.ParentID] = struct{}{}
				parentIDs = append(parentIDs, r.ParentID)
			}
		}
	}
	parentMap := map[int]localdb.SyncedPerson{}
	if len(parentIDs) > 0 {
		var parents []localdb.SyncedPerson
		h.db.Where("ct_id IN ?", parentIDs).Find(&parents)
		for _, p := range parents {
			parentMap[p.CTID] = p
		}
	}

	fmtDate := func(t *time.Time) string {
		if t == nil {
			return ""
		}
		return t.Format("2006-01-02")
	}
	fmtTime := func(t *time.Time) string {
		if t == nil {
			return ""
		}
		return t.Format("15:04:05")
	}

	cw := csv.NewWriter(f)
	_ = cw.Write([]string{
		"Gruppe", "Vorname", "Nachname",
		"Eltern-Vorname", "Eltern-Nachname",
		"Anmeldedatum", "Anmeldezeit",
		"Check-in Datum", "Check-in Zeit",
		"Check-out Datum", "Check-out Zeit",
	})
	for _, rec := range records {
		parent := parentMap[rec.ParentID]
		_ = cw.Write([]string{
			rec.GroupName,
			rec.FirstName, rec.LastName,
			parent.FirstName, parent.LastName,
			fmtDate(rec.RegisteredAt), fmtTime(rec.RegisteredAt),
			fmtDate(rec.CheckedInAt), fmtTime(rec.CheckedInAt),
			fmtDate(rec.CheckedOutAt), fmtTime(rec.CheckedOutAt),
		})
	}
	cw.Flush()
	return cw.Error()
}

// nextReportSeq counts existing CSV files for date and returns the next sequence number.
func nextReportSeq(dir, date string) (int, error) {
	matches, err := filepath.Glob(filepath.Join(dir, date+"_*.csv"))
	if err != nil {
		return 0, err
	}
	return len(matches) + 1, nil
}

// ListReports returns metadata for all saved event report CSV files, newest first.
func (h *Handler) ListReports(w http.ResponseWriter, r *http.Request) {
	type reportEntry struct {
		Filename string `json:"filename"`
		Date     string `json:"date"`
		Size     int64  `json:"size"`
	}

	entries, err := os.ReadDir(h.reportsDir)
	if err != nil {
		if os.IsNotExist(err) {
			writeJSON(w, http.StatusOK, []reportEntry{})
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	result := make([]reportEntry, 0)
	for _, e := range entries {
		if e.IsDir() || !reportFilenameRe.MatchString(e.Name()) {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		result = append(result, reportEntry{
			Filename: e.Name(),
			Date:     e.Name()[:10],
			Size:     info.Size(),
		})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Filename > result[j].Filename })
	writeJSON(w, http.StatusOK, result)
}

// GetReport serves a single CSV report file for download.
func (h *Handler) GetReport(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "filename")
	if !reportFilenameRe.MatchString(name) {
		http.Error(w, "invalid filename", http.StatusBadRequest)
		return
	}
	path := filepath.Join(h.reportsDir, name)
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer f.Close()
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, name))
	_, _ = io.Copy(w, f)
}

// ── helpers ───────────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// ── Web Push ──────────────────────────────────────────────────────────────

// GetVAPIDPublicKey returns the server's VAPID public key so the browser can
// subscribe to push notifications.
func (h *Handler) GetVAPIDPublicKey(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"publicKey": h.vapidPublicKey})
}

// GetParentManifest returns a Web App Manifest with start_url set to the
// parent's own check-in URL. This ensures that "Add to Home Screen" on iOS
// launches directly into the parent's check-in page, not the admin login.
// GET /api/parent/{token}/manifest.json
func (h *Handler) GetParentManifest(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	if _, err := auth.ValidateParentToken(h.jwtSecret, token); err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}
	manifest := map[string]any{
		"name":             "Kinder Check-in",
		"short_name":       "Check-in",
		"description":      "Kinder Anmeldung CCF",
		"start_url":        "/checkin/" + token,
		"scope":            "/",
		"display":          "standalone",
		"background_color": "#f9fafb",
		"theme_color":      "#2563eb",
		"icons": []map[string]string{
			{"src": "/favicon.svg", "sizes": "any", "type": "image/svg+xml", "purpose": "any maskable"},
		},
	}
	w.Header().Set("Content-Type", "application/manifest+json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(manifest)
}

// SavePushSubscription stores (or updates) a Web Push subscription for the parent
// identified by the URL token.
// POST /api/parent/{token}/push-subscription
func (h *Handler) SavePushSubscription(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	claims, err := auth.ValidateParentToken(h.jwtSecret, token)
	if err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	var body struct {
		Endpoint string `json:"endpoint"`
		P256dh   string `json:"p256dh"`
		Auth     string `json:"auth"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Endpoint == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	sub := localdb.PushSubscription{
		ParentID: claims.ParentID,
		Endpoint: body.Endpoint,
		P256dh:   body.P256dh,
		Auth:     body.Auth,
	}
	// Upsert: update keys if endpoint already known.
	if err := h.db.
		Where(localdb.PushSubscription{Endpoint: body.Endpoint}).
		Assign(localdb.PushSubscription{ParentID: claims.ParentID, P256dh: body.P256dh, Auth: body.Auth}).
		FirstOrCreate(&sub).Error; err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// SendParentMessage sends a Web Push notification to all subscribed devices of
// the parent linked to the given checkin record.
// POST /api/admin/checkins/{id}/notify (requires admin JWT)
func (h *Handler) SendParentMessage(w http.ResponseWriter, r *http.Request) {
	if h.vapidPrivateKey == "" || h.vapidPublicKey == "" {
		http.Error(w, "push not configured", http.StatusNotImplemented)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var record localdb.CheckIn
	if err := h.db.First(&record, id).Error; err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	var subs []localdb.PushSubscription
	if err := h.db.Where("parent_id = ?", record.ParentID).Find(&subs).Error; err != nil || len(subs) == 0 {
		http.Error(w, "no subscription found", http.StatusNotFound)
		return
	}

	payload, _ := json.Marshal(map[string]string{
		"title": "Bitte zum Kind kommen",
		"body":  fmt.Sprintf("Bitte komm zu %s %s in %s.", record.FirstName, record.LastName, record.GroupName),
	})

	var lastErr error
	sent := 0
	for _, sub := range subs {
		resp, err := webpush.SendNotification(payload, &webpush.Subscription{
			Endpoint: sub.Endpoint,
			Keys: webpush.Keys{
				P256dh: sub.P256dh,
				Auth:   sub.Auth,
			},
		}, &webpush.Options{
			VAPIDPublicKey:  h.vapidPublicKey,
			VAPIDPrivateKey: h.vapidPrivateKey,
			Subscriber:      h.frontendBase,
			TTL:             86400,
		})
		if err != nil {
			slog.Warn("push send failed", "endpoint", sub.Endpoint, "err", err)
			lastErr = err
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		slog.Info("push response", "status", resp.StatusCode, "body", string(body), "endpoint", sub.Endpoint)
		if resp.StatusCode == http.StatusGone {
			// Subscription expired — remove it.
			h.db.Delete(&sub)
		} else if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			sent++
		} else {
			lastErr = fmt.Errorf("push service returned %d: %s", resp.StatusCode, string(body))
		}
	}

	if sent == 0 {
		msg := "keine aktive Subscription gefunden"
		if lastErr != nil {
			msg = lastErr.Error()
		}
		http.Error(w, msg, http.StatusBadGateway)
		return
	}
	now := time.Now()
	record.LastNotifiedAt = &now
	h.db.Save(&record)
	writeJSON(w, http.StatusOK, map[string]any{"sent": sent})
}

// ClearNotify clears the LastNotifiedAt timestamp for a check-in record.
func (h *Handler) ClearNotify(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var record localdb.CheckIn
	if err := h.db.First(&record, id).Error; err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	record.LastNotifiedAt = nil
	if err := h.db.Save(&record).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, record)
}
