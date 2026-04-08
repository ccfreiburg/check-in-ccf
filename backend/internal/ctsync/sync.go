package ctsync

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/ccf/check-in/backend/internal/ct"
	localdb "github.com/ccf/check-in/backend/internal/db"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	settingLastSync = "last_ct_sync"
	staleDuration   = 12 * time.Hour
	relConcurrency  = 10 // max parallel relationship API calls
)

// GroupConfig identifies a ChurchTools group whose members are treated as children.
type GroupConfig struct {
	ID   int
	Name string
}

// Service syncs ChurchTools data into the local database.
// It only ever reads from CT; it never writes back to CT.
type Service struct {
	ct            *ct.Client
	db            *gorm.DB
	groups        []GroupConfig
	adminGroupIDs []int
	mu            sync.Mutex
	running       bool
}

// New creates a new sync Service.
func New(ctClient *ct.Client, db *gorm.DB, groups []GroupConfig, adminGroupIDs []int) *Service {
	return &Service{ct: ctClient, db: db, groups: groups, adminGroupIDs: adminGroupIDs}
}

// Groups returns the configured group list.
func (s *Service) Groups() []GroupConfig {
	return s.groups
}

// IsStale reports whether a new sync is due (never synced, or last sync > 12 h ago).
func (s *Service) IsStale() bool {
	var setting localdb.Setting
	if err := s.db.First(&setting, "key = ?", settingLastSync).Error; err != nil {
		return true
	}
	t, err := time.Parse(time.RFC3339, setting.Value)
	if err != nil {
		return true
	}
	return time.Since(t) > staleDuration
}

// LastSync returns the time of the last successful sync, or zero if never synced.
func (s *Service) LastSync() time.Time {
	var setting localdb.Setting
	if err := s.db.First(&setting, "key = ?", settingLastSync).Error; err != nil {
		return time.Time{}
	}
	t, _ := time.Parse(time.RFC3339, setting.Value)
	return t
}

// Run performs a full CT -> local DB sync. Only reads from CT, never writes.
// Returns an error if a sync is already in progress.
//
// API call budget:
//   - 1 per configured group  (member list)
//   - 1-2 bulk calls for all child persons
//   - 1 per child in parallel (relationships) — bounded by relConcurrency
//   - 1-2 bulk calls for all parent persons
func (s *Service) Run(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("sync already in progress")
	}
	s.running = true
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
	}()

	slog.Info("CT sync: starting", "groups", len(s.groups))
	start := time.Now()
	// ── Step 0: fetch sex mappings ────────────────────────────────────────────
	sexMap, err := s.ct.GetSexes()
	if err != nil {
		slog.Warn("CT sync: could not fetch sexes, sex field will be empty", "err", err)
		sexMap = map[int]string{}
	}
	// ── Step 1: collect child IDs per group ───────────────────────────────
	type memberEntry struct {
		groupID   int
		groupName string
	}
	childGroups := map[int][]memberEntry{} // childCTID → memberships
	for _, g := range s.groups {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		ids, err := s.ct.GetGroupMemberIDs(g.ID)
		if err != nil {
			return fmt.Errorf("fetch group %d members: %w", g.ID, err)
		}
		slog.Info("CT sync: group members", "groupId", g.ID, "name", g.Name, "count", len(ids))
		for _, id := range ids {
			childGroups[id] = append(childGroups[id], memberEntry{g.ID, g.Name})
		}
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}

	// ── Step 2: bulk fetch all child persons ──────────────────────────────
	childIDs := make([]int, 0, len(childGroups))
	for id := range childGroups {
		childIDs = append(childIDs, id)
	}
	childPersons, err := s.ct.GetPersonsBulk(childIDs)
	if err != nil {
		return fmt.Errorf("bulk fetch children: %w", err)
	}
	slog.Info("CT sync: fetched child persons", "requested", len(childIDs), "received", len(childPersons))
	if ctx.Err() != nil {
		return ctx.Err()
	}

	// ── Step 3: fetch relationships for all children in parallel ──────────
	type relResult struct {
		childID int
		pids    []int
		err     error
	}

	sem := make(chan struct{}, relConcurrency)
	results := make(chan relResult, len(childIDs))
	var wg sync.WaitGroup

	for _, childID := range childIDs {
		if ctx.Err() != nil {
			break
		}
		wg.Add(1)
		go func(cid int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			rels, err := s.ct.GetRelationships(cid)
			if err != nil {
				results <- relResult{childID: cid, err: err}
				return
			}
			var pids []int
			for _, rel := range rels {
				if rel.RelationshipTypeID == 1 && rel.DegreeOfRelationship == "relationship.part.parent" {
					id := 0
					fmt.Sscanf(rel.Relative.DomainIdentifier, "%d", &id)
					if id != 0 {
						pids = append(pids, id)
					}
				}
			}
			results <- relResult{childID: cid, pids: pids}
		}(childID)
	}
	wg.Wait()
	close(results)

	if ctx.Err() != nil {
		return ctx.Err()
	}

	childToParents := map[int][]int{}
	parentIDSet := map[int]struct{}{}
	for res := range results {
		if res.err != nil {
			slog.Warn("CT sync: relationships error", "childId", res.childID, "err", res.err)
			continue
		}
		childToParents[res.childID] = res.pids
		for _, pid := range res.pids {
			parentIDSet[pid] = struct{}{}
		}
	}

	// ── Step 4: bulk fetch all parent persons ─────────────────────────────
	parentIDs := make([]int, 0, len(parentIDSet))
	for id := range parentIDSet {
		parentIDs = append(parentIDs, id)
	}
	parentPersons, err := s.ct.GetPersonsBulk(parentIDs)
	if err != nil {
		return fmt.Errorf("bulk fetch parents: %w", err)
	}
	slog.Info("CT sync: fetched parent persons", "requested", len(parentIDs), "received", len(parentPersons))
	if ctx.Err() != nil {
		return ctx.Err()
	}

	// ── Step 5: upsert children into DB ───────────────────────────────────
	for childID, memberships := range childGroups {
		p, ok := childPersons[childID]
		if !ok {
			slog.Warn("CT sync: child person missing from bulk response", "ctId", childID)
			continue
		}
		s.savePerson(p, true, false, sexMap)
		s.db.Where("person_ct_id = ?", childID).Delete(&localdb.SyncedGroupMembership{})
		for _, m := range memberships {
			s.db.Create(&localdb.SyncedGroupMembership{
				PersonCTID: childID,
				GroupID:    m.groupID,
				GroupName:  m.groupName,
			})
		}
	}

	// ── Step 6: upsert parents into DB ────────────────────────────────────
	for _, p := range parentPersons {
		s.savePerson(p, false, true, sexMap)
	}

	// ── Step 7: upsert relationships ──────────────────────────────────────
	for childID, pids := range childToParents {
		for _, parentID := range pids {
			s.db.Clauses(clause.OnConflict{DoNothing: true}).
				Create(&localdb.SyncedRelationship{
					ParentCTID: parentID,
					ChildCTID:  childID,
				})
		}
	}

	// ── Step 8: save last-sync timestamp ──────────────────────────────────
	s.db.Save(&localdb.Setting{Key: settingLastSync, Value: start.UTC().Format(time.RFC3339)})
	// ── Step 9: sync staff roles from group memberships ────────────────────────
	if err := s.syncStaff(ctx); err != nil {
		slog.Warn("CT sync: staff sync failed (non-fatal)", "err", err)
	}
	slog.Info("CT sync: completed",
		"children", len(childGroups),
		"parents", len(parentIDs),
		"duration", time.Since(start).Round(time.Millisecond),
	)
	return nil
}

// savePerson upserts a person into the local DB.
// Role flags (IsChild / IsParent) are set additively.
func (s *Service) savePerson(p ct.Person, isChild, isParent bool, sexMap map[int]string) {
	sex := sexMap[p.SexID]
	if sex == "" && p.SexID != 0 {
		sex = "female"
	}
	s.db.Transaction(func(tx *gorm.DB) error { //nolint:errcheck
		var existing localdb.SyncedPerson
		tx.Where("ct_id = ?", p.ID).Limit(1).Find(&existing)
		if existing.ID != 0 {
			existing.FirstName = p.FirstName
			existing.LastName = p.LastName
			existing.Birthdate = p.Birthdate
			existing.Email = p.Email
			existing.PhoneNumber = p.PhoneNumber
			existing.Mobile = p.Mobile
			existing.Sex = sex
			existing.IsChild = existing.IsChild || isChild
			existing.IsParent = existing.IsParent || isParent
			return tx.Save(&existing).Error
		}
		return tx.Create(&localdb.SyncedPerson{
			CTID:        p.ID,
			FirstName:   p.FirstName,
			LastName:    p.LastName,
			Birthdate:   p.Birthdate,
			Email:       p.Email,
			PhoneNumber: p.PhoneNumber,
			Mobile:      p.Mobile,
			Sex:         sex,
			IsChild:     isChild,
			IsParent:    isParent,
		}).Error
	})
}

// syncStaff builds the SyncedStaff table from CT group memberships:
//   - All members of adminGroupIDs → role "admin"
//   - Leaders/Deacons of child groups     → role "admin"
//   - Co-Leaders of child groups          → role "volunteer"
func (s *Service) syncStaff(ctx context.Context) error {
	if len(s.adminGroupIDs) == 0 && len(s.groups) == 0 {
		return nil
	}

	// Fetch and classify group member types (Leader, Co-Leader, Deacon, …).
	memberTypes, err := s.ct.GetGroupMemberTypes()
	if err != nil {
		return fmt.Errorf("get group member types: %w", err)
	}

	// staffRoles: ctPersonID → highest-privilege role found so far.
	staffRoles := map[int]string{}
	promote := func(id int, role string) {
		if cur, ok := staffRoles[id]; !ok || (cur == "volunteer" && role == "admin") {
			staffRoles[id] = role
		}
	}

	// Admin groups: every member becomes an admin.
	for _, gid := range s.adminGroupIDs {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		members, err := s.ct.GetGroupMembersWithTypes(gid, memberTypes)
		if err != nil {
			slog.Warn("CT sync: admin group members error", "groupId", gid, "err", err)
			continue
		}
		for _, m := range members {
			promote(m.PersonID, "admin")
		}
	}

	// Child groups: classify by member type.
	for _, g := range s.groups {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		members, err := s.ct.GetGroupMembersWithTypes(g.ID, memberTypes)
		if err != nil {
			slog.Warn("CT sync: child group members role error", "groupId", g.ID, "err", err)
			continue
		}
		for _, m := range members {
			switch m.TypeName {
			case "leader", "deacon":
				promote(m.PersonID, "admin")
			case "coleader":
				promote(m.PersonID, "volunteer")
				// regular members: not staff
			}
		}
	}

	if len(staffRoles) == 0 {
		s.db.Where("1 = 1").Delete(&localdb.SyncedStaff{})
		return nil
	}

	// Bulk fetch person details for all staff.
	staffIDs := make([]int, 0, len(staffRoles))
	for id := range staffRoles {
		staffIDs = append(staffIDs, id)
	}
	persons, err := s.ct.GetPersonsBulk(staffIDs)
	if err != nil {
		return fmt.Errorf("bulk fetch staff persons: %w", err)
	}

	// Upsert each staff member.
	for ctID, role := range staffRoles {
		p, ok := persons[ctID]
		if !ok {
			slog.Warn("CT sync: staff person missing from bulk response", "ctId", ctID)
			continue
		}
		s.db.Transaction(func(tx *gorm.DB) error { //nolint:errcheck
			var existing localdb.SyncedStaff
			tx.Where("ct_id = ?", ctID).Limit(1).Find(&existing)
			if existing.ID != 0 {
				existing.FirstName = p.FirstName
				existing.LastName = p.LastName
				existing.Email = p.Email
				existing.Role = role
				return tx.Save(&existing).Error
			}
			return tx.Create(&localdb.SyncedStaff{
				CTID:      ctID,
				FirstName: p.FirstName,
				LastName:  p.LastName,
				Email:     p.Email,
				Role:      role,
			}).Error
		})
	}

	// Remove entries no longer in any configured group.
	s.db.Where("ct_id NOT IN ?", staffIDs).Delete(&localdb.SyncedStaff{})

	slog.Info("CT sync: staff synced", "count", len(staffRoles))
	return nil
}
