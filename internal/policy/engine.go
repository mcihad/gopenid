package policy

import (
	"context"
	"net"
	"time"

	"gopenid/internal/domain"
	"gopenid/internal/store"
)

// Engine gathers the policies relevant to a login attempt from the store and
// evaluates them with the most-specific-wins hierarchy.
type Engine struct {
	db *store.Store
}

func New(db *store.Store) *Engine {
	return &Engine{db: db}
}

// EvaluateLogin evaluates user, group and client policies for an authenticated
// user attempting to access a client.
func (e *Engine) EvaluateLogin(ctx context.Context, user domain.User, clientDBID int64, ip net.IP, now time.Time) (Decision, error) {
	userPolicies, err := e.db.PoliciesForSubjects(ctx, domain.PolicySubjectUser, []int64{user.ID})
	if err != nil {
		return Decision{}, err
	}
	groupIDs := make([]int64, 0, len(user.Groups))
	for _, g := range user.Groups {
		groupIDs = append(groupIDs, g.ID)
	}
	groupPolicies, err := e.db.PoliciesForSubjects(ctx, domain.PolicySubjectGroup, groupIDs)
	if err != nil {
		return Decision{}, err
	}
	var clientPolicies []domain.Policy
	if clientDBID > 0 {
		clientPolicies, err = e.db.PoliciesForSubjects(ctx, domain.PolicySubjectClient, []int64{clientDBID})
		if err != nil {
			return Decision{}, err
		}
	}
	levels := [][]domain.Policy{userPolicies, groupPolicies, clientPolicies}
	return Evaluate(levels, ip, now), nil
}

// EvaluateClient evaluates only the client (application) level policies. It is
// used before authentication to surface time/IP restrictions on the login page.
func (e *Engine) EvaluateClient(ctx context.Context, clientDBID int64, ip net.IP, now time.Time) (Decision, error) {
	if clientDBID <= 0 {
		return Decision{Allowed: true}, nil
	}
	clientPolicies, err := e.db.PoliciesForSubjects(ctx, domain.PolicySubjectClient, []int64{clientDBID})
	if err != nil {
		return Decision{}, err
	}
	return Evaluate([][]domain.Policy{clientPolicies}, ip, now), nil
}
