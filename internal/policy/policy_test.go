package policy

import (
	"net"
	"testing"
	"time"

	"gopenid/internal/domain"
)

func ipPolicy(effect domain.PolicyEffect, cidrs string) domain.Policy {
	return domain.Policy{Type: domain.PolicyTypeIP, Effect: effect, IPCIDRs: cidrs}
}

func timePolicy(effect domain.PolicyEffect, days []int, start, end string) domain.Policy {
	return domain.Policy{Type: domain.PolicyTypeTime, Effect: effect, DaysOfWeek: days, StartTime: start, EndTime: end}
}

func TestEvaluateIPPolicies(t *testing.T) {
	ip := net.ParseIP("10.0.0.5")
	allDays := []int{0, 1, 2, 3, 4, 5, 6}
	now := time.Date(2026, 6, 9, 12, 0, 0, 0, time.UTC)
	_ = allDays

	cases := []struct {
		name   string
		levels [][]domain.Policy
		wantOK bool
	}{
		{"deny matching ip", [][]domain.Policy{nil, nil, {ipPolicy(domain.PolicyEffectDeny, "10.0.0.0/24")}}, false},
		{"deny non-matching ip", [][]domain.Policy{nil, nil, {ipPolicy(domain.PolicyEffectDeny, "192.168.0.0/24")}}, true},
		{"allow matching ip", [][]domain.Policy{nil, nil, {ipPolicy(domain.PolicyEffectAllow, "10.0.0.0/24")}}, true},
		{"allow non-matching ip => denied", [][]domain.Policy{nil, nil, {ipPolicy(domain.PolicyEffectAllow, "192.168.0.0/24")}}, false},
		{"no policies => allow", [][]domain.Policy{nil, nil, nil}, true},
		{"single ip exact match deny", [][]domain.Policy{nil, nil, {ipPolicy(domain.PolicyEffectDeny, "10.0.0.5")}}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Evaluate(tc.levels, ip, now)
			if got.Allowed != tc.wantOK {
				t.Fatalf("Evaluate allowed=%v want=%v reason=%q", got.Allowed, tc.wantOK, got.Reason)
			}
		})
	}
}

func TestEvaluateTimePolicies(t *testing.T) {
	ip := net.ParseIP("1.2.3.4")
	// Tuesday 2026-06-09 14:30 UTC. Weekday() => Tuesday = 2.
	now := time.Date(2026, 6, 9, 14, 30, 0, 0, time.UTC)

	cases := []struct {
		name   string
		policy domain.Policy
		wantOK bool
	}{
		{"allow inside window", timePolicy(domain.PolicyEffectAllow, []int{2}, "08:00", "18:00"), true},
		{"allow outside window => denied", timePolicy(domain.PolicyEffectAllow, []int{2}, "15:00", "18:00"), false},
		{"deny inside window", timePolicy(domain.PolicyEffectDeny, []int{2}, "08:00", "18:00"), false},
		{"deny wrong day => allowed", timePolicy(domain.PolicyEffectDeny, []int{0}, "08:00", "18:00"), true},
		{"overnight allow inside", timePolicy(domain.PolicyEffectAllow, nil, "22:00", "06:00"), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Evaluate([][]domain.Policy{nil, nil, {tc.policy}}, ip, now)
			if got.Allowed != tc.wantOK {
				t.Fatalf("Evaluate allowed=%v want=%v reason=%q", got.Allowed, tc.wantOK, got.Reason)
			}
		})
	}
}

func TestEvaluateHierarchyUserOverridesGroupAndClient(t *testing.T) {
	ip := net.ParseIP("10.0.0.5")
	now := time.Date(2026, 6, 9, 12, 0, 0, 0, time.UTC)

	clientDeny := []domain.Policy{timePolicy(domain.PolicyEffectDeny, nil, "00:00", "23:59")}
	groupDeny := []domain.Policy{timePolicy(domain.PolicyEffectDeny, nil, "00:00", "23:59")}
	userAllow := []domain.Policy{timePolicy(domain.PolicyEffectAllow, nil, "00:00", "23:59")}

	// User allow overrides group+client deny.
	if d := Evaluate([][]domain.Policy{userAllow, groupDeny, clientDeny}, ip, now); !d.Allowed {
		t.Fatalf("user allow should override deny: %q", d.Reason)
	}
	// Group decides when user has no policy (group deny overrides client allow).
	clientAllow := []domain.Policy{timePolicy(domain.PolicyEffectAllow, nil, "00:00", "23:59")}
	if d := Evaluate([][]domain.Policy{nil, groupDeny, clientAllow}, ip, now); d.Allowed {
		t.Fatalf("group deny should win over client allow")
	}
	// Client decides when user and group are empty.
	if d := Evaluate([][]domain.Policy{nil, nil, clientDeny}, ip, now); d.Allowed {
		t.Fatalf("client deny should apply")
	}
}
