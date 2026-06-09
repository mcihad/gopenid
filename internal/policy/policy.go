// Package policy evaluates login policies. Policies are either IP-based or
// time-based, carry an allow/deny effect and are attached to clients, groups or
// users. Evaluation follows a most-specific-wins hierarchy: a user policy
// overrides a group policy, which overrides a client (application) policy.
package policy

import (
	"net"
	"strconv"
	"strings"
	"time"

	"gopenid/internal/domain"
)

// Decision is the outcome of evaluating a set of policies.
type Decision struct {
	Allowed bool
	Reason  string
	Policy  *domain.Policy
}

// Evaluate applies the level groups in order (most specific first). The first
// level that expresses an opinion (allow or deny) decides the outcome. When no
// level has any applicable policy the default is to allow.
func Evaluate(levels [][]domain.Policy, ip net.IP, now time.Time) Decision {
	for _, level := range levels {
		if decided, decision := evaluateLevel(level, ip, now); decided {
			return decision
		}
	}
	return Decision{Allowed: true}
}

func evaluateLevel(policies []domain.Policy, ip net.IP, now time.Time) (bool, Decision) {
	if len(policies) == 0 {
		return false, Decision{}
	}
	var allowPolicies []domain.Policy
	var allowMatched bool
	for i := range policies {
		p := policies[i]
		match := matches(p, ip, now)
		if p.Effect == domain.PolicyEffectDeny {
			if match {
				return true, Decision{Allowed: false, Reason: denyReason(p), Policy: clone(p)}
			}
			continue
		}
		// allow policy
		allowPolicies = append(allowPolicies, p)
		if match {
			allowMatched = true
		}
	}
	if len(allowPolicies) > 0 {
		if allowMatched {
			return true, Decision{Allowed: true}
		}
		return true, Decision{Allowed: false, Reason: allowMissReason(allowPolicies[0]), Policy: clone(allowPolicies[0])}
	}
	// Only deny policies existed and none matched -> allowed at this level.
	return true, Decision{Allowed: true}
}

// matches reports whether the policy condition holds for the given context.
func matches(p domain.Policy, ip net.IP, now time.Time) bool {
	switch p.Type {
	case domain.PolicyTypeIP:
		return ipMatches(p.IPCIDRs, ip)
	case domain.PolicyTypeTime:
		return timeMatches(p, now)
	default:
		return false
	}
}

func ipMatches(cidrs string, ip net.IP) bool {
	if ip == nil {
		return false
	}
	for _, raw := range strings.Split(cidrs, ",") {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		if strings.Contains(raw, "/") {
			if _, network, err := net.ParseCIDR(raw); err == nil && network.Contains(ip) {
				return true
			}
			continue
		}
		if parsed := net.ParseIP(raw); parsed != nil && parsed.Equal(ip) {
			return true
		}
	}
	return false
}

func timeMatches(p domain.Policy, now time.Time) bool {
	if len(p.DaysOfWeek) > 0 {
		day := int(now.Weekday())
		found := false
		for _, d := range p.DaysOfWeek {
			if d == day {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	start, okStart := parseMinutes(p.StartTime)
	end, okEnd := parseMinutes(p.EndTime)
	if !okStart || !okEnd {
		// Only a day constraint was provided.
		return true
	}
	cur := now.Hour()*60 + now.Minute()
	if start <= end {
		return cur >= start && cur <= end
	}
	// Overnight window (e.g. 22:00 - 06:00).
	return cur >= start || cur <= end
}

func parseMinutes(value string) (int, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}
	parts := strings.SplitN(value, ":", 2)
	if len(parts) != 2 {
		return 0, false
	}
	h, err1 := strconv.Atoi(parts[0])
	m, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil || h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, false
	}
	return h*60 + m, true
}

func denyReason(p domain.Policy) string {
	if p.Type == domain.PolicyTypeIP {
		return "Bu IP adresinden bu uygulamaya giriş izniniz bulunmuyor."
	}
	return "Bu uygulamaya şu an (zaman kısıtlaması nedeniyle) giriş yapamazsınız."
}

func allowMissReason(p domain.Policy) string {
	if p.Type == domain.PolicyTypeIP {
		return "Bu uygulamaya yalnızca izin verilen IP adreslerinden giriş yapılabilir."
	}
	return "Bu uygulamaya yalnızca izin verilen saatlerde giriş yapabilirsiniz."
}

func clone(p domain.Policy) *domain.Policy {
	c := p
	return &c
}
