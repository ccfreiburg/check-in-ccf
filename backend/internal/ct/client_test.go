package ct_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ccf/check-in/backend/internal/ct"
)

func newTestClient(baseURL string) *ct.Client {
	return ct.NewClient(baseURL, "test-token")
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	b, _ := json.Marshal(v)
	w.Write(b)
}

func TestGetPerson_ReturnsPersonData(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || !strings.HasPrefix(r.URL.Path, "/persons/") {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, map[string]any{
			"data": map[string]any{
				"id": 42, "firstName": "Max", "lastName": "Mustermann",
				"email": "max@example.com", "phoneNumber": "0123",
			},
		})
	}))
	defer srv.Close()
	client := newTestClient(srv.URL)
	p, err := client.GetPerson(42)
	if err != nil {
		t.Fatal(err)
	}
	if p.ID != 42 || p.FirstName != "Max" {
		t.Errorf("unexpected person: %+v", p)
	}
}

func TestGetPerson_ServerError_ReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer srv.Close()
	client := newTestClient(srv.URL)
	_, err := client.GetPerson(1)
	if err == nil {
		t.Error("expected error for 500 response")
	}
}

func TestGetGroup_ReturnsGroup(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{
			"data": map[string]any{"id": 10, "name": "TestGroup"},
		})
	}))
	defer srv.Close()
	client := newTestClient(srv.URL)
	g, err := client.GetGroup(10)
	if err != nil {
		t.Fatal(err)
	}
	if g.ID != 10 || g.Name != "TestGroup" {
		t.Errorf("unexpected group: %+v", g)
	}
}

func TestGetGroup_NetworkError_ReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close()
	client := ct.NewClient(fmt.Sprintf("http://localhost:%d", 1), "token")
	_, err := client.GetGroup(1)
	if err == nil {
		t.Error("expected error for closed server")
	}
}

func TestGetGroupMemberIDs_ReturnsMemberList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{
			"data": []map[string]any{
				{"personId": 1}, {"personId": 2}, {"personId": 3},
			},
			"meta": map[string]any{"pagination": map[string]any{"lastPage": 1}},
		})
	}))
	defer srv.Close()
	client := newTestClient(srv.URL)
	ids, err := client.GetGroupMemberIDs(5)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 3 {
		t.Errorf("expected 3 members, got %d", len(ids))
	}
}

func TestGetGroupMemberIDs_Empty_ReturnsEmptySlice(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{
			"data": []any{},
			"meta": map[string]any{"pagination": map[string]any{"lastPage": 1}},
		})
	}))
	defer srv.Close()
	client := newTestClient(srv.URL)
	ids, err := client.GetGroupMemberIDs(5)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 0 {
		t.Errorf("expected empty slice, got %d", len(ids))
	}
}

func TestGetGroupMemberTypes_ClassifiesRoles(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/group/roles" {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, map[string]any{
			"data": []map[string]any{
				{"id": 1, "name": "Leiter", "isLeader": true},
				{"id": 2, "name": "Co-Leiter", "isLeader": true},
				{"id": 3, "name": "Teilnehmer", "isLeader": false},
			},
		})
	}))
	defer srv.Close()
	client := newTestClient(srv.URL)
	types, err := client.GetGroupMemberTypes()
	if err != nil {
		t.Fatal(err)
	}
	if types[1] != "leader" {
		t.Errorf("expected Leiter=leader, got %q", types[1])
	}
	if types[2] != "coleader" {
		t.Errorf("expected Co-Leiter=coleader, got %q", types[2])
	}
	if types[3] != "member" {
		t.Errorf("expected Teilnehmer=member, got %q", types[3])
	}
}

func TestGetGroupMemberTypes_ServerError_ReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	}))
	defer srv.Close()
	client := newTestClient(srv.URL)
	_, err := client.GetGroupMemberTypes()
	if err == nil {
		t.Error("expected error for 400 response")
	}
}

func TestGetGroupMembersWithTypes_ReturnsEntries(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{
			"data": []map[string]any{
				{"personId": 10, "groupTypeRoleId": 1},
				{"personId": 20, "groupTypeRoleId": 3},
			},
			"meta": map[string]any{"pagination": map[string]any{"lastPage": 1}},
		})
	}))
	defer srv.Close()
	client := newTestClient(srv.URL)
	typeMap := map[int]string{1: "leader", 3: "member"}
	entries, err := client.GetGroupMembersWithTypes(5, typeMap)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].TypeName != "leader" {
		t.Errorf("expected leader, got %q", entries[0].TypeName)
	}
	if entries[1].TypeName != "member" {
		t.Errorf("expected member, got %q", entries[1].TypeName)
	}
}

func TestGetGroupMembersWithTypes_UnknownRoleID_DefaultsMember(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{
			"data": []map[string]any{
				{"personId": 11, "groupTypeRoleId": 99},
			},
			"meta": map[string]any{"pagination": map[string]any{"lastPage": 1}},
		})
	}))
	defer srv.Close()
	client := newTestClient(srv.URL)
	entries, err := client.GetGroupMembersWithTypes(5, map[int]string{})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].TypeName != "member" {
		t.Errorf("expected default member, got %q", entries[0].TypeName)
	}
}

func TestGetSexes_ReturnsSexMap(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{
			"data": map[string]any{
				"sexes": []map[string]any{
					{"id": 1, "name": "sex.male"},
					{"id": 2, "name": "sex.female"},
				},
			},
		})
	}))
	defer srv.Close()
	client := newTestClient(srv.URL)
	sexes, err := client.GetSexes()
	if err != nil {
		t.Fatal(err)
	}
	if sexes[1] != "male" {
		t.Errorf("expected id 1=male, got %q", sexes[1])
	}
	if sexes[2] != "female" {
		t.Errorf("expected id 2=female, got %q", sexes[2])
	}
}

func TestGetPersonsBulk_ReturnsMap(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{
			"data": []map[string]any{
				{"id": 1, "firstName": "Alice", "lastName": "Smith"},
				{"id": 2, "firstName": "Bob", "lastName": "Jones"},
			},
			"meta": map[string]any{"pagination": map[string]any{"lastPage": 1}},
		})
	}))
	defer srv.Close()
	client := newTestClient(srv.URL)
	persons, err := client.GetPersonsBulk([]int{1, 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(persons) != 2 {
		t.Errorf("expected 2 persons, got %d", len(persons))
	}
	if persons[1].FirstName != "Alice" {
		t.Errorf("expected Alice, got %q", persons[1].FirstName)
	}
}

func TestGetPersonsBulk_Empty_ReturnsEmptyMap(t *testing.T) {
	client := newTestClient("http://localhost:1")
	persons, err := client.GetPersonsBulk([]int{})
	if err != nil {
		t.Fatal(err)
	}
	if len(persons) != 0 {
		t.Errorf("expected empty map for empty input")
	}
}

func TestGetRelationships_ReturnsEntries(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{
			"data": []map[string]any{
				{
					"relationshipTypeId":   1,
					"degreeOfRelationship": "relationship.part.parent",
					"relative": map[string]any{
						"domainIdentifier": "200",
						"domainAttributes": map[string]any{
							"firstName": "Parent",
							"lastName":  "Smith",
						},
					},
				},
			},
		})
	}))
	defer srv.Close()
	client := newTestClient(srv.URL)
	rels, err := client.GetRelationships(100)
	if err != nil {
		t.Fatal(err)
	}
	if len(rels) != 1 {
		t.Fatalf("expected 1 relationship, got %d", len(rels))
	}
	if rels[0].DegreeOfRelationship != "relationship.part.parent" {
		t.Errorf("unexpected degree: %q", rels[0].DegreeOfRelationship)
	}
}

func TestGetParentsForChild_ExtractsParentIDs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{
			"data": []map[string]any{
				{
					"relationshipTypeId":   1,
					"degreeOfRelationship": "relationship.part.parent",
					"relative": map[string]any{
						"domainIdentifier": "999",
						"domainAttributes": map[string]any{"firstName": "P", "lastName": "Parent"},
					},
				},
				{
					"relationshipTypeId":   1,
					"degreeOfRelationship": "relationship.part.child",
					"relative": map[string]any{
						"domainIdentifier": "888",
						"domainAttributes": map[string]any{"firstName": "C", "lastName": "Child"},
					},
				},
			},
		})
	}))
	defer srv.Close()
	client := newTestClient(srv.URL)
	ids, err := client.GetParentsForChild(100)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 1 || ids[0] != 999 {
		t.Errorf("expected [999], got %v", ids)
	}
}

func TestGetChildrenForParent_ExtractsChildrenWithPersonLookup(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/relationships") {
			writeJSON(w, map[string]any{
				"data": []map[string]any{
					{
						"relationshipTypeId":   1,
						"degreeOfRelationship": "relationship.part.child",
						"relative": map[string]any{
							"domainIdentifier": "55",
							"domainAttributes": map[string]any{"firstName": "Kid", "lastName": "Test"},
						},
					},
				},
			})
			return
		}
		writeJSON(w, map[string]any{
			"data": map[string]any{"id": 55, "firstName": "Kid", "lastName": "Test"},
		})
	}))
	defer srv.Close()
	client := newTestClient(srv.URL)
	children, err := client.GetChildrenForParent(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(children))
	}
	if children[0].FirstName != "Kid" {
		t.Errorf("expected firstName=Kid, got %q", children[0].FirstName)
	}
}

func TestLoginUser_WithPersonIDInResponse_ReturnsID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/login" {
			writeJSON(w, map[string]any{
				"data": map[string]any{"userId": 77, "personId": 77},
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()
	client := newTestClient(srv.URL)
	id, err := client.LoginUser("user@example.com", "password")
	if err != nil {
		t.Fatal(err)
	}
	if id != 77 {
		t.Errorf("expected id=77, got %d", id)
	}
}

func TestLoginUser_FallbackToPersonsMe_ReturnsID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login":
			writeJSON(w, map[string]any{"data": map[string]any{}})
		case "/persons/me":
			writeJSON(w, map[string]any{"data": map[string]any{"id": 88}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	client := newTestClient(srv.URL)
	id, err := client.LoginUser("user@example.com", "password")
	if err != nil {
		t.Fatal(err)
	}
	if id != 88 {
		t.Errorf("expected id=88 from /persons/me, got %d", id)
	}
}

func TestLoginUser_BadCredentials_Returns403(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	}))
	defer srv.Close()
	client := newTestClient(srv.URL)
	_, err := client.LoginUser("user@example.com", "wrong")
	if err == nil {
		t.Error("expected error for 401 login response")
	}
}

func TestCheckIn_CreatesAndUsesExistingMeeting(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/meetings") {
			writeJSON(w, map[string]any{
				"data": []map[string]any{{"id": 500}},
			})
			return
		}
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusOK)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()
	client := newTestClient(srv.URL)
	err := client.CheckIn(10, 5)
	if err != nil {
		t.Errorf("unexpected error on CheckIn: %v", err)
	}
}

func TestCheckIn_NoMeeting_CreatesNewMeeting(t *testing.T) {
	meetingCreated := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/meetings") {
			writeJSON(w, map[string]any{"data": []any{}})
			return
		}
		if r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/meetings") {
			meetingCreated = true
			writeJSON(w, map[string]any{"data": map[string]any{"id": 600}})
			return
		}
		if r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/checkin/") {
			w.WriteHeader(http.StatusOK)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()
	client := newTestClient(srv.URL)
	err := client.CheckIn(10, 5)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !meetingCreated {
		t.Error("expected a new meeting to be created when none exists today")
	}
}

func TestGetPerson_404Response_ReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()
	client := newTestClient(srv.URL)
	_, err := client.GetPerson(1)
	if err == nil {
		t.Error("expected error for 404 response")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("expected 404 in error message, got: %v", err)
	}
}
