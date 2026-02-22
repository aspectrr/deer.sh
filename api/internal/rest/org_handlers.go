package rest

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/aspectrr/fluid.sh/api/internal/auth"
	serverError "github.com/aspectrr/fluid.sh/api/internal/error"
	"github.com/aspectrr/fluid.sh/api/internal/id"
	serverJSON "github.com/aspectrr/fluid.sh/api/internal/json"
	"github.com/aspectrr/fluid.sh/api/internal/store"
)

var slugRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,48}[a-z0-9]$`)

const maxSlugLen = 50

func generateSlugSuffix() string {
	b := make([]byte, 2)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func isDuplicateSlugErr(err error) bool {
	return errors.Is(err, store.ErrAlreadyExists) || strings.Contains(err.Error(), "duplicate key")
}

// --- Create Org ---

type createOrgRequest struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type orgResponse struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Slug             string `json:"slug"`
	OwnerID          string `json:"owner_id"`
	StripeCustomerID string `json:"stripe_customer_id,omitempty"`
	CreatedAt        string `json:"created_at"`
}

func toOrgResponse(o *store.Organization) *orgResponse {
	return &orgResponse{
		ID:        o.ID,
		Name:      o.Name,
		Slug:      o.Slug,
		OwnerID:   o.OwnerID,
		CreatedAt: o.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func toOrgResponseForOwner(o *store.Organization) *orgResponse {
	r := toOrgResponse(o)
	r.StripeCustomerID = o.StripeCustomerID
	return r
}

// handleCreateOrg godoc
// @Summary      Create organization
// @Description  Create a new organization and add the current user as owner
// @Tags         Organizations
// @Accept       json
// @Produce      json
// @Param        request  body      createOrgRequest  true  "Organization details"
// @Success      201      {object}  orgResponse
// @Failure      400      {object}  error.ErrorResponse
// @Failure      409      {object}  error.ErrorResponse
// @Failure      500      {object}  error.ErrorResponse
// @Security     CookieAuth
// @Router       /v1/orgs [post]
func (s *Server) handleCreateOrg(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())

	var req createOrgRequest
	if err := serverJSON.DecodeJSON(r.Context(), r, &req); err != nil {
		serverError.RespondError(w, http.StatusBadRequest, err)
		return
	}

	if req.Name == "" {
		serverError.RespondError(w, http.StatusBadRequest, fmt.Errorf("name is required"))
		return
	}

	slug := req.Slug
	if slug == "" {
		slug = strings.ToLower(req.Name)
		slug = strings.Map(func(r rune) rune {
			if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
				return r
			}
			if r == ' ' || r == '-' || r == '_' {
				return '-'
			}
			return -1
		}, slug)
		for strings.Contains(slug, "--") {
			slug = strings.ReplaceAll(slug, "--", "-")
		}
		slug = strings.Trim(slug, "-")
	}
	if !slugRegex.MatchString(slug) {
		serverError.RespondError(w, http.StatusBadRequest, fmt.Errorf("slug must be 3-50 lowercase alphanumeric chars and hyphens"))
		return
	}

	autoSlug := req.Slug == ""

	orgID, err := id.Generate("ORG-")
	if err != nil {
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to generate org ID"))
		return
	}

	org := &store.Organization{
		ID:      orgID,
		Name:    req.Name,
		Slug:    slug,
		OwnerID: user.ID,
	}

	baseSlug := slug
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			if !autoSlug {
				break
			}
			suffix := generateSlugSuffix()
			// Truncate base so base-XXXX fits within maxSlugLen
			maxBase := maxSlugLen - 1 - len(suffix) // 1 for the hyphen
			b := baseSlug
			if len(b) > maxBase {
				b = strings.TrimRight(b[:maxBase], "-")
			}
			slug = b + "-" + suffix
			org.Slug = slug
		}

		err = s.store.WithTx(r.Context(), func(tx store.DataStore) error {
			if err := tx.CreateOrganization(r.Context(), org); err != nil {
				return err
			}
			memberID, err := id.Generate("MBR-")
			if err != nil {
				return fmt.Errorf("generate member ID: %w", err)
			}
			member := &store.OrgMember{
				ID:     memberID,
				OrgID:  org.ID,
				UserID: user.ID,
				Role:   store.OrgRoleOwner,
			}
			return tx.CreateOrgMember(r.Context(), member)
		})
		if err == nil {
			break
		}
		if !isDuplicateSlugErr(err) {
			break
		}
	}

	if err != nil {
		if isDuplicateSlugErr(err) {
			serverError.RespondErrorMsg(w, http.StatusConflict, "organization slug already taken", err)
			return
		}
		serverError.RespondErrorMsg(w, http.StatusInternalServerError, "failed to create organization", err)
		return
	}

	s.telemetry.Track(user.ID, "org_created", map[string]any{"org_slug": slug})

	_ = serverJSON.RespondJSON(w, http.StatusCreated, toOrgResponseForOwner(org))
}

// --- List Orgs ---

// handleListOrgs godoc
// @Summary      List organizations
// @Description  List all organizations the current user belongs to
// @Tags         Organizations
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Failure      500  {object}  error.ErrorResponse
// @Security     CookieAuth
// @Router       /v1/orgs [get]
func (s *Server) handleListOrgs(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	orgs, err := s.store.ListOrganizationsByUser(r.Context(), user.ID)
	if err != nil {
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to list organizations"))
		return
	}

	result := make([]*orgResponse, 0, len(orgs))
	for _, o := range orgs {
		result = append(result, toOrgResponse(o))
	}

	_ = serverJSON.RespondJSON(w, http.StatusOK, map[string]any{
		"organizations": result,
		"total":         len(result),
	})
}

// --- Get Org ---

// handleGetOrg godoc
// @Summary      Get organization
// @Description  Get organization details by slug
// @Tags         Organizations
// @Produce      json
// @Param        slug  path      string  true  "Organization slug"
// @Success      200   {object}  orgResponse
// @Failure      403   {object}  error.ErrorResponse
// @Failure      404   {object}  error.ErrorResponse
// @Failure      500   {object}  error.ErrorResponse
// @Security     CookieAuth
// @Router       /v1/orgs/{slug} [get]
func (s *Server) handleGetOrg(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	user := auth.UserFromContext(r.Context())

	org, err := s.store.GetOrganizationBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			serverError.RespondError(w, http.StatusNotFound, fmt.Errorf("organization not found"))
			return
		}
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to get organization"))
		return
	}

	// Check membership
	member, err := s.store.GetOrgMember(r.Context(), org.ID, user.ID)
	if err != nil {
		serverError.RespondError(w, http.StatusForbidden, fmt.Errorf("not a member of this organization"))
		return
	}

	if member.Role == store.OrgRoleOwner {
		_ = serverJSON.RespondJSON(w, http.StatusOK, toOrgResponseForOwner(org))
	} else {
		_ = serverJSON.RespondJSON(w, http.StatusOK, toOrgResponse(org))
	}
}

// --- Update Org ---

type updateOrgRequest struct {
	Name *string `json:"name,omitempty"`
}

// handleUpdateOrg godoc
// @Summary      Update organization
// @Description  Update organization details (owner or admin only)
// @Tags         Organizations
// @Accept       json
// @Produce      json
// @Param        slug     path      string            true  "Organization slug"
// @Param        request  body      updateOrgRequest  true  "Fields to update"
// @Success      200      {object}  orgResponse
// @Failure      400      {object}  error.ErrorResponse
// @Failure      403      {object}  error.ErrorResponse
// @Failure      404      {object}  error.ErrorResponse
// @Failure      500      {object}  error.ErrorResponse
// @Security     CookieAuth
// @Router       /v1/orgs/{slug} [patch]
func (s *Server) handleUpdateOrg(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	user := auth.UserFromContext(r.Context())

	org, err := s.store.GetOrganizationBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			serverError.RespondError(w, http.StatusNotFound, fmt.Errorf("organization not found"))
			return
		}
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to get organization"))
		return
	}

	member, err := s.store.GetOrgMember(r.Context(), org.ID, user.ID)
	if err != nil || (member.Role != store.OrgRoleOwner && member.Role != store.OrgRoleAdmin) {
		serverError.RespondError(w, http.StatusForbidden, fmt.Errorf("insufficient permissions"))
		return
	}

	var req updateOrgRequest
	if err := serverJSON.DecodeJSON(r.Context(), r, &req); err != nil {
		serverError.RespondError(w, http.StatusBadRequest, err)
		return
	}

	if req.Name != nil {
		org.Name = *req.Name
	}

	if err := s.store.UpdateOrganization(r.Context(), org); err != nil {
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to update organization"))
		return
	}

	_ = serverJSON.RespondJSON(w, http.StatusOK, toOrgResponse(org))
}

// --- Delete Org ---

// handleDeleteOrg godoc
// @Summary      Delete organization
// @Description  Delete an organization (owner only)
// @Tags         Organizations
// @Produce      json
// @Param        slug  path      string  true  "Organization slug"
// @Success      200   {object}  map[string]string
// @Failure      403   {object}  error.ErrorResponse
// @Failure      404   {object}  error.ErrorResponse
// @Failure      500   {object}  error.ErrorResponse
// @Security     CookieAuth
// @Router       /v1/orgs/{slug} [delete]
func (s *Server) handleDeleteOrg(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	user := auth.UserFromContext(r.Context())

	org, err := s.store.GetOrganizationBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			serverError.RespondError(w, http.StatusNotFound, fmt.Errorf("organization not found"))
			return
		}
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to get organization"))
		return
	}

	if org.OwnerID != user.ID {
		serverError.RespondError(w, http.StatusForbidden, fmt.Errorf("only the owner can delete an organization"))
		return
	}

	if err := s.store.DeleteOrganization(r.Context(), org.ID); err != nil {
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to delete organization"))
		return
	}

	_ = serverJSON.RespondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// --- Members ---

type memberResponse struct {
	ID        string `json:"id"`
	UserID    string `json:"user_id"`
	Role      string `json:"role"`
	CreatedAt string `json:"created_at"`
}

// handleListMembers godoc
// @Summary      List members
// @Description  List all members of an organization
// @Tags         Members
// @Produce      json
// @Param        slug  path      string  true  "Organization slug"
// @Success      200   {object}  map[string]interface{}
// @Failure      403   {object}  error.ErrorResponse
// @Failure      404   {object}  error.ErrorResponse
// @Failure      500   {object}  error.ErrorResponse
// @Security     CookieAuth
// @Router       /v1/orgs/{slug}/members [get]
func (s *Server) handleListMembers(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	user := auth.UserFromContext(r.Context())

	org, err := s.store.GetOrganizationBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			serverError.RespondError(w, http.StatusNotFound, fmt.Errorf("organization not found"))
			return
		}
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to get organization"))
		return
	}

	_, err = s.store.GetOrgMember(r.Context(), org.ID, user.ID)
	if err != nil {
		serverError.RespondError(w, http.StatusForbidden, fmt.Errorf("not a member of this organization"))
		return
	}

	members, err := s.store.ListOrgMembers(r.Context(), org.ID)
	if err != nil {
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to list members"))
		return
	}

	result := make([]*memberResponse, 0, len(members))
	for _, m := range members {
		result = append(result, &memberResponse{
			ID:        m.ID,
			UserID:    m.UserID,
			Role:      string(m.Role),
			CreatedAt: m.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	_ = serverJSON.RespondJSON(w, http.StatusOK, map[string]any{
		"members": result,
		"total":   len(result),
	})
}

type addMemberRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

// handleAddMember godoc
// @Summary      Add member
// @Description  Add a user to an organization (owner or admin only)
// @Tags         Members
// @Accept       json
// @Produce      json
// @Param        slug     path      string            true  "Organization slug"
// @Param        request  body      addMemberRequest  true  "Member details"
// @Success      201      {object}  memberResponse
// @Failure      400      {object}  error.ErrorResponse
// @Failure      403      {object}  error.ErrorResponse
// @Failure      404      {object}  error.ErrorResponse
// @Failure      409      {object}  error.ErrorResponse
// @Failure      500      {object}  error.ErrorResponse
// @Security     CookieAuth
// @Router       /v1/orgs/{slug}/members [post]
func (s *Server) handleAddMember(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	user := auth.UserFromContext(r.Context())

	org, err := s.store.GetOrganizationBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			serverError.RespondError(w, http.StatusNotFound, fmt.Errorf("organization not found"))
			return
		}
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to get organization"))
		return
	}

	member, err := s.store.GetOrgMember(r.Context(), org.ID, user.ID)
	if err != nil || (member.Role != store.OrgRoleOwner && member.Role != store.OrgRoleAdmin) {
		serverError.RespondError(w, http.StatusForbidden, fmt.Errorf("insufficient permissions"))
		return
	}

	var req addMemberRequest
	if err := serverJSON.DecodeJSON(r.Context(), r, &req); err != nil {
		serverError.RespondError(w, http.StatusBadRequest, err)
		return
	}

	if req.Email == "" {
		serverError.RespondError(w, http.StatusBadRequest, fmt.Errorf("email is required"))
		return
	}

	role := store.OrgRole(req.Role)
	if role == "" {
		role = store.OrgRoleMember
	}
	if role == store.OrgRoleOwner {
		serverError.RespondError(w, http.StatusBadRequest, fmt.Errorf("cannot add another owner"))
		return
	}
	if role != store.OrgRoleMember && role != store.OrgRoleAdmin {
		serverError.RespondError(w, http.StatusBadRequest, fmt.Errorf("invalid role: must be member or admin"))
		return
	}

	targetUser, err := s.store.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			serverError.RespondError(w, http.StatusNotFound, fmt.Errorf("user not found"))
			return
		}
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to look up user"))
		return
	}

	memberID, err := id.Generate("MBR-")
	if err != nil {
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to generate member ID"))
		return
	}

	newMember := &store.OrgMember{
		ID:     memberID,
		OrgID:  org.ID,
		UserID: targetUser.ID,
		Role:   role,
	}

	if err := s.store.CreateOrgMember(r.Context(), newMember); err != nil {
		if errors.Is(err, store.ErrAlreadyExists) {
			serverError.RespondError(w, http.StatusConflict, fmt.Errorf("user is already a member"))
			return
		}
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to add member"))
		return
	}

	_ = serverJSON.RespondJSON(w, http.StatusCreated, &memberResponse{
		ID:        newMember.ID,
		UserID:    newMember.UserID,
		Role:      string(newMember.Role),
		CreatedAt: newMember.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

// handleRemoveMember godoc
// @Summary      Remove member
// @Description  Remove a member from an organization (owner or admin only)
// @Tags         Members
// @Produce      json
// @Param        slug      path      string  true  "Organization slug"
// @Param        memberID  path      string  true  "Member ID"
// @Success      200       {object}  map[string]string
// @Failure      403       {object}  error.ErrorResponse
// @Failure      404       {object}  error.ErrorResponse
// @Failure      500       {object}  error.ErrorResponse
// @Security     CookieAuth
// @Router       /v1/orgs/{slug}/members/{memberID} [delete]
func (s *Server) handleRemoveMember(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	memberID := chi.URLParam(r, "memberID")
	user := auth.UserFromContext(r.Context())

	org, err := s.store.GetOrganizationBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			serverError.RespondError(w, http.StatusNotFound, fmt.Errorf("organization not found"))
			return
		}
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to get organization"))
		return
	}

	callerMember, err := s.store.GetOrgMember(r.Context(), org.ID, user.ID)
	if err != nil || (callerMember.Role != store.OrgRoleOwner && callerMember.Role != store.OrgRoleAdmin) {
		serverError.RespondError(w, http.StatusForbidden, fmt.Errorf("insufficient permissions"))
		return
	}

	// Prevent removing the org owner
	targetMember, err := s.store.GetOrgMemberByID(r.Context(), org.ID, memberID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			serverError.RespondError(w, http.StatusNotFound, fmt.Errorf("member not found"))
			return
		}
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to get member"))
		return
	}
	if targetMember.Role == store.OrgRoleOwner {
		serverError.RespondError(w, http.StatusForbidden, fmt.Errorf("cannot remove the organization owner"))
		return
	}

	if err := s.store.DeleteOrgMember(r.Context(), org.ID, memberID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			serverError.RespondError(w, http.StatusNotFound, fmt.Errorf("member not found"))
			return
		}
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to remove member"))
		return
	}

	_ = serverJSON.RespondJSON(w, http.StatusOK, map[string]string{"status": "removed"})
}
