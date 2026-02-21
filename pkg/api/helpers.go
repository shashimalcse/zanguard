package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"zanguard/pkg/model"
	"zanguard/pkg/storage"
	"zanguard/pkg/tenant"
)

const maxTupleTTLSeconds int64 = 86400

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func readJSON(r *http.Request, v any) error {
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return err
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("unexpected trailing JSON value")
		}
		return err
	}
	return nil
}

// tenantCtxFromPath builds a tenant-scoped context from the {tenantID} path segment.
func (s *Server) tenantCtxFromPath(ctx context.Context, r *http.Request) (context.Context, error) {
	tenantID := r.PathValue("tenantID")
	if tenantID == "" {
		return ctx, errors.New("missing tenantID path parameter")
	}
	return tenant.BuildContext(ctx, s.store, tenantID)
}

// tenantCtxFromHeader builds a tenant-scoped context from the X-Tenant-ID header.
func (s *Server) tenantCtxFromHeader(ctx context.Context, r *http.Request) (context.Context, error) {
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		return ctx, errors.New("missing X-Tenant-ID header")
	}
	return tenant.BuildContext(ctx, s.store, tenantID)
}

// errStatus maps well-known storage errors to HTTP status codes.
func errStatus(err error) int {
	switch {
	case errors.Is(err, storage.ErrTenantNotFound), errors.Is(err, storage.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, storage.ErrTenantDeleted):
		return http.StatusGone
	case errors.Is(err, storage.ErrTenantSuspended):
		return http.StatusForbidden
	case errors.Is(err, storage.ErrDuplicateTuple):
		return http.StatusConflict
	case errors.Is(err, storage.ErrQuotaExceeded):
		return http.StatusTooManyRequests
	case errors.Is(err, tenant.ErrInvalidStateTransition):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

// tupleFromRequest converts a TupleRequest into a RelationTuple.
func tupleFromRequest(req TupleRequest) *model.RelationTuple {
	return &model.RelationTuple{
		ObjectType:      req.ObjectType,
		ObjectID:        req.ObjectID,
		Relation:        req.Relation,
		SubjectType:     req.SubjectType,
		SubjectID:       req.SubjectID,
		SubjectRelation: req.SubjectRelation,
		Attributes:      req.Attributes,
	}
}

func tupleFromWriteRequest(req TupleRequest, now time.Time) (*model.RelationTuple, error) {
	tuple := tupleFromRequest(req)
	expiresAt, err := tupleExpiryFromRequest(req, now)
	if err != nil {
		return nil, err
	}
	tuple.ExpiresAt = expiresAt
	return tuple, nil
}

func tupleExpiryFromRequest(req TupleRequest, now time.Time) (*time.Time, error) {
	expiresRaw := strings.TrimSpace(req.ExpiresAt)
	if req.TTLSeconds != nil && expiresRaw != "" {
		return nil, fmt.Errorf("ttl_seconds and expires_at are mutually exclusive")
	}
	if req.TTLSeconds != nil {
		ttl := *req.TTLSeconds
		if ttl <= 0 {
			return nil, fmt.Errorf("ttl_seconds must be greater than 0")
		}
		if ttl > maxTupleTTLSeconds {
			return nil, fmt.Errorf("ttl_seconds must be <= %d", maxTupleTTLSeconds)
		}
		expiresAt := now.UTC().Add(time.Duration(ttl) * time.Second)
		return &expiresAt, nil
	}
	if expiresRaw == "" {
		return nil, nil
	}
	expiresAt, err := time.Parse(time.RFC3339, expiresRaw)
	if err != nil {
		return nil, fmt.Errorf("expires_at must be RFC3339: %w", err)
	}
	expiresUTC := expiresAt.UTC()
	return &expiresUTC, nil
}
