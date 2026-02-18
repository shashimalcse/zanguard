package api

import (
	"io"
	"net/http"
	"strconv"

	"zanguard/pkg/model"
	"zanguard/pkg/schema"
)

// ── Health ───────────────────────────────────────────────────────────────────

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ── Tenants ──────────────────────────────────────────────────────────────────

// POST /api/v1/tenants
func (s *Server) handleCreateTenant(w http.ResponseWriter, r *http.Request) {
	var req CreateTenantRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	if req.ID == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}
	if req.SchemaMode == "" {
		req.SchemaMode = model.SchemaOwn
	}

	t, err := s.mgr.Create(r.Context(), req.ID, req.DisplayName, req.SchemaMode)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Persist optional parent / shared-schema fields
	if req.ParentTenantID != "" || req.SharedSchemaRef != "" {
		t.ParentTenantID = req.ParentTenantID
		t.SharedSchemaRef = req.SharedSchemaRef
		_ = s.store.UpdateTenant(r.Context(), t)
	}

	writeJSON(w, http.StatusCreated, t)
}

// GET /api/v1/tenants
func (s *Server) handleListTenants(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter := &model.TenantFilter{}
	if v := q.Get("status"); v != "" {
		filter.Status = model.TenantStatus(v)
	}
	if v := q.Get("parent_id"); v != "" {
		filter.ParentID = v
	}
	if v := q.Get("limit"); v != "" {
		filter.Limit, _ = strconv.Atoi(v)
	}
	if v := q.Get("offset"); v != "" {
		filter.Offset, _ = strconv.Atoi(v)
	}

	tenants, err := s.mgr.List(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, ListTenantsResponse{Tenants: tenants, Count: len(tenants)})
}

// GET /api/v1/tenants/{tenantID}
func (s *Server) handleGetTenant(w http.ResponseWriter, r *http.Request) {
	t, err := s.mgr.Get(r.Context(), r.PathValue("tenantID"))
	if err != nil {
		writeError(w, errStatus(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, t)
}

// DELETE /api/v1/tenants/{tenantID}
func (s *Server) handleDeleteTenant(w http.ResponseWriter, r *http.Request) {
	if err := s.mgr.Delete(r.Context(), r.PathValue("tenantID")); err != nil {
		writeError(w, errStatus(err), err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// POST /api/v1/tenants/{tenantID}/activate
func (s *Server) handleActivateTenant(w http.ResponseWriter, r *http.Request) {
	tenantID := r.PathValue("tenantID")
	if err := s.mgr.Activate(r.Context(), tenantID); err != nil {
		writeError(w, errStatus(err), err.Error())
		return
	}
	t, _ := s.mgr.Get(r.Context(), tenantID)
	writeJSON(w, http.StatusOK, t)
}

// POST /api/v1/tenants/{tenantID}/suspend
func (s *Server) handleSuspendTenant(w http.ResponseWriter, r *http.Request) {
	tenantID := r.PathValue("tenantID")
	if err := s.mgr.Suspend(r.Context(), tenantID); err != nil {
		writeError(w, errStatus(err), err.Error())
		return
	}
	t, _ := s.mgr.Get(r.Context(), tenantID)
	writeJSON(w, http.StatusOK, t)
}

// ── Schema ───────────────────────────────────────────────────────────────────

// PUT /api/v1/tenants/{tenantID}/schema   (body = raw YAML)
func (s *Server) handleLoadSchema(w http.ResponseWriter, r *http.Request) {
	tenantID := r.PathValue("tenantID")

	if _, err := s.mgr.Get(r.Context(), tenantID); err != nil {
		writeError(w, errStatus(err), err.Error())
		return
	}

	data, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read body: "+err.Error())
		return
	}

	raw, err := schema.Parse(data)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "parse error: "+err.Error())
		return
	}

	cs, err := schema.Compile(raw, data)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "compile error: "+err.Error())
		return
	}

	if errs := schema.Validate(cs); len(errs) > 0 {
		msgs := make([]string, len(errs))
		for i, e := range errs {
			msgs[i] = e.Error()
		}
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
			"error":   "schema validation failed",
			"details": msgs,
		})
		return
	}

	s.eng.LoadSchema(tenantID, cs)

	s.schemasMu.Lock()
	s.schemas[tenantID] = data
	s.compiled[tenantID] = cs
	s.schemasMu.Unlock()

	writeJSON(w, http.StatusOK, SchemaResponse{
		TenantID:   tenantID,
		Hash:       cs.Hash,
		Version:    cs.Version,
		Source:     string(data),
		CompiledAt: cs.CompiledAt.Format("2006-01-02T15:04:05Z"),
	})
}

// GET /api/v1/tenants/{tenantID}/schema
func (s *Server) handleGetSchema(w http.ResponseWriter, r *http.Request) {
	tenantID := r.PathValue("tenantID")

	s.schemasMu.RLock()
	data, ok := s.schemas[tenantID]
	cs := s.compiled[tenantID]
	s.schemasMu.RUnlock()

	if !ok {
		writeError(w, http.StatusNotFound, "no schema loaded for tenant "+tenantID)
		return
	}

	writeJSON(w, http.StatusOK, SchemaResponse{
		TenantID:   tenantID,
		Hash:       cs.Hash,
		Version:    cs.Version,
		Source:     string(data),
		CompiledAt: cs.CompiledAt.Format("2006-01-02T15:04:05Z"),
	})
}

// ── Tuples ───────────────────────────────────────────────────────────────────

// POST /api/v1/tuples
func (s *Server) handleWriteTuple(w http.ResponseWriter, r *http.Request) {
	tCtx, err := s.tenantCtxFromHeader(r.Context(), r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	var req TupleRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}

	if err := s.store.WriteTuple(tCtx, tupleFromRequest(req)); err != nil {
		writeError(w, errStatus(err), err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"status": "ok"})
}

// POST /api/v1/tuples/batch
func (s *Server) handleWriteTuples(w http.ResponseWriter, r *http.Request) {
	tCtx, err := s.tenantCtxFromHeader(r.Context(), r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	var req BatchTuplesRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}

	tuples := make([]*model.RelationTuple, len(req.Tuples))
	for i, t := range req.Tuples {
		tuples[i] = tupleFromRequest(t)
	}

	if err := s.store.WriteTuples(tCtx, tuples); err != nil {
		writeError(w, errStatus(err), err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"status": "ok", "count": len(tuples)})
}

// DELETE /api/v1/tuples   (body identifies the tuple to delete)
func (s *Server) handleDeleteTuple(w http.ResponseWriter, r *http.Request) {
	tCtx, err := s.tenantCtxFromHeader(r.Context(), r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	var req TupleRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}

	if err := s.store.DeleteTuple(tCtx, tupleFromRequest(req)); err != nil {
		writeError(w, errStatus(err), err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GET /api/v1/tuples?object_type=&object_id=&relation=&subject_type=&subject_id=&subject_relation=
func (s *Server) handleReadTuples(w http.ResponseWriter, r *http.Request) {
	tCtx, err := s.tenantCtxFromHeader(r.Context(), r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	q := r.URL.Query()
	filter := &model.TupleFilter{
		ObjectType:      q.Get("object_type"),
		ObjectID:        q.Get("object_id"),
		Relation:        q.Get("relation"),
		SubjectType:     q.Get("subject_type"),
		SubjectID:       q.Get("subject_id"),
		SubjectRelation: q.Get("subject_relation"),
	}

	tuples, err := s.store.ReadTuples(tCtx, filter)
	if err != nil {
		writeError(w, errStatus(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, TuplesResponse{Tuples: tuples, Count: len(tuples)})
}

// ── Attributes ───────────────────────────────────────────────────────────────

// PUT /api/v1/attributes/objects/{type}/{id}
func (s *Server) handleSetObjectAttributes(w http.ResponseWriter, r *http.Request) {
	tCtx, err := s.tenantCtxFromHeader(r.Context(), r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	var req AttributesRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}

	objType, objID := r.PathValue("type"), r.PathValue("id")
	if err := s.store.SetObjectAttributes(tCtx, objType, objID, req.Attributes); err != nil {
		writeError(w, errStatus(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, AttributesResponse{Attributes: req.Attributes})
}

// GET /api/v1/attributes/objects/{type}/{id}
func (s *Server) handleGetObjectAttributes(w http.ResponseWriter, r *http.Request) {
	tCtx, err := s.tenantCtxFromHeader(r.Context(), r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	attrs, err := s.store.GetObjectAttributes(tCtx, r.PathValue("type"), r.PathValue("id"))
	if err != nil {
		writeError(w, errStatus(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, AttributesResponse{Attributes: attrs})
}

// PUT /api/v1/attributes/subjects/{type}/{id}
func (s *Server) handleSetSubjectAttributes(w http.ResponseWriter, r *http.Request) {
	tCtx, err := s.tenantCtxFromHeader(r.Context(), r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	var req AttributesRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}

	subType, subID := r.PathValue("type"), r.PathValue("id")
	if err := s.store.SetSubjectAttributes(tCtx, subType, subID, req.Attributes); err != nil {
		writeError(w, errStatus(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, AttributesResponse{Attributes: req.Attributes})
}

// GET /api/v1/attributes/subjects/{type}/{id}
func (s *Server) handleGetSubjectAttributes(w http.ResponseWriter, r *http.Request) {
	tCtx, err := s.tenantCtxFromHeader(r.Context(), r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	attrs, err := s.store.GetSubjectAttributes(tCtx, r.PathValue("type"), r.PathValue("id"))
	if err != nil {
		writeError(w, errStatus(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, AttributesResponse{Attributes: attrs})
}

// ── Changelog ────────────────────────────────────────────────────────────────

// GET /api/v1/changelog?since_seq=0&limit=100
func (s *Server) handleReadChangelog(w http.ResponseWriter, r *http.Request) {
	tCtx, err := s.tenantCtxFromHeader(r.Context(), r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	q := r.URL.Query()
	var sinceSeq uint64
	limit := 100
	if v := q.Get("since_seq"); v != "" {
		sinceSeq, _ = strconv.ParseUint(v, 10, 64)
	}
	if v := q.Get("limit"); v != "" {
		if n, _ := strconv.Atoi(v); n > 0 {
			limit = n
		}
	}

	entries, err := s.store.ReadChangelog(tCtx, sinceSeq, limit)
	if err != nil {
		writeError(w, errStatus(err), err.Error())
		return
	}
	latest, _ := s.store.LatestSequence(tCtx)
	writeJSON(w, http.StatusOK, ChangelogResponse{
		Entries:        entries,
		Count:          len(entries),
		LatestSequence: latest,
	})
}

// ── Expand ───────────────────────────────────────────────────────────────────

// POST /api/v1/expand
func (s *Server) handleExpand(w http.ResponseWriter, r *http.Request) {
	tCtx, err := s.tenantCtxFromHeader(r.Context(), r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	var req ExpandRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}

	tree, err := s.eng.Expand(tCtx, req.ObjectType, req.ObjectID, req.Relation)
	if err != nil {
		writeError(w, errStatus(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, tree)
}
