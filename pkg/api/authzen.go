package api

import (
	"net/http"

	"zanguard/pkg/engine"
)

// handleAuthZenEvaluation implements POST /access/v1/evaluation
// per the AuthZen 1.0 specification.
//
// The tenant is identified by the X-Tenant-ID request header.
// On any evaluation error the response is {"decision": false} with 200 OK,
// as required by the AuthZen spec.
func (s *Server) handleAuthZenEvaluation(w http.ResponseWriter, r *http.Request) {
	tCtx, err := s.tenantCtxFromHeader(r.Context(), r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	var req AuthZenEvaluationRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}

	result, err := s.eng.Check(tCtx, &engine.CheckRequest{
		ObjectType:  req.Resource.Type,
		ObjectID:    req.Resource.ID,
		Permission:  req.Action.Name,
		SubjectType: req.Subject.Type,
		SubjectID:   req.Subject.ID,
		Context:     req.Context,
	})
	if err != nil {
		// Per AuthZen spec: evaluation errors yield decision:false, not an HTTP error
		writeJSON(w, http.StatusOK, AuthZenEvaluationResponse{Decision: false})
		return
	}

	writeJSON(w, http.StatusOK, AuthZenEvaluationResponse{Decision: result.Allowed})
}

// handleAuthZenEvaluations implements POST /access/v1/evaluations
// per the AuthZen 1.0 batch-evaluation specification.
//
// All evaluations share the top-level subject. Per-item context may be provided.
// Individual item errors yield decision:false for that item.
func (s *Server) handleAuthZenEvaluations(w http.ResponseWriter, r *http.Request) {
	tCtx, err := s.tenantCtxFromHeader(r.Context(), r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	var req AuthZenBatchRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}

	results := make([]AuthZenEvaluationResponse, len(req.Evaluations))
	for i, item := range req.Evaluations {
		result, err := s.eng.Check(tCtx, &engine.CheckRequest{
			ObjectType:  item.Resource.Type,
			ObjectID:    item.Resource.ID,
			Permission:  item.Action.Name,
			SubjectType: req.Subject.Type,
			SubjectID:   req.Subject.ID,
			Context:     item.Context,
		})
		if err != nil {
			results[i] = AuthZenEvaluationResponse{Decision: false}
			continue
		}
		results[i] = AuthZenEvaluationResponse{Decision: result.Allowed}
	}

	writeJSON(w, http.StatusOK, AuthZenBatchResponse{Evaluations: results})
}
