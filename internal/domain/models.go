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
	Name        string `json:"name"`
	Description string `json:"description"`
}

type Role struct {
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
	DepartmentID      *int64       `json:"departmentId"`
	Department        Department   `json:"department"`
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
	ClientID     string       `json:"clientId"`
	ClientSecret string       `json:"clientSecret"`
	Name         string       `json:"name"`
	RedirectURIs string       `json:"redirectUris"`
	Roles        []ClientRole `json:"roles"`
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
