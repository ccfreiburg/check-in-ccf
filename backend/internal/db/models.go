package db

import (
	"time"

	"gorm.io/gorm"
)

const (
	StatusPending    = "pending"
	StatusRegistered = "registered"
	StatusCheckedIn  = "checked_in"
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

	// Status is one of: pending | registered | checked_in
	// pending     – parent tapped "Anmelden" in the app; child is on the way
	// registered  – door volunteer confirmed the name tag was handed out
	// checked_in  – group volunteer confirmed the child arrived at the group
	Status         string
	RegisteredAt   *time.Time
	CheckedInAt    *time.Time
	LastNotifiedAt *time.Time
}
