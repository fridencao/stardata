package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/fridencao/stardata/admin/database"
	"github.com/fridencao/stardata/admin/server/auth"
	"github.com/fridencao/stardata/runtime/pkg/httputil"
	"gopkg.in/yaml.v3"
)

// dataRequestsVirtualPath is the virtual file that stores the shared data-request backlog.
// It is stored in the "dev" environment so dev (Studio) deployments see it in their repo
// (mounted at "/__virtual__/requests.yaml"), while prod deployments and publishes are unaffected.
const dataRequestsVirtualPath = "requests.yaml"

// dataRequestsEnvironment is the virtual file environment for the data-request backlog.
const dataRequestsEnvironment = "dev"

// maxDataRequestBodySize limits the JSON body size for data-request submissions.
const maxDataRequestBodySize = 32 << 10 // 32kb

// dataRequestItem mirrors the RequestItem model in web-common (features/chat/requests/requests-file.ts).
// The YAML layout (top-level "requests:" list) must stay compatible with the frontend parser.
type dataRequestItem struct {
	Question  string `yaml:"question" json:"question"`
	Note      string `yaml:"note,omitempty" json:"note,omitempty"`
	CreatedAt string `yaml:"created_at" json:"created_at"`
	Status    string `yaml:"status" json:"status"`
}

// dataRequestsDoc is the YAML document layout of requests.yaml.
type dataRequestsDoc struct {
	Requests []dataRequestItem `yaml:"requests"`
}

// dataRequestsForOrgAndProject serves the data-request backlog endpoint (StarData):
//   - POST: business users submit a data requirement from the portal chat (requires ReadProject).
//   - GET: technical users list the backlog in Studio (requires ManageProject).
//   - PUT: technical users replace the backlog, e.g. marking items done (requires ManageProject).
//
// Requests are persisted as a virtual file in the "dev" environment, so submissions work even
// when the viewer has no runtime repo permissions and no dev deployment is running.
func (s *Server) dataRequestsForOrgAndProject(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	org := r.PathValue("org")
	project := r.PathValue("project")

	proj, err := s.admin.DB.FindProjectByName(ctx, org, project)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return httputil.Errorf(http.StatusNotFound, "project not found")
		}
		return httputil.Error(http.StatusInternalServerError, err)
	}

	claims := auth.GetClaims(ctx)
	perms := claims.ProjectPermissions(ctx, proj.OrganizationID, proj.ID)

	switch r.Method {
	case http.MethodPost:
		if !perms.ReadProject {
			return httputil.Errorf(http.StatusForbidden, "does not have permission to submit data requests")
		}
		return s.submitDataRequest(w, r, proj.ID)
	case http.MethodGet:
		if !perms.ManageProject {
			return httputil.Errorf(http.StatusForbidden, "does not have permission to list data requests")
		}
		items, err := s.readDataRequests(r, proj.ID)
		if err != nil {
			return err
		}
		return writeJSON(w, map[string]any{"requests": items})
	case http.MethodPut:
		if !perms.ManageProject {
			return httputil.Errorf(http.StatusForbidden, "does not have permission to update data requests")
		}
		return s.replaceDataRequests(w, r, proj.ID)
	default:
		return httputil.Errorf(http.StatusMethodNotAllowed, "method %s not allowed", r.Method)
	}
}

// submitDataRequest appends one open request to the backlog virtual file.
func (s *Server) submitDataRequest(w http.ResponseWriter, r *http.Request, projectID string) error {
	var body struct {
		Question string `json:"question"`
		Note     string `json:"note"`
	}
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxDataRequestBodySize))
	if err := dec.Decode(&body); err != nil {
		return httputil.Errorf(http.StatusBadRequest, "invalid request body: %s", err.Error())
	}

	question := strings.TrimSpace(body.Question)
	if question == "" {
		return httputil.Errorf(http.StatusBadRequest, "question is required")
	}

	items, err := s.readDataRequests(r, projectID)
	if err != nil {
		return err
	}

	items = append(items, dataRequestItem{
		Question:  encodeHTMLText(question),
		Note:      encodeHTMLText(strings.TrimSpace(body.Note)),
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		Status:    "open",
	})

	data, err := yaml.Marshal(dataRequestsDoc{Requests: items})
	if err != nil {
		return httputil.Error(http.StatusInternalServerError, err)
	}
	if len(data) > 1<<17 { // Virtual file limit is 128kb
		return httputil.Errorf(http.StatusRequestEntityTooLarge, "data request backlog is full, resolve existing requests first")
	}

	err = s.admin.DB.UpsertVirtualFile(r.Context(), &database.InsertVirtualFileOptions{
		ProjectID:   projectID,
		Environment: dataRequestsEnvironment,
		Path:        dataRequestsVirtualPath,
		Data:        data,
	})
	if err != nil {
		return httputil.Error(http.StatusInternalServerError, err)
	}

	w.WriteHeader(http.StatusNoContent)
	return nil
}

// replaceDataRequests overwrites the backlog with the provided list (Studio "mark done" flow).
// Items are round-tripped from GET, so they are already HTML-encoded — no re-encoding here.
func (s *Server) replaceDataRequests(w http.ResponseWriter, r *http.Request, projectID string) error {
	var body struct {
		Requests []dataRequestItem `json:"requests"`
	}
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<17))
	if err := dec.Decode(&body); err != nil {
		return httputil.Errorf(http.StatusBadRequest, "invalid request body: %s", err.Error())
	}

	items := body.Requests
	if items == nil {
		items = []dataRequestItem{}
	}
	for i := range items {
		if strings.TrimSpace(items[i].Question) == "" {
			return httputil.Errorf(http.StatusBadRequest, "requests[%d].question is required", i)
		}
		if items[i].Status != "open" && items[i].Status != "done" {
			return httputil.Errorf(http.StatusBadRequest, "requests[%d].status must be %q or %q", i, "open", "done")
		}
	}

	data, err := yaml.Marshal(dataRequestsDoc{Requests: items})
	if err != nil {
		return httputil.Error(http.StatusInternalServerError, err)
	}
	if len(data) > 1<<17 { // Virtual file limit is 128kb
		return httputil.Errorf(http.StatusRequestEntityTooLarge, "data request backlog is too large")
	}

	err = s.admin.DB.UpsertVirtualFile(r.Context(), &database.InsertVirtualFileOptions{
		ProjectID:   projectID,
		Environment: dataRequestsEnvironment,
		Path:        dataRequestsVirtualPath,
		Data:        data,
	})
	if err != nil {
		return httputil.Error(http.StatusInternalServerError, err)
	}

	w.WriteHeader(http.StatusNoContent)
	return nil
}

// readDataRequests loads the current backlog from the virtual file. A missing or deleted file yields an empty list.
func (s *Server) readDataRequests(r *http.Request, projectID string) ([]dataRequestItem, error) {
	vf, err := s.admin.DB.FindVirtualFile(r.Context(), projectID, dataRequestsEnvironment, dataRequestsVirtualPath)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return []dataRequestItem{}, nil
		}
		return nil, httputil.Error(http.StatusInternalServerError, err)
	}
	if vf.Deleted {
		return []dataRequestItem{}, nil
	}

	var doc dataRequestsDoc
	if err := yaml.Unmarshal(vf.Data, &doc); err != nil {
		// Corrupt file: start over rather than blocking submissions.
		return []dataRequestItem{}, nil
	}
	if doc.Requests == nil {
		return []dataRequestItem{}, nil
	}
	return doc.Requests, nil
}

// encodeHTMLText mirrors the HTML encoding in the frontend requests-file.ts (stored XSS defense).
var htmlTextReplacer = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;")

func encodeHTMLText(s string) string {
	return htmlTextReplacer.Replace(s)
}

// writeJSON writes a JSON response body.
func writeJSON(w http.ResponseWriter, v any) error {
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(v)
}
