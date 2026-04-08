package db

import "gorm.io/gorm"

// Setting is a key-value store for application state (e.g. "last_ct_sync").
type Setting struct {
	Key   string `gorm:"primaryKey"`
	Value string
}

// SyncedPerson is a person record synced from ChurchTools.
// It is updated on every sync; never written to CT.
type SyncedPerson struct {
	gorm.Model
	CTID        int `gorm:"uniqueIndex;column:ct_id"`
	FirstName   string
	LastName    string
	Birthdate   string
	Email       string
	PhoneNumber string
	Mobile      string
	Sex         string // "male", "female", or ""
	IsChild     bool
	IsParent    bool
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
