package admin

import (
	"strconv"

	"gopenid/internal/domain"
	"gopenid/internal/httpx"

	"github.com/gofiber/fiber/v3"
)

func (h *Handler) listPolicies(c fiber.Ctx) error {
	rows, err := h.db.ListPolicies(c.Context())
	if err != nil {
		return httpx.Error(c, 500, "list failed")
	}
	return c.JSON(rows)
}

func (h *Handler) createPolicy(c fiber.Ctx) error {
	var req policyInput
	if err := c.Bind().Body(&req); err != nil {
		return httpx.BadRequest(c, "invalid json")
	}
	if msg, ok := req.validate(); !ok {
		return httpx.BadRequest(c, msg)
	}
	row, err := h.db.CreatePolicy(c.Context(), req.toPolicy())
	if err != nil {
		return httpx.BadRequest(c, "policy already exists or invalid")
	}
	return c.Status(201).JSON(row)
}

func (h *Handler) getPolicy(c fiber.Ctx) error {
	row, err := h.db.GetPolicy(c.Context(), idParam(c))
	if err != nil {
		return httpx.NotFound(c)
	}
	return c.JSON(row)
}

func (h *Handler) updatePolicy(c fiber.Ctx) error {
	var req policyInput
	if err := c.Bind().Body(&req); err != nil {
		return httpx.BadRequest(c, "invalid json")
	}
	if msg, ok := req.validate(); !ok {
		return httpx.BadRequest(c, msg)
	}
	row, err := h.db.UpdatePolicy(c.Context(), idParam(c), req.toPolicy())
	if err != nil {
		return httpx.BadRequest(c, "policy already exists or invalid")
	}
	return c.JSON(row)
}

func (h *Handler) deletePolicy(c fiber.Ctx) error {
	if err := h.db.DeletePolicy(c.Context(), idParam(c)); err != nil {
		return httpx.Error(c, 500, "delete failed")
	}
	return c.SendStatus(204)
}

func (h *Handler) listPolicyAssignments(c fiber.Ctx) error {
	rows, err := h.db.ListPolicyAssignments(c.Context(), idParam(c))
	if err != nil {
		return httpx.Error(c, 500, "list failed")
	}
	return c.JSON(rows)
}

func (h *Handler) assignPolicy(c fiber.Ctx) error {
	var req struct {
		SubjectType string `json:"subjectType"`
		SubjectID   int64  `json:"subjectId"`
	}
	if err := c.Bind().Body(&req); err != nil || req.SubjectID == 0 {
		return httpx.BadRequest(c, "subjectType and subjectId are required")
	}
	subject := domain.PolicySubject(req.SubjectType)
	switch subject {
	case domain.PolicySubjectClient, domain.PolicySubjectGroup, domain.PolicySubjectUser:
	default:
		return httpx.BadRequest(c, "subjectType must be client, group or user")
	}
	row, err := h.db.AssignPolicy(c.Context(), idParam(c), subject, req.SubjectID)
	if err != nil {
		return httpx.BadRequest(c, "assignment failed")
	}
	return c.Status(201).JSON(row)
}

func (h *Handler) unassignPolicy(c fiber.Ctx) error {
	assignmentID, _ := strconv.ParseInt(c.Params("assignmentId"), 10, 64)
	if err := h.db.UnassignPolicy(c.Context(), assignmentID); err != nil {
		return httpx.Error(c, 500, "unassign failed")
	}
	return c.SendStatus(204)
}

type policyInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
	Effect      string `json:"effect"`
	IPCIDRs     string `json:"ipCidrs"`
	DaysOfWeek  []int  `json:"daysOfWeek"`
	StartTime   string `json:"startTime"`
	EndTime     string `json:"endTime"`
}

func (req policyInput) validate() (string, bool) {
	if req.Name == "" {
		return "name is required", false
	}
	switch domain.PolicyType(req.Type) {
	case domain.PolicyTypeIP, domain.PolicyTypeTime:
	default:
		return "type must be ip or time", false
	}
	switch domain.PolicyEffect(req.Effect) {
	case domain.PolicyEffectAllow, domain.PolicyEffectDeny:
	default:
		return "effect must be allow or deny", false
	}
	if domain.PolicyType(req.Type) == domain.PolicyTypeIP && req.IPCIDRs == "" {
		return "ipCidrs is required for ip policies", false
	}
	return "", true
}

func (req policyInput) toPolicy() domain.Policy {
	return domain.Policy{
		Name:        req.Name,
		Description: req.Description,
		Type:        domain.PolicyType(req.Type),
		Effect:      domain.PolicyEffect(req.Effect),
		IPCIDRs:     req.IPCIDRs,
		DaysOfWeek:  req.DaysOfWeek,
		StartTime:   req.StartTime,
		EndTime:     req.EndTime,
	}
}
