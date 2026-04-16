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
// Returns local gorm_id as "id".
func (h *Handler) ListChildren(w http.ResponseWriter, r *http.Request) {
	type row struct {
		ID        uint   `json:"id"` // local gorm_id
		FirstName string `json:"firstName"`
		LastName  string `json:"lastName"`
		Birthdate string `json:"birthdate,omitempty"`
		GroupID   int    `json:"groupId"`
		GroupName string `json:"groupName"`
		HasFather bool   `json:"hasFather"`
		HasMother bool   `json:"hasMother"`
	}

	var persons []localdb.SyncedPerson
	h.db.Where("is_child = ?", true).Order("last_name, first_name").Find(&persons)

	if len(persons) == 0 {
		writeJSON(w, http.StatusOK, []row{})
		return
	}

	childCTIDs := make([]int, 0, len(persons))
	for _, p := range persons {
		childCTIDs = append(childCTIDs, p.CTID)
	}

	// Load relationships to determine which children have parents + parent sex.
	var rels []localdb.SyncedRelationship
	h.db.Where("child_ct_id IN ?", childCTIDs).Find(&rels)

	childToParentCTIDs := map[int][]int{}
	parentCTIDSet := map[int]struct{}{}
	for _, rel := range rels {
		childToParentCTIDs[rel.ChildCTID] = append(childToParentCTIDs[rel.ChildCTID], rel.ParentCTID)
		parentCTIDSet[rel.ParentCTID] = struct{}{}
	}

	parentCTIDs := make([]int, 0, len(parentCTIDSet))
	for id := range parentCTIDSet {
		parentCTIDs = append(parentCTIDs, id)
	}

	parentSexMap := map[int]string{}
	if len(parentCTIDs) > 0 {
		var parents []localdb.SyncedPerson
		h.db.Select("ct_id, sex").Where("ct_id IN ?", parentCTIDs).Find(&parents)
		for _, p := range parents {
			parentSexMap[p.CTID] = p.Sex
		}
	}

	var memberships []localdb.SyncedGroupMembership
	h.db.Where("person_ct_id IN ?", childCTIDs).Find(&memberships)
	membershipMap := map[int]localdb.SyncedGroupMembership{}
	for _, m := range memberships {
		if _, exists := membershipMap[m.PersonCTID]; !exists {
			membershipMap[m.PersonCTID] = m
		}
	}

	result := make([]row, 0, len(persons))
	for _, p := range persons {
		pCTIDs, hasParent := childToParentCTIDs[p.CTID]
		if !hasParent {
			continue
		}
		hasFather, hasMother := false, false
		for _, pCTID := range pCTIDs {
			switch parentSexMap[pCTID] {
			case "male":
				hasFather = true
			case "female":
				hasMother = true
			}
		}
		m := membershipMap[p.CTID]
		result = append(result, row{
			ID:        p.ID,
			FirstName: p.FirstName,
			LastName:  p.LastName,
			Birthdate: p.Birthdate,
			GroupID:   m.GroupID,
			GroupName: m.GroupName,
			HasFather: hasFather,
			HasMother: hasMother,
		})
	}
	writeJSON(w, http.StatusOK, result)
}

// writeParentDetail is a shared helper that loads children for a parent
// (via SyncedRelationship) and writes the full parent+children JSON response.
func (h *Handler) writeParentDetail(w http.ResponseWriter, parent *localdb.SyncedPerson) {
	type childRow struct {
		ID        uint   `json:"id"` // local gorm_id
		FirstName string `json:"firstName"`
		LastName  string `json:"lastName"`
		Birthdate string `json:"birthdate,omitempty"`
		GroupID   int    `json:"groupId"`
		GroupName string `json:"groupName"`
	}
	type parentRow struct {
		ID          uint   `json:"id"` // local gorm_id
		FirstName   string `json:"firstName"`
		LastName    string `json:"lastName"`
		Email       string `json:"email"`
		PhoneNumber string `json:"phoneNumber"`
		Mobile      string `json:"mobile"`
		Sex         string `json:"sex"`
		IsGuest     bool   `json:"isGuest"`
	}

	var rels []localdb.SyncedRelationship
	h.db.Where("parent_ct_id = ?", parent.CTID).Find(&rels)

	children := make([]childRow, 0, len(rels))
	for _, rel := range rels {
		var child localdb.SyncedPerson
		if h.db.Where("ct_id = ? AND is_child = ?", rel.ChildCTID, true).Limit(1).Find(&child).Error != nil || child.ID == 0 {
			continue
		}
		var m localdb.SyncedGroupMembership
		h.db.Where("person_ct_id = ?", rel.ChildCTID).Limit(1).Find(&m)
		children = append(children, childRow{
			ID:        child.ID,
			FirstName: child.FirstName,
			LastName:  child.LastName,
			Birthdate: child.Birthdate,
			GroupID:   m.GroupID,
			GroupName: m.GroupName,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"parent": parentRow{
			ID:          parent.ID,
			FirstName:   parent.FirstName,
			LastName:    parent.LastName,
			Email:       parent.Email,
			PhoneNumber: parent.PhoneNumber,
			Mobile:      parent.Mobile,
			Sex:         parent.Sex,
			IsGuest:     parent.IsGuest,
		},
		"children": children,
	})
}

// GetParentDetail returns parent+children detail, navigated by child gorm_id.
func (h *Handler) GetParentDetail(w http.ResponseWriter, r *http.Request) {
	childID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var child localdb.SyncedPerson
	if err := h.db.First(&child, childID).Error; err != nil {
		http.Error(w, "child not found", http.StatusNotFound)
		return
	}

	var rel localdb.SyncedRelationship
	h.db.Where("child_ct_id = ?", child.CTID).Limit(1).Find(&rel)
	if rel.ParentCTID == 0 {
		http.Error(w, "no parent found", http.StatusNotFound)
		return
	}

	var parent localdb.SyncedPerson
	if h.db.Where("ct_id = ? AND is_parent = ?", rel.ParentCTID, true).Limit(1).Find(&parent).Error != nil || parent.ID == 0 {
		http.Error(w, "parent not found", http.StatusNotFound)
		return
	}
	h.writeParentDetail(w, &parent)
}

// GetParentDetailByParentID returns parent+children detail, navigated by parent gorm_id.
func (h *Handler) GetParentDetailByParentID(w http.ResponseWriter, r *http.Request) {
	parentID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var parent localdb.SyncedPerson
	if err := h.db.First(&parent, parentID).Error; err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	h.writeParentDetail(w, &parent)
}

// GetChildParents returns all parents linked to a child (by child gorm_id) from the local DB.
func (h *Handler) GetChildParents(w http.ResponseWriter, r *http.Request) {
	childID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var child localdb.SyncedPerson
	if err := h.db.First(&child, childID).Error; err != nil {
		http.Error(w, "child not found", http.StatusNotFound)
		return
	}

	var rels []localdb.SyncedRelationship
	h.db.Where("child_ct_id = ?", child.CTID).Find(&rels)

	type parentRow struct {
		ID          uint   `json:"id"` // local gorm_id
		FirstName   string `json:"firstName"`
		LastName    string `json:"lastName"`
		Email       string `json:"email"`
		PhoneNumber string `json:"phoneNumber"`
		Mobile      string `json:"mobile"`
	}

	parents := make([]parentRow, 0, len(rels))
	for _, rel := range rels {
		var p localdb.SyncedPerson
		if h.db.Where("ct_id = ? AND is_parent = ?", rel.ParentCTID, true).Limit(1).Find(&p).Error != nil || p.ID == 0 {
			continue
		}
		parents = append(parents, parentRow{
			ID:          p.ID,
			FirstName:   p.FirstName,
			LastName:    p.LastName,
			Email:       p.Email,
			PhoneNumber: p.PhoneNumber,
			Mobile:      p.Mobile,
		})
	}
	writeJSON(w, http.StatusOK, parents)
}

// GenerateQR generates a QR code PNG for the parent identified by parent gorm_id.
// POST /api/admin/parents/{id}/qr
func (h *Handler) GenerateQR(w http.ResponseWriter, r *http.Request) {
	parentID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var parent localdb.SyncedPerson
	if err := h.db.First(&parent, parentID).Error; err != nil {
		http.Error(w, "parent not found", http.StatusNotFound)
		return
	}
	token, err := auth.NewParentToken(h.jwtSecret, int(parent.ID))
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
// Returns local gorm_id as "id"; includes isGuest flag.
// Optional query param: ?sex=male  ?sex=female  (omit for all)
func (h *Handler) ListParents(w http.ResponseWriter, r *http.Request) {
	type groupEntry struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	type row struct {
		ID          uint         `json:"id"` // local gorm_id
		FirstName   string       `json:"firstName"`
		LastName    string       `json:"lastName"`
		Sex         string       `json:"sex"`
		Email       string       `json:"email"`
		PhoneNumber string       `json:"phoneNumber"`
		Mobile      string       `json:"mobile"`
		IsGuest     bool         `json:"isGuest"`
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

	// Collect all parent CTIDs for relationship lookups.
	parentCTIDs := make([]int, len(persons))
	for i, p := range persons {
		parentCTIDs[i] = p.CTID
	}

	// Load relationships: parent CTID → children CTIDs.
	var rels []localdb.SyncedRelationship
	h.db.Where("parent_ct_id IN ?", parentCTIDs).Find(&rels)

	childIDSet := map[int]struct{}{}
	parentToChildren := map[int][]int{} // parentCTID → []childCTID
	for _, rel := range rels {
		parentToChildren[rel.ParentCTID] = append(parentToChildren[rel.ParentCTID], rel.ChildCTID)
		childIDSet[rel.ChildCTID] = struct{}{}
	}

	childIDs := make([]int, 0, len(childIDSet))
	for id := range childIDSet {
		childIDs = append(childIDs, id)
	}

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
		seen := map[int]struct{}{}
		var groups []groupEntry
		for _, childCTID := range parentToChildren[p.CTID] {
			for _, m := range childGroupMap[childCTID] {
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
			ID:          p.ID,
			FirstName:   p.FirstName,
			LastName:    p.LastName,
			Sex:         p.Sex,
			Email:       p.Email,
			PhoneNumber: p.PhoneNumber,
			Mobile:      p.Mobile,
			IsGuest:     p.IsGuest,
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
	// Best-effort sync to ChurchTools (skipped for guest children).
	if !record.IsGuest {
		var child localdb.SyncedPerson
		if err := h.db.First(&child, record.ChildID).Error; err == nil {
			if ctErr := h.ct.CheckIn(child.CTID, record.GroupID); ctErr != nil {
				slog.Warn("CheckInAtGroup: CT sync failed", "childId", record.ChildID, "err", ctErr)
			}
		}
	}
	writeJSON(w, http.StatusOK, record)
}

// ── Parent-facing endpoints ───────────────────────────────────────────────

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
// Uses the local DB for both CT-synced and guest families (no CT API dependency).
func (h *Handler) GetParentCheckinPage(w http.ResponseWriter, r *http.Request) {
	tokenStr := chi.URLParam(r, "token")
	claims, err := auth.ParseToken(h.jwtSecret, tokenStr)
	if err != nil || claims.Role != "parent" {
		http.Error(w, "invalid or expired token", http.StatusUnauthorized)
		return
	}

	var parent localdb.SyncedPerson
	if err := h.db.First(&parent, claims.ParentID).Error; err != nil {
		http.Error(w, "parent not found", http.StatusNotFound)
		return
	}

	var rels []localdb.SyncedRelationship
	h.db.Where("parent_ct_id = ?", parent.CTID).Find(&rels)

	type childItem struct {
		ID             uint       `json:"id"` // child gorm_id
		FirstName      string     `json:"firstName"`
		LastName       string     `json:"lastName"`
		Birthdate      string     `json:"birthdate"`
		GroupID        int        `json:"groupId"`
		GroupName      string     `json:"groupName"`
		Status         string     `json:"status"`
		LastNotifiedAt *time.Time `json:"lastNotifiedAt"`
	}
	type parentItem struct {
		ID          uint   `json:"id"`
		FirstName   string `json:"firstName"`
		LastName    string `json:"lastName"`
		Email       string `json:"email"`
		PhoneNumber string `json:"phoneNumber"`
		Mobile      string `json:"mobile"`
	}

	today := localdb.Today()
	children := make([]childItem, 0, len(rels))
	for _, rel := range rels {
		var child localdb.SyncedPerson
		if h.db.Where("ct_id = ? AND is_child = ?", rel.ChildCTID, true).Limit(1).Find(&child).Error != nil || child.ID == 0 {
			continue
		}
		var m localdb.SyncedGroupMembership
		h.db.Where("person_ct_id = ?", rel.ChildCTID).Limit(1).Find(&m)

		var record localdb.CheckIn
		status := ""
		var lastNotifiedAt *time.Time
		if h.db.Where("event_date = ? AND child_id = ?", today, child.ID).First(&record).Error == nil {
			status = record.Status
			lastNotifiedAt = record.LastNotifiedAt
		}
		children = append(children, childItem{
			ID:             child.ID,
			FirstName:      child.FirstName,
			LastName:       child.LastName,
			Birthdate:      child.Birthdate,
			GroupID:        m.GroupID,
			GroupName:      m.GroupName,
			Status:         status,
			LastNotifiedAt: lastNotifiedAt,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"parent": parentItem{
			ID:          parent.ID,
			FirstName:   parent.FirstName,
			LastName:    parent.LastName,
			Email:       parent.Email,
			PhoneNumber: parent.PhoneNumber,
			Mobile:      parent.Mobile,
		},
		"children": children,
	})
}

// RegisterChild creates (or resets to pending) a check-in record.
// Step 1 of the 2-step flow: parent taps "Anmelden" at the entrance.
// childId in the URL is the child's local gorm_id.
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

	// Load parent from local DB by gorm_id (stored in JWT).
	var parent localdb.SyncedPerson
	if err := h.db.First(&parent, claims.ParentID).Error; err != nil {
		http.Error(w, "parent not found", http.StatusUnauthorized)
		return
	}

	// Load child from local DB by gorm_id.
	var child localdb.SyncedPerson
	if err := h.db.First(&child, childID).Error; err != nil {
		http.Error(w, "child not found", http.StatusNotFound)
		return
	}

	// Verify the child belongs to this parent via SyncedRelationship.
	var rel localdb.SyncedRelationship
	h.db.Where("parent_ct_id = ? AND child_ct_id = ?", parent.CTID, child.CTID).Limit(1).Find(&rel)
	if rel.ParentCTID == 0 {
		http.Error(w, "child not found for this parent", http.StatusForbidden)
		return
	}

	// Get group from local DB.
	var membership localdb.SyncedGroupMembership
	h.db.Where("person_ct_id = ?", child.CTID).Limit(1).Find(&membership)

	today := localdb.Today()
	var record localdb.CheckIn
	h.db.Unscoped().Where("event_date = ? AND child_id = ?", today, child.ID).First(&record)
	record.DeletedAt = gorm.DeletedAt{}

	record.EventDate = today
	record.ChildID = int(child.ID)
	record.FirstName = child.FirstName
	record.LastName = child.LastName
	record.Birthdate = child.Birthdate
	record.GroupID = membership.GroupID
	record.GroupName = membership.GroupName
	record.ParentID = claims.ParentID
	record.Status = localdb.StatusPending
	record.IsGuest = child.IsGuest

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

// ── Guest management ──────────────────────────────────────────────────────

// guestCTIDOffset is added to a guest's local gorm_id to produce a synthetic CTID
// that is guaranteed to never collide with real ChurchTools IDs (which are <1B).
const guestCTIDOffset = 1_000_000_000

type guestRequest struct {
	Parent struct {
		FirstName string `json:"firstName"`
		LastName  string `json:"lastName"`
		Sex       string `json:"sex"`
		Mobile    string `json:"mobile"`
	} `json:"parent"`
	Children []struct {
		FirstName string `json:"firstName"`
		LastName  string `json:"lastName"`
		Birthdate string `json:"birthdate"`
		GroupID   int    `json:"groupId"`
		GroupName string `json:"groupName"`
	} `json:"children"`
}

// CreateGuest creates a new guest family (parent + children) in the local DB.
// POST /api/admin/guests
func (h *Handler) CreateGuest(w http.ResponseWriter, r *http.Request) {
	var req guestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if req.Parent.FirstName == "" || req.Parent.LastName == "" {
		http.Error(w, "parent name required", http.StatusBadRequest)
		return
	}

	// Create parent (step 1: CTID=0, step 2: update with guestCTIDOffset + ID).
	parent := localdb.SyncedPerson{
		FirstName: req.Parent.FirstName,
		LastName:  req.Parent.LastName,
		Sex:       req.Parent.Sex,
		Mobile:    req.Parent.Mobile,
		IsParent:  true,
		IsGuest:   true,
	}
	if err := h.db.Create(&parent).Error; err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	parent.CTID = guestCTIDOffset + int(parent.ID)
	if err := h.db.Save(&parent).Error; err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Create each child and link to parent.
	for _, c := range req.Children {
		child := localdb.SyncedPerson{
			FirstName: c.FirstName,
			LastName:  c.LastName,
			Birthdate: c.Birthdate,
			IsChild:   true,
			IsGuest:   true,
		}
		if err := h.db.Create(&child).Error; err != nil {
			http.Error(w, "db error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		child.CTID = guestCTIDOffset + int(child.ID)
		if err := h.db.Save(&child).Error; err != nil {
			http.Error(w, "db error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if c.GroupID != 0 {
			h.db.Create(&localdb.SyncedGroupMembership{
				PersonCTID: child.CTID,
				GroupID:    c.GroupID,
				GroupName:  c.GroupName,
			})
		}
		h.db.Create(&localdb.SyncedRelationship{
			ParentCTID: parent.CTID,
			ChildCTID:  child.CTID,
		})
	}

	writeJSON(w, http.StatusCreated, map[string]any{"id": parent.ID})
}

// UpdateGuest updates an existing guest family.
// PUT /api/admin/guests/{id}
func (h *Handler) UpdateGuest(w http.ResponseWriter, r *http.Request) {
	parentID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var req guestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	var parent localdb.SyncedPerson
	if err := h.db.First(&parent, parentID).Error; err != nil || !parent.IsGuest {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	parent.FirstName = req.Parent.FirstName
	parent.LastName = req.Parent.LastName
	parent.Sex = req.Parent.Sex
	parent.Mobile = req.Parent.Mobile
	if err := h.db.Save(&parent).Error; err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Load existing children ordered by ChildCTID (= creation order).
	var rels []localdb.SyncedRelationship
	h.db.Where("parent_ct_id = ?", parent.CTID).Find(&rels)
	sort.Slice(rels, func(i, j int) bool { return rels[i].ChildCTID < rels[j].ChildCTID })

	// Update existing children in-place (preserving CheckIn status) or create new ones.
	for i, c := range req.Children {
		if i < len(rels) {
			// Update the existing child in-place so any active CheckIn record is preserved.
			var child localdb.SyncedPerson
			h.db.Unscoped().Where("ct_id = ?", rels[i].ChildCTID).Limit(1).Find(&child)
			if child.ID == 0 {
				http.Error(w, "db error: child not found", http.StatusInternalServerError)
				return
			}
			child.FirstName = c.FirstName
			child.LastName = c.LastName
			child.Birthdate = c.Birthdate
			if err := h.db.Save(&child).Error; err != nil {
				http.Error(w, "db error: "+err.Error(), http.StatusInternalServerError)
				return
			}
			// Update group membership.
			h.db.Where("person_ct_id = ?", child.CTID).Delete(&localdb.SyncedGroupMembership{})
			if c.GroupID != 0 {
				h.db.Create(&localdb.SyncedGroupMembership{
					PersonCTID: child.CTID,
					GroupID:    c.GroupID,
					GroupName:  c.GroupName,
				})
			}
			// Sync cached name/group fields into today's CheckIn record if present.
			h.db.Model(&localdb.CheckIn{}).
				Where("child_id = ? AND event_date = ?", child.ID, localdb.Today()).
				Updates(map[string]any{
					"first_name": c.FirstName,
					"last_name":  c.LastName,
					"birthdate":  c.Birthdate,
					"group_id":   c.GroupID,
					"group_name": c.GroupName,
				})
		} else {
			// Create a new child.
			child := localdb.SyncedPerson{
				FirstName: c.FirstName,
				LastName:  c.LastName,
				Birthdate: c.Birthdate,
				IsChild:   true,
				IsGuest:   true,
			}
			if err := h.db.Create(&child).Error; err != nil {
				http.Error(w, "db error: "+err.Error(), http.StatusInternalServerError)
				return
			}
			child.CTID = guestCTIDOffset + int(child.ID)
			if err := h.db.Save(&child).Error; err != nil {
				http.Error(w, "db error: "+err.Error(), http.StatusInternalServerError)
				return
			}
			if c.GroupID != 0 {
				h.db.Create(&localdb.SyncedGroupMembership{
					PersonCTID: child.CTID,
					GroupID:    c.GroupID,
					GroupName:  c.GroupName,
				})
			}
			h.db.Create(&localdb.SyncedRelationship{
				ParentCTID: parent.CTID,
				ChildCTID:  child.CTID,
			})
		}
	}

	// Delete excess old children (when the new list is shorter).
	for i := len(req.Children); i < len(rels); i++ {
		var oldChild localdb.SyncedPerson
		h.db.Unscoped().Where("ct_id = ?", rels[i].ChildCTID).Limit(1).Find(&oldChild)
		if oldChild.ID != 0 {
			h.db.Where("child_id = ?", oldChild.ID).Delete(&localdb.CheckIn{})
		}
		h.db.Where("person_ct_id = ?", rels[i].ChildCTID).Delete(&localdb.SyncedGroupMembership{})
		h.db.Unscoped().Where("ct_id = ?", rels[i].ChildCTID).Delete(&localdb.SyncedPerson{})
		h.db.Where("parent_ct_id = ? AND child_ct_id = ?", parent.CTID, rels[i].ChildCTID).Delete(&localdb.SyncedRelationship{})
	}

	w.WriteHeader(http.StatusNoContent)
}

// DeleteGuest removes a guest family entirely from the local DB.
// DELETE /api/admin/guests/{id}
func (h *Handler) DeleteGuest(w http.ResponseWriter, r *http.Request) {
	parentID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var parent localdb.SyncedPerson
	if err := h.db.First(&parent, parentID).Error; err != nil || !parent.IsGuest {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	var rels []localdb.SyncedRelationship
	h.db.Where("parent_ct_id = ?", parent.CTID).Find(&rels)
	for _, rel := range rels {
		h.db.Where("person_ct_id = ?", rel.ChildCTID).Delete(&localdb.SyncedGroupMembership{})
		h.db.Unscoped().Where("ct_id = ?", rel.ChildCTID).Delete(&localdb.SyncedPerson{})
	}
	h.db.Where("parent_ct_id = ?", parent.CTID).Delete(&localdb.SyncedRelationship{})
	h.db.Unscoped().Delete(&parent)

	w.WriteHeader(http.StatusNoContent)
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
		h.db.Where("id IN ?", parentIDs).Find(&parents)
		for _, p := range parents {
			parentMap[int(p.ID)] = p
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
