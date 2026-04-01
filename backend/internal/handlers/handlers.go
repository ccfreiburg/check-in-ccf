package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"

	"github.com/ccf/check-in/backend/internal/auth"
	"github.com/ccf/check-in/backend/internal/ct"
	"github.com/go-chi/chi/v5"
	qrcode "github.com/skip2/go-qrcode"
)

type Handler struct {
	ct            *ct.Client
	jwtSecret     []byte
	frontendBase  string
	adminPassword string
}

func New(ctClient *ct.Client, jwtSecret []byte, frontendBase string) *Handler {
	return &Handler{
		ct:            ctClient,
		jwtSecret:     jwtSecret,
		frontendBase:  frontendBase,
		adminPassword: os.Getenv("ADMIN_PASSWORD"),
	}
}

// AdminLogin exchanges the admin password for a signed JWT.
func (h *Handler) AdminLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Password == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
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
	writeJSON(w, http.StatusOK, map[string]string{"token": token})
}

// ListChildren returns all children from CT.
func (h *Handler) ListChildren(w http.ResponseWriter, r *http.Request) {
	children, err := h.ct.GetChildren()
	if err != nil {
		slog.Warn("ListChildren CT error", "err", err)
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, http.StatusOK, children)
}

// GetParentDetail takes a child's person ID, finds their parent via relationships,
// then returns the parent's details and children.
func (h *Handler) GetParentDetail(w http.ResponseWriter, r *http.Request) {
	childID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	// Step 1: find the parent ID(s) for this child
	parentIDs, err := h.ct.GetParentsForChild(childID)
	if err != nil {
		slog.Warn("GetParentDetail: could not get parents", "childId", childID, "err", err)
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	if len(parentIDs) == 0 {
		// Fallback: treat the selected ID itself as the parent
		slog.Warn("GetParentDetail: no parents found, falling back to ID as parent", "id", childID)
		parentIDs = []int{childID}
	}
	parentID := parentIDs[0]

	// Step 2: fetch parent person record
	parent, err := h.ct.GetPerson(parentID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	// Step 3: fetch parent's children
	children, err := h.ct.GetChildrenForParent(parentID)
	if err != nil {
		slog.Warn("GetParentDetail: could not get children", "parentId", parentID, "err", err)
		children = nil
	}
	writeJSON(w, http.StatusOK, map[string]any{"parent": parent, "children": children})
}

// GenerateQR creates a parent JWT and returns a PNG QR code.
// The {id} in the URL may be a child's person ID; we resolve the parent first.
func (h *Handler) GenerateQR(w http.ResponseWriter, r *http.Request) {
	childID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	// Resolve the parent ID for this child
	parentID := childID
	parentIDs, err := h.ct.GetParentsForChild(childID)
	if err == nil && len(parentIDs) > 0 {
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

type childWithStatus struct {
	ct.Child
	CheckedIn bool `json:"checkedIn"`
}

// GetParentCheckinPage returns the parent and their children with check-in status.
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
	var withStatus []childWithStatus
	for _, child := range children {
		status, _ := h.ct.GetCheckInStatus(child.ID, child.GroupID)
		checked := status != nil && status.CheckedIn
		withStatus = append(withStatus, childWithStatus{Child: child, CheckedIn: checked})
	}
	writeJSON(w, http.StatusOK, map[string]any{"parent": parent, "children": withStatus})
}

// CheckIn checks a child in.
func (h *Handler) CheckIn(w http.ResponseWriter, r *http.Request) {
	_, childID, groupID, err := h.resolveParentAction(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := h.ct.CheckIn(childID, groupID); err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"checkedIn": true})
}

// CheckOut checks a child out.
func (h *Handler) CheckOut(w http.ResponseWriter, r *http.Request) {
	_, childID, groupID, err := h.resolveParentAction(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := h.ct.CheckOut(childID, groupID); err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"checkedIn": false})
}

func (h *Handler) resolveParentAction(r *http.Request) (*auth.Claims, int, int, error) {
	tokenStr := chi.URLParam(r, "token")
	claims, err := auth.ParseToken(h.jwtSecret, tokenStr)
	if err != nil || claims.Role != "parent" {
		return nil, 0, 0, fmt.Errorf("invalid or expired token")
	}
	childID, err := strconv.Atoi(chi.URLParam(r, "childId"))
	if err != nil {
		return nil, 0, 0, fmt.Errorf("invalid childId")
	}
	// groupId may be provided as a query param for forward compat;
	// fall back to the hardcoded primary group 599.
	groupID := 599
	if g := r.URL.Query().Get("groupId"); g != "" {
		if parsed, err := strconv.Atoi(g); err == nil && parsed > 0 {
			groupID = parsed
		}
	}
	return claims, childID, groupID, nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
