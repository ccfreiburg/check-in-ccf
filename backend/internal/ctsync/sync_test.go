package ctsync_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ccf/check-in/backend/internal/ct"
	"github.com/ccf/check-in/backend/internal/ctsync"
	localdb "github.com/ccf/check-in/backend/internal/db"
	"gorm.io/gorm"
)

func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := localdb.Open(":memory:")
	if err != nil {
		t.Fatalf("open in-memory db: %v", err)
	}
	return db
}

func wj(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	b, _ := json.Marshal(v)
	w.Write(b)
}

// ── Groups / IsStale / LastSync ──────────────────────────────────────────

func TestGroups_ReturnsConfigured(t *testing.T) {
	db := openTestDB(t)
	groups := []ctsync.GroupConfig{{ID: 1, Name: "Gruppe A"}, {ID: 2, Name: "Gruppe B"}}
	svc := ctsync.New(nil, db, groups, nil)
	got := svc.Groups()
	if len(got) != 2 {
		t.Errorf("expected 2 groups, got %d", len(got))
	}
	if got[0].ID != 1 || got[0].Name != "Gruppe A" {
		t.Errorf("unexpected group[0]: %+v", got[0])
	}
}

func TestIsStale_NeverSynced_ReturnsTrue(t *testing.T) {
	db := openTestDB(t)
	svc := ctsync.New(nil, db, nil, nil)
	if !svc.IsStale() {
		t.Error("expected IsStale=true when never synced")
	}
}

func TestLastSync_NeverSynced_ReturnsZeroTime(t *testing.T) {
	db := openTestDB(t)
	svc := ctsync.New(nil, db, nil, nil)
	if !svc.LastSync().IsZero() {
		t.Error("expected zero time when never synced")
	}
}

func TestIsStale_RecentSync_ReturnsFalse(t *testing.T) {
	db := openTestDB(t)
	db.Save(&localdb.Setting{Key: "last_ct_sync", Value: time.Now().UTC().Format(time.RFC3339)})
	svc := ctsync.New(nil, db, nil, nil)
	if svc.IsStale() {
		t.Error("expected IsStale=false for recent sync")
	}
}

func TestLastSync_AfterInsert_ReturnsNonZero(t *testing.T) {
	db := openTestDB(t)
	syncTime := time.Now().Add(-1 * time.Hour).UTC().Truncate(time.Second)
	db.Save(&localdb.Setting{Key: "last_ct_sync", Value: syncTime.Format(time.RFC3339)})
	svc := ctsync.New(nil, db, nil, nil)
	got := svc.LastSync()
	if got.IsZero() {
		t.Error("expected non-zero time")
	}
}

// ── Run – concurrent lock ────────────────────────────────────────────────

func TestRun_AlreadyRunning_ReturnsError(t *testing.T) {
	db := openTestDB(t)
	block := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/person/masterdata" {
			<-block
			wj(w, map[string]any{"data": map[string]any{"sexes": []any{}}})
			return
		}
		wj(w, map[string]any{"data": []any{}, "meta": map[string]any{"pagination": map[string]any{"lastPage": 1}}})
	}))
	defer func() { close(block); srv.Close() }()

	ctClient := ct.NewClient(srv.URL, "tok")
	svc := ctsync.New(ctClient, db, []ctsync.GroupConfig{{ID: 1, Name: "G"}}, nil)

	started := make(chan struct{})
	go func() {
		close(started)
		svc.Run(context.Background()) //nolint:errcheck
	}()
	<-started
	time.Sleep(20 * time.Millisecond)

	err := svc.Run(context.Background())
	if err == nil || !strings.Contains(err.Error(), "in progress") {
		t.Errorf("expected 'in progress' error, got: %v", err)
	}
}

// ── Run – full sync with mock CT ─────────────────────────────────────────

func TestRun_FullSync_PopulatesDB(t *testing.T) {
	db := openTestDB(t)
	mux := http.NewServeMux()

	mux.HandleFunc("/person/masterdata", func(w http.ResponseWriter, r *http.Request) {
		wj(w, map[string]any{
			"data": map[string]any{
				"sexes": []map[string]any{
					{"id": 1, "name": "sex.male"},
					{"id": 2, "name": "sex.female"},
				},
			},
		})
	})

	mux.HandleFunc("/groups/", func(w http.ResponseWriter, r *http.Request) {
		wj(w, map[string]any{
			"data": []map[string]any{{"personId": 100, "groupTypeRoleId": 1}},
			"meta": map[string]any{"pagination": map[string]any{"lastPage": 1}},
		})
	})

	mux.HandleFunc("/persons", func(w http.ResponseWriter, r *http.Request) {
		wj(w, map[string]any{
			"data": []map[string]any{
				{"id": 100, "firstName": "Child", "lastName": "One", "sexId": 1},
				{"id": 200, "firstName": "Parent", "lastName": "One", "sexId": 2},
			},
			"meta": map[string]any{"pagination": map[string]any{"lastPage": 1}},
		})
	})

	mux.HandleFunc("/persons/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/relationships") {
			wj(w, map[string]any{
				"data": []map[string]any{
					{
						"relationshipTypeId":   1,
						"degreeOfRelationship": "relationship.part.parent",
						"relative": map[string]any{
							"domainIdentifier": "200",
							"domainAttributes": map[string]any{"firstName": "Parent", "lastName": "One"},
						},
					},
				},
			})
			return
		}
		wj(w, map[string]any{
			"data": map[string]any{"id": 100, "firstName": "Child", "lastName": "One"},
		})
	})

	mux.HandleFunc("/group/roles", func(w http.ResponseWriter, r *http.Request) {
		wj(w, map[string]any{
			"data": []map[string]any{{"id": 1, "name": "Leiter", "isLeader": true}},
		})
	})

	ctSrv := httptest.NewServer(mux)
	defer ctSrv.Close()

	ctClient := ct.NewClient(ctSrv.URL, "test-token")
	groups := []ctsync.GroupConfig{{ID: 1, Name: "KinderGruppe"}}
	svc := ctsync.New(ctClient, db, groups, nil)

	if err := svc.Run(context.Background()); err != nil {
		t.Fatalf("Run() failed: %v", err)
	}

	var children []localdb.SyncedPerson
	db.Where("is_child = ?", true).Find(&children)
	if len(children) == 0 {
		t.Error("expected at least 1 synced child")
	}

	var parents []localdb.SyncedPerson
	db.Where("is_parent = ?", true).Find(&parents)
	if len(parents) == 0 {
		t.Error("expected at least 1 synced parent")
	}

	var rels []localdb.SyncedRelationship
	db.Find(&rels)
	if len(rels) == 0 {
		t.Error("expected at least 1 synced relationship")
	}

	if svc.IsStale() {
		t.Error("expected IsStale=false after successful sync")
	}

	if svc.LastSync().IsZero() {
		t.Error("expected non-zero LastSync after Run")
	}
}

func TestRun_NoGroups_Succeeds(t *testing.T) {
	db := openTestDB(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/person/masterdata" {
			wj(w, map[string]any{"data": map[string]any{"sexes": []any{}}})
			return
		}
		wj(w, map[string]any{"data": []any{}, "meta": map[string]any{"pagination": map[string]any{"lastPage": 1}}})
	}))
	defer srv.Close()

	ctClient := ct.NewClient(srv.URL, "tok")
	svc := ctsync.New(ctClient, db, []ctsync.GroupConfig{}, nil)
	if err := svc.Run(context.Background()); err != nil {
		t.Errorf("unexpected error for empty groups: %v", err)
	}
}
