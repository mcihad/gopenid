package domain

import "time"

type Base struct {
	ID        int64      `json:"ID"`
	CreatedAt time.Time  `json:"CreatedAt"`
	UpdatedAt time.Time  `json:"UpdatedAt"`
	DeletedAt *time.Time `json:"DeletedAt"`
}

type Department struct {
	Base
	Name        string       `json:"name"`
	Description string       `json:"description"`
	ParentID    *int64       `json:"parentId"`
	Children    []Department `json:"children,omitempty"`
}

type Role struct {
	Base
	Name        string `json:"name"`
	Description string `json:"description"`
}

type Group struct {
	Base
	Name        string `json:"name"`
	Description string `json:"description"`
}

type User struct {
	Base
	Email             string       `json:"email"`
	Name              string       `json:"name"`
	PasswordHash      string       `json:"-"`
	Active            bool         `json:"active"`
	Blocked           bool         `json:"blocked"`
	BlockedReason     string       `json:"blockedReason"`
	Phone             string       `json:"phone"`
	Title             string       `json:"title"`
	AvatarURL         string       `json:"avatarUrl"`
	LastLoginAt       *time.Time   `json:"lastLoginAt"`
	FailedLoginCount  int          `json:"failedLoginCount"`
	LockedUntil       *time.Time   `json:"lockedUntil"`
	TOTPSecret        string       `json:"-"`
	MFAEnabled        bool         `json:"mfaEnabled"`
	EmailVerified     bool         `json:"emailVerified"`
	DepartmentID      *int64       `json:"departmentId"`
	Department        Department   `json:"department"`
	Departments       []Department `json:"departments"`
	Groups            []Group      `json:"groups"`
	Roles             []Role       `json:"roles"`
	AuthorizedClients []Client     `json:"authorizedClients"`
	ClientRoles       []ClientRole `json:"clientRoles"`
}

type AuthCode struct {
	Base
	Code                string
	UserID              int64
	ClientID            string
	RedirectURI         string
	Scope               string
	Nonce               string
	CodeChallenge       string
	CodeChallengeMethod string
	Used                bool
}

type Client struct {
	Base
	ClientID           string       `json:"clientId"`
	ClientSecret       string       `json:"-"`
	ClientSecretPlain  string       `json:"clientSecret,omitempty"`
	HasClientSecret    bool         `json:"hasClientSecret"`
	Name               string       `json:"name"`
	Description        string       `json:"description"`
	HomeURL            string       `json:"homeUrl"`
	LogoURL            string       `json:"logoUrl"`
	RedirectURIs       string       `json:"redirectUris"`
	TokenTTLSeconds    int          `json:"tokenTtlSeconds"`
	RefreshTTLSeconds  int          `json:"refreshTtlSeconds"`
	AllowPasswordGrant bool         `json:"allowPasswordGrant"`
	Roles              []ClientRole `json:"roles"`
}

type ClientRole struct {
	Base
	ClientID    int64  `json:"clientId"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type SigningKey struct {
	Base
	KeyID      string
	PrivatePEM string
	Active     bool
}

// PolicyType is the kind of condition a login policy evaluates.
type PolicyType string

const (
	PolicyTypeIP   PolicyType = "ip"
	PolicyTypeTime PolicyType = "time"
)

// PolicyEffect decides whether a matching policy permits or rejects access.
type PolicyEffect string

const (
	PolicyEffectAllow PolicyEffect = "allow"
	PolicyEffectDeny  PolicyEffect = "deny"
)

// PolicySubject is the entity a policy is attached to. The evaluation
// hierarchy is user > group > client (most specific level wins).
type PolicySubject string

const (
	PolicySubjectClient PolicySubject = "client"
	PolicySubjectGroup  PolicySubject = "group"
	PolicySubjectUser   PolicySubject = "user"
)

type Policy struct {
	Base
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Type        PolicyType   `json:"type"`
	Effect      PolicyEffect `json:"effect"`
	// IPCIDRs holds comma separated CIDR ranges or single IPs (ip policy).
	IPCIDRs string `json:"ipCidrs"`
	// DaysOfWeek lists allowed weekdays (0=Sunday..6=Saturday). Empty means
	// every day. StartTime/EndTime are HH:MM 24h window bounds (time policy).
	DaysOfWeek []int  `json:"daysOfWeek"`
	StartTime  string `json:"startTime"`
	EndTime    string `json:"endTime"`
}

type PolicyAssignment struct {
	Base
	PolicyID    int64         `json:"policyId"`
	SubjectType PolicySubject `json:"subjectType"`
	SubjectID   int64         `json:"subjectId"`
}

// RefreshToken stores a hashed opaque refresh token tied to a session.
type RefreshToken struct {
	Base
	TokenHash string     `json:"-"`
	UserID    int64      `json:"userId"`
	ClientID  string     `json:"clientId"`
	Scope     string     `json:"scope"`
	ExpiresAt time.Time  `json:"expiresAt"`
	Revoked   bool       `json:"revoked"`
	RevokedAt *time.Time `json:"revokedAt"`
}

// RevokedToken blacklists an access token by its JWT ID until it expires.
type RevokedToken struct {
	Base
	JTI       string    `json:"jti"`
	UserID    int64     `json:"userId"`
	Reason    string    `json:"reason"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type AccountToken struct {
	Base
	UserID    int64      `json:"userId"`
	TokenHash string     `json:"-"`
	Type      string     `json:"type"`
	ExpiresAt time.Time  `json:"expiresAt"`
	UsedAt    *time.Time `json:"usedAt"`
}

type BrowserSession struct {
	Base
	TokenHash string    `json:"-"`
	UserID    int64     `json:"userId"`
	AuthTime  time.Time `json:"authTime"`
	ExpiresAt time.Time `json:"expiresAt"`
	Revoked   bool      `json:"revoked"`
}

// AuditLog records authentication lifecycle events.
type AuditLog struct {
	Base
	UserID    *int64 `json:"userId"`
	Email     string `json:"email"`
	ClientID  string `json:"clientId"`
	Event     string `json:"event"`
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	IP        string `json:"ip"`
	UserAgent string `json:"userAgent"`
	Device    string `json:"device"`
	Browser   string `json:"browser"`
	OS        string `json:"os"`
}

// Audit event names.
const (
	EventLogin         = "login"
	EventLoginFailed   = "login_failed"
	EventLogout        = "logout"
	EventTokenRefresh  = "token_refresh"
	EventTokenRevoke   = "token_revoke"
	EventAccessDenied  = "access_denied"
	EventUserBlocked   = "user_blocked"
	EventUserUnblocked = "user_unblocked"
)
