package db

import "gorm.io/gorm"

// Setting is a key-value store for application state (e.g. "last_ct_sync").
type Setting struct {
	Key   string `gorm:"primaryKey"`
	Value string
}

// SyncedPerson is a person record synced from ChurchTools (or manually created for guests).
// It is updated on every CT sync for non-guest rows; guest rows (IsGuest=true) are never
// touched by CT sync.
// For guests, CTID = guestCTIDOffset + local gorm ID, guaranteeing no collision with real CT IDs.
type SyncedPerson struct {
	gorm.Model
	CTID        int `gorm:"index;column:ct_id"`
	FirstName   string
	LastName    string
	Birthdate   string
	Email       string
	PhoneNumber string
	Mobile      string
	Sex         string // "male", "female", or ""
	IsChild     bool
	IsParent    bool
	IsGuest     bool // true for manually registered guest families (not in ChurchTools)
}

// SyncedGroupMembership links a child to a ChurchTools group.
// No soft-delete: records are hard-deleted and recreated on each sync.
type SyncedGroupMembership struct {
	ID         uint `gorm:"primaryKey;autoIncrement"`
	PersonCTID int  `gorm:"index;column:person_ct_id"`
	GroupID    int
	GroupName  string
}

// SyncedRelationship stores a parent→child link from ChurchTools.
// No soft-delete; composite primary key.
type SyncedRelationship struct {
	ParentCTID int `gorm:"primaryKey;autoIncrement:false;column:parent_ct_id"`
	ChildCTID  int `gorm:"primaryKey;autoIncrement:false;column:child_ct_id"`
}

// SyncedStaff stores persons that have a volunteer or admin role derived from
// their ChurchTools group memberships. Rebuilt on every sync.
// Role is "admin" (Leiter/Diakon or admin-group member)
// or "volunteer" (Co-Leiter of a child group).
type SyncedStaff struct {
	gorm.Model
	CTID      int `gorm:"uniqueIndex;column:ct_id"`
	FirstName string
	LastName  string
	Email     string `gorm:"index"`
	Role      string // "admin" or "volunteer"
}
