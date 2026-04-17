package db

import (
	"time"

	"gorm.io/gorm"
)

const (
	StatusPending   = "pending"
	StatusCheckedIn = "checked_in"
)

// PushSubscription stores a Web Push subscription for a parent identified by their token.
// Multiple devices (subscriptions) per parent are supported.
type PushSubscription struct {
	gorm.Model
	// ParentID is the ChurchTools person ID of the parent.
	ParentID int    `gorm:"index"`
	Endpoint string `gorm:"uniqueIndex"`
	P256dh   string
	Auth     string
}

// EventStats stores a per-group, per-event attendance snapshot.
// One row per (EventDate, GroupID), populated when EndEvent is called.
type EventStats struct {
	EventDate  string `gorm:"primaryKey;not null"`
	GroupID    int    `gorm:"primaryKey;not null"`
	GroupName  string
	Registered int
	CheckedIn  int
	CheckedOut int
}

// CheckIn tracks a child's arrival and check-in state for a single service day.
// One record per (EventDate, ChildID), created when the parent taps "Anmelden".
type CheckIn struct {
	gorm.Model
	// EventDate is the service date in "YYYY-MM-DD" format.
	EventDate string `gorm:"uniqueIndex:idx_event_child"`
	// ChildID is the ChurchTools person ID.
	ChildID int `gorm:"uniqueIndex:idx_event_child"`

	// Cached from ChurchTools at registration time.
	FirstName string
	LastName  string
	Birthdate string
	GroupID   int
	GroupName string
	ParentID  int

	// Status is one of: pending | checked_in
	// pending    – parent tapped "Anmelden"; child is on the way or not yet group-confirmed
	// checked_in – group volunteer confirmed the child arrived
	Status string
	// TagReceived tracks name-tag handout independently of check-in status.
	TagReceived    bool
	RegisteredAt   *time.Time // set when TagReceived is first toggled true
	CheckedInAt    *time.Time
	CheckedOutAt   *time.Time // set when a volunteer checks the child out (status overridden to "")
	LastNotifiedAt *time.Time
	// IsGuest is true when the child belongs to a manually registered guest family (not in CT).
	// Stored at registration time to avoid a DB lookup in hot paths.
	IsGuest bool
}
