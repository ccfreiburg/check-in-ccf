package ct

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

type Client struct {
	baseURL    string
	apiToken   string
	httpClient *http.Client
}

func NewClient(baseURL, apiToken string) *Client {
	return &Client{
		baseURL:    baseURL,
		apiToken:   apiToken,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

type Person struct {
	ID          int    `json:"id"`
	FirstName   string `json:"firstName"`
	LastName    string `json:"lastName"`
	Birthdate   string `json:"birthday,omitempty"`
	Email       string `json:"email"`
	PhoneNumber string `json:"phoneNumber"`
	Mobile      string `json:"mobile,omitempty"`
	SexID       int    `json:"sexId,omitempty"`
}

// GetSexes returns a map of sex ID → short name ("male" or "female").
// Data comes from /person/masterdata under the "sexes" key.
func (c *Client) GetSexes() (map[int]string, error) {
	var resp struct {
		Data struct {
			Sexes []struct {
				ID   int    `json:"id"`
				Name string `json:"name"`
			} `json:"sexes"`
		} `json:"data"`
	}
	if err := c.get("/person/masterdata", &resp); err != nil {
		return nil, err
	}
	m := make(map[int]string, len(resp.Data.Sexes))
	for _, s := range resp.Data.Sexes {
		switch s.Name {
		case "sex.male":
			m[s.ID] = "male"
		case "sex.female":
			m[s.ID] = "female"
		}
	}
	return m, nil
}

type Child struct {
	Person
	GroupID   int    `json:"groupId"`
	GroupName string `json:"groupName"`
}

type CheckInStatus struct {
	ChildID   int  `json:"childId"`
	MeetingID int  `json:"meetingId"`
	CheckedIn bool `json:"checkedIn"`
}

func (c *Client) GetPerson(id int) (*Person, error) {
	var resp struct {
		Data Person `json:"data"`
	}
	if err := c.get(fmt.Sprintf("/persons/%d", id), &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// GetPersonsBulk fetches multiple persons by ID in batches of 50 using the
// /persons?ids[]=... endpoint. Returns a map keyed by CT person ID.
func (c *Client) GetPersonsBulk(ids []int) (map[int]Person, error) {
	const chunkSize = 50
	type listResp struct {
		Data []Person `json:"data"`
	}
	result := make(map[int]Person, len(ids))
	for start := 0; start < len(ids); start += chunkSize {
		end := start + chunkSize
		if end > len(ids) {
			end = len(ids)
		}
		chunk := ids[start:end]
		path := "/persons?page=1&limit=50"
		for _, id := range chunk {
			path += fmt.Sprintf("&ids[]=%d", id)
		}
		// CT may paginate even within our chunk; loop just in case.
		for page := 1; ; page++ {
			// Replace page param on subsequent iterations.
			pagePath := path
			if page > 1 {
				pagePath = fmt.Sprintf("/persons?page=%d&limit=50", page)
				for _, id := range chunk {
					pagePath += fmt.Sprintf("&ids[]=%d", id)
				}
			}
			raw, err := c.getRaw(pagePath)
			if err != nil {
				return nil, fmt.Errorf("GetPersonsBulk page %d: %w", page, err)
			}
			var resp struct {
				Data []Person `json:"data"`
				Meta struct {
					Pagination struct {
						LastPage int `json:"lastPage"`
					} `json:"pagination"`
				} `json:"meta"`
			}
			if err := json.Unmarshal(raw, &resp); err != nil {
				return nil, fmt.Errorf("GetPersonsBulk unmarshal: %w", err)
			}
			for _, p := range resp.Data {
				result[p.ID] = p
			}
			if page >= resp.Meta.Pagination.LastPage || len(resp.Data) == 0 {
				break
			}
		}
	}
	return result, nil
}

// GetRelationships returns the raw relationship list for a person.
// Exported so ctsync can call it directly without re-parsing.
func (c *Client) GetRelationships(personID int) ([]relEntry, error) {
	return c.getRelationships(personID)
}

// Group holds basic group info returned by ChurchTools.
type Group struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// GetGroup returns the group metadata for a given group ID.
func (c *Client) GetGroup(id int) (*Group, error) {
	var resp struct {
		Data Group `json:"data"`
	}
	if err := c.get(fmt.Sprintf("/groups/%d", id), &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// GetGroupMemberIDs returns the CT person IDs of all members in a group.
func (c *Client) GetGroupMemberIDs(groupID int) ([]int, error) {
	type memberPage struct {
		Data []struct {
			PersonID int `json:"personId"`
		} `json:"data"`
		Meta struct {
			Pagination struct {
				LastPage int `json:"lastPage"`
			} `json:"pagination"`
		} `json:"meta"`
	}
	var ids []int
	for page := 1; ; page++ {
		raw, err := c.getRaw(fmt.Sprintf("/groups/%d/members?page=%d&limit=100", groupID, page))
		if err != nil {
			return nil, err
		}
		var resp memberPage
		if err := json.Unmarshal(raw, &resp); err != nil {
			return nil, fmt.Errorf("members unmarshal: %w", err)
		}
		for _, m := range resp.Data {
			ids = append(ids, m.PersonID)
		}
		if page >= resp.Meta.Pagination.LastPage || len(resp.Data) == 0 {
			break
		}
	}
	return ids, nil
}

// RelEntry is a CT relationship list item.
type RelEntry struct {
	RelationshipTypeID   int    `json:"relationshipTypeId"`
	DegreeOfRelationship string `json:"degreeOfRelationship"`
	Relative             struct {
		DomainIdentifier string `json:"domainIdentifier"`
		DomainAttributes struct {
			FirstName string `json:"firstName"`
			LastName  string `json:"lastName"`
		} `json:"domainAttributes"`
	} `json:"relative"`
}

// relEntry is kept as an alias so existing internal callers still compile.
type relEntry = RelEntry

func (c *Client) getRelationships(personID int) ([]relEntry, error) {
	raw, err := c.getRaw(fmt.Sprintf("/persons/%d/relationships", personID))
	if err != nil {
		return nil, err
	}
	slog.Debug("CT relationships raw", "personId", personID, "body", string(raw))
	var resp struct {
		Data []relEntry `json:"data"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("relationships unmarshal: %w", err)
	}
	return resp.Data, nil
}

// GetChildrenForParent returns children linked to a parent via CT relationships.
func (c *Client) GetChildrenForParent(parentID int) ([]Child, error) {
	rels, err := c.getRelationships(parentID)
	if err != nil {
		return nil, err
	}
	children := make([]Child, 0)
	for _, rel := range rels {
		slog.Debug("CT relationship entry", "typeId", rel.RelationshipTypeID, "degree", rel.DegreeOfRelationship, "person", rel.Relative.DomainAttributes.FirstName)
		if rel.RelationshipTypeID == 1 && rel.DegreeOfRelationship == "relationship.part.child" {
			id := 0
			fmt.Sscanf(rel.Relative.DomainIdentifier, "%d", &id)
			if id == 0 {
				continue
			}
			// Fetch full person details to get birthdate.
			person, err := c.GetPerson(id)
			if err != nil || person == nil {
				// Fall back to relationship name data only.
				person = &Person{
					ID:        id,
					FirstName: rel.Relative.DomainAttributes.FirstName,
					LastName:  rel.Relative.DomainAttributes.LastName,
				}
			}
			children = append(children, Child{
				Person:    *person,
				GroupID:   599,
				GroupName: "KinderKirche",
			})
		}
	}
	return children, nil
}

// GetParentsForChild returns parent person IDs linked to a child via CT relationships.
func (c *Client) GetParentsForChild(childID int) ([]int, error) {
	rels, err := c.getRelationships(childID)
	if err != nil {
		return nil, err
	}
	var parentIDs []int
	for _, rel := range rels {
		slog.Debug("CT child relationship entry", "typeId", rel.RelationshipTypeID, "degree", rel.DegreeOfRelationship, "person", rel.Relative.DomainAttributes.FirstName)
		if rel.RelationshipTypeID == 1 && rel.DegreeOfRelationship == "relationship.part.parent" {
			id := 0
			fmt.Sscanf(rel.Relative.DomainIdentifier, "%d", &id)
			if id != 0 {
				parentIDs = append(parentIDs, id)
			}
		}
	}
	return parentIDs, nil
}

// getTodayMeetingID returns the meeting ID for today in the given group.
// If no meeting exists for today, it creates one.
func (c *Client) getTodayMeetingID(groupID int) (int, error) {
	today := time.Now().UTC().Format("2006-01-02")
	tomorrow := time.Now().UTC().AddDate(0, 0, 1).Format("2006-01-02")
	var listResp struct {
		Data []struct {
			ID int `json:"id"`
		} `json:"data"`
	}
	if err := c.get(fmt.Sprintf("/groups/%d/meetings?from=%s&to=%s", groupID, today, tomorrow), &listResp); err != nil {
		return 0, fmt.Errorf("list meetings: %w", err)
	}
	if len(listResp.Data) > 0 {
		// Use the last (most recent) meeting of the day
		return listResp.Data[len(listResp.Data)-1].ID, nil
	}
	// No meeting today — create one
	startDate := time.Now().UTC().Format("2006-01-02T15:04:05Z")
	var createResp struct {
		Data struct {
			ID int `json:"id"`
		} `json:"data"`
	}
	if err := c.post(fmt.Sprintf("/groups/%d/meetings", groupID), map[string]any{"startDate": startDate}, &createResp); err != nil {
		return 0, fmt.Errorf("create meeting: %w", err)
	}
	slog.Info("CT created new meeting for today", "groupId", groupID, "meetingId", createResp.Data.ID)
	return createResp.Data.ID, nil
}

func (c *Client) CheckIn(childID, groupID int) error {
	meetingID, err := c.getTodayMeetingID(groupID)
	if err != nil {
		return fmt.Errorf("getTodayMeeting: %w", err)
	}
	return c.post(fmt.Sprintf("/groups/%d/checkin/%d", groupID, childID), map[string]any{
		"date":           time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		"groupMeetingId": meetingID,
	}, nil)
}

func (c *Client) get(path string, out any) error {
	raw, err := c.getRaw(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, out)
}

func (c *Client) getRaw(path string) ([]byte, error) {
	url := c.baseURL + path
	slog.Debug("CT GET", "url", url)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	c.setHeaders(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		slog.Warn("CT GET failed", "url", url, "err", err)
		return nil, err
	}
	defer resp.Body.Close()
	slog.Debug("CT GET response", "url", url, "status", resp.StatusCode)
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		slog.Warn("CT GET error response", "url", url, "status", resp.StatusCode, "body", string(b))
		return nil, fmt.Errorf("CT API %s: %s", resp.Status, b)
	}
	return b, nil
}

func (c *Client) post(path string, body any, out any) error {
	url := c.baseURL + path
	slog.Debug("CT POST", "url", url)
	pr, pw := io.Pipe()
	go func() {
		_ = json.NewEncoder(pw).Encode(body)
		pw.Close()
	}()
	req, err := http.NewRequest(http.MethodPost, url, pr)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	c.setHeaders(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		slog.Warn("CT POST failed", "url", url, "err", err)
		return err
	}
	defer resp.Body.Close()
	slog.Debug("CT POST response", "url", url, "status", resp.StatusCode)
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		slog.Warn("CT POST error response", "url", url, "status", resp.StatusCode, "body", string(b))
		return fmt.Errorf("CT API %s: %s", resp.Status, b)
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

func (c *Client) delete(path string) error {
	url := c.baseURL + path
	slog.Debug("CT DELETE", "url", url)
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	c.setHeaders(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		slog.Warn("CT DELETE failed", "url", url, "err", err)
		return err
	}
	defer resp.Body.Close()
	slog.Debug("CT DELETE response", "url", url, "status", resp.StatusCode)
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		slog.Warn("CT DELETE error response", "url", url, "status", resp.StatusCode, "body", string(b))
		return fmt.Errorf("CT API %s: %s", resp.Status, b)
	}
	return nil
}

func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Login "+c.apiToken)
	req.Header.Set("Accept", "application/json")
}
