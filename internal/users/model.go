package users

import "time"

// Swagger model for Login
type LoginRequestBody struct {
	Data Data `json:"data"`
}

type Data struct {
	Attributes Attributes `json:"attributes"`
}

type CLISessionRequestBody struct {
	Name   string `json:"name"`
	SignUp bool   `json:"signup"`
}

type UserRegisterRequestBody struct {
	FirstName      string `json:"firstName" validate:"required"`
	LastName       string `json:"lastName" validate:"required"`
	Email          string `json:"email" validate:"required"`
	Password       string `json:"password" validate:"required"`
	PhoneNumber    string `json:"phoneNumber" validate:"required"`
	CompanyName    string `json:"companyName" validate:"required"`
	Industry       string `json:"industry"`
	BillingAddress string `json:"billingAddress"`
	Location       string `json:"location" validate:"required"`
	SessionId      string `json:"sessionId"`
	Line1          string `json:"line1"`
	Line2          string `json:"line2"`
	City           string `json:"city"`
	State          string `json:"state"`
	Country        string `json:"country"`
	Postalcode     string `json:"postalcode"`
	UserName       string `json:"userName"`
	SsoType        string `json:"ssoType"`
	RollId         string `json:"rollId"`
}

type Attributes struct {
	Email    string `json:"email" validate:"required"`
	Password string `json:"password" validate:"required"`
}

// Swagger model for Refresh Token
type RefreshTokenRequestBody struct {
	RefreshToken string `json:"refresh_token"`
}

// Swagger response body for Login and Refresh Token
type TokenResponseBody struct {
	Data TokenAttributes `json:"data"`
}

type TokenAttributes struct {
	TokenAttrib TokenAttrib `json:"attributes"`
}

type TokenAttrib struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type TokenErrorBody struct {
	Message string `json:"message"`
}

type GoogleAccessTokenDetails struct {
	Email interface{} `json:"email"`
}

type SsoLoginDetails struct {
	AccessToken string `json:"accessToken"`
	SSOType     string `json:"ssoType"`
	GithubCode  string `json:"code"`
}

type Roles struct {
	Name      string    `json:"name"`
	IsActive  bool      `json:"isActive"`
	CreatedAt time.Time `json:"createdAt"`
}

type RolePermission struct {
	Module string `json:"module"`
	Title  string `json:"title"`
	Create bool   `json:"create"`
	View   bool   `json:"view"`
	Delete bool   `json:"delete"`
	Update bool   `json:"update"`
}
