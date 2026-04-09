package db_test

import (
	"strings"
	"testing"
	"time"

	db "github.com/ccf/check-in/backend/internal/db"
)

func TestToday_FormatsAsYYYYMMDD(t *testing.T) {
	got := db.Today()
	if len(got) != 10 {
		t.Fatalf("expected 10 chars, got %q", got)
	}
	parts := strings.Split(got, "-")
	if len(parts) != 3 || len(parts[0]) != 4 || len(parts[1]) != 2 || len(parts[2]) != 2 {
		t.Errorf("unexpected format: %q", got)
	}
	// Must equal time.Now formatted the same way
	want := time.Now().Format("2006-01-02")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestOpen_InMemory_CreatesAllTables(t *testing.T) {
	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	tables := []struct {
		name  string
		model any
	}{
		{"check_ins", &db.CheckIn{}},
		{"push_subscriptions", &db.PushSubscription{}},
	}
	for _, tc := range tables {
		var count int64
		if err := database.Model(tc.model).Count(&count).Error; err != nil {
			t.Errorf("table %q not accessible: %v", tc.name, err)
		}
	}
}

func TestOpen_InMemory_IndependentInstances(t *testing.T) {
	db1, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db2, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	// Insert into db1; db2 should still be empty.
	record := db.CheckIn{EventDate: "2025-01-01", ChildID: 999, Status: db.StatusPending}
	db1.Create(&record)

	var count1, count2 int64
	db1.Model(&db.CheckIn{}).Count(&count1)
	db2.Model(&db.CheckIn{}).Count(&count2)
	if count1 != 1 {
		t.Errorf("db1 should have 1 record, got %d", count1)
	}
	if count2 != 0 {
		t.Errorf("db2 should have 0 records (independent), got %d", count2)
	}
}

func TestOpen_CheckInStatus_Constants(t *testing.T) {
	if db.StatusPending != "pending" {
		t.Errorf("StatusPending = %q, want %q", db.StatusPending, "pending")
	}
	if db.StatusCheckedIn != "checked_in" {
		t.Errorf("StatusCheckedIn = %q, want %q", db.StatusCheckedIn, "checked_in")
	}
}
