package users

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"
)

type EmploymentHistory struct {
	ID               string    `json:"_id"`
	CreatedAt        time.Time `json:"created_at"`
	Current          bool      `json:"current"`
	Degree           string    `json:"degree"`
	Description      string    `json:"description"`
	Emails           []string  `json:"emails"`
	EndDate          time.Time `json:"end_date"`
	GradeLevel       string    `json:"grade_level"`
	Kind             string    `json:"kind"`
	Major            string    `json:"major"`
	OrganizationID   string    `json:"organization_id"`
	OrganizationName string    `json:"organization_name"`
	RawAddress       string    `json:"raw_address"`
	StartDate        time.Time `json:"start_date"`
	Title            string    `json:"title"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type PrimaryPhone struct {
	Number          string `json:"number"`
	Source          string `json:"source"`
	SanitizedNumber string `json:"sanitized_number"`
}

type Organization struct {
	ID                     string            `json:"id"`
	Name                   string            `json:"name"`
	WebsiteURL             string            `json:"website_url"`
	BlogURL                string            `json:"blog_url"`
	AngellistURL           string            `json:"angellist_url"`
	LinkedinURL            string            `json:"linkedin_url"`
	TwitterURL             string            `json:"twitter_url"`
	FacebookURL            string            `json:"facebook_url"`
	PrimaryPhone           PrimaryPhone      `json:"primary_phone"`
	Languages              []string          `json:"languages"`
	AlexaRanking           interface{}       `json:"alexa_ranking"`
	Phone                  string            `json:"phone"`
	LinkedinUID            string            `json:"linkedin_uid"`
	FoundedYear            int               `json:"founded_year"`
	PubliclyTradedSymbol   interface{}       `json:"publicly_traded_symbol"`
	PubliclyTradedExchange interface{}       `json:"publicly_traded_exchange"`
	LogoURL                string            `json:"logo_url"`
	CrunchbaseURL          interface{}       `json:"crunchbase_url"`
	PrimaryDomain          string            `json:"primary_domain"`
	SanitizedPhone         string            `json:"sanitized_phone"`
	Industry               string            `json:"industry"`
	Keywords               []string          `json:"keywords"`
	EstimatedNumEmployees  int               `json:"estimated_num_employees"`
	Industries             []string          `json:"industries"`
	SecondaryIndustries    []string          `json:"secondary_industries"`
	SnippetsLoaded         bool              `json:"snippets_loaded"`
	IndustryTagID          string            `json:"industry_tag_id"`
	IndustryTagHash        map[string]string `json:"industry_tag_hash"`
	RetailLocationCount    int               `json:"retail_location_count"`
	RawAddress             string            `json:"raw_address"`
	StreetAddress          string            `json:"street_address"`
	City                   string            `json:"city"`
	State                  string            `json:"state"`
	PostalCode             string            `json:"postal_code"`
	Country                string            `json:"country"`
	OwnedByOrganizationID  interface{}       `json:"owned_by_organization_id"`
	Suborganizations       []interface{}     `json:"suborganizations"`
	NumSuborganizations    int               `json:"num_suborganizations"`
	SEODescription         string            `json:"seo_description"`
	ShortDescription       string            `json:"short_description"`
	TotalFunding           interface{}       `json:"total_funding"`
	TotalFundingPrinted    interface{}       `json:"total_funding_printed"`
	LatestFundingRoundDate interface{}       `json:"latest_funding_round_date"`
	LatestFundingStage     interface{}       `json:"latest_funding_stage"`
	FundingEvents          []interface{}     `json:"funding_events"`
	TechnologyNames        []string          `json:"technology_names"`
	CurrentTechnologies    []struct {
		UID      string `json:"uid"`
		Name     string `json:"name"`
		Category string `json:"category"`
	} `json:"current_technologies"`
	OrgChartRootPeopleIDs []interface{} `json:"org_chart_root_people_ids"`
	OrgChartSector        string        `json:"org_chart_sector"`
}

type Person struct {
	ID                          string              `json:"id"`
	FirstName                   interface{}         `json:"first_name"`
	LastName                    interface{}         `json:"last_name"`
	Name                        string              `json:"name"`
	LinkedinURL                 interface{}         `json:"linkedin_url"`
	Title                       interface{}         `json:"title"`
	EmailStatus                 interface{}         `json:"email_status"`
	PhotoURL                    interface{}         `json:"photo_url"`
	TwitterURL                  interface{}         `json:"twitter_url"`
	GithubURL                   interface{}         `json:"github_url"`
	FacebookURL                 interface{}         `json:"facebook_url"`
	ExtrapolatedEmailConfidence interface{}         `json:"extrapolated_email_confidence"`
	Headline                    interface{}         `json:"headline"`
	Email                       string              `json:"email"`
	OrganizationID              string              `json:"organization_id"`
	EmploymentHistory           []EmploymentHistory `json:"employment_history"`
	Organization                Organization        `json:"organization"`
	IsLikelyToEngage            bool                `json:"is_likely_to_engage"`
	PhoneNumbers                []struct {
		RawNumber       string      `json:"raw_number"`
		SanitizedNumber string      `json:"sanitized_number"`
		Type            string      `json:"type"`
		Position        int         `json:"position"`
		Status          string      `json:"status"`
		DNCStatus       interface{} `json:"dnc_status"`
		DNCOtherInfo    interface{} `json:"dnc_other_info"`
		DialerFlags     interface{} `json:"dialer_flags"`
	} `json:"phone_numbers"`
	IntentStrength         interface{}   `json:"intent_strength"`
	ShowIntent             bool          `json:"show_intent"`
	RevealedForCurrentTeam bool          `json:"revealed_for_current_team"`
	Departments            []interface{} `json:"departments"`
	Subdepartments         []interface{} `json:"subdepartments"`
	Functions              []interface{} `json:"functions"`
	Seniority              interface{}   `json:"seniority"`
}

type ApolloAPIResponse struct {
	Person Person `json:"person"`
}

func GetUserDetailsInApolloAPI(email string) (*ApolloAPIResponse, error) {

	url := "https://api.apollo.io/v1/people/match"
	method := "POST"
	apolloKey := os.Getenv("APOLLO_API_KEY")
	payload := strings.NewReader(`{` + "" + `"api_key": "` + apolloKey + `",` + "" + `"email": "` + email + `"` + "" + `}`)

	client := &http.Client{}
	req, err := http.NewRequest(method, url, payload)

	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Cache-Control", "no-cache")
	req.Header.Add("Cookie", "X-CSRF-TOKEN=61-fdg6a6SaSihRe8YHTpuZTI05F6M_JlJ3l4Kr78Bgaqg8qoUlCb1vrQLCOYdQrHpZNhfNVdSwQ3vY0ylHMqQ; _leadgenie_session=4u4tPb%2FJ4TlOXu09gyU511I7US2aeDQMph9tWkBzlMqdsXTT7436%2FeSkwN7c%2BceC35QrXTo49dHyOumkBL9mxiFwzJJJoWkcEQ37Tmc2X6WpR8cleqG4gDi11BR4PQ4VIdOGvcAqodKiKuaTCD%2FJQuaKrQMOriJU9BeOEwY4a5ulSeRYozPXxyeLmYhkZFpKZadyBOC0lAHQeIfpusej%2FmBqBIESj14a8KGoeKf2SVGtLQ0qPNU4trfznrIqDH%2BCr3TYAhA8c4TOKIMaurzU5EId1dsgNvwIy7E%3D--esLtDiuJ3gxovt7H--SNzndphkoDe4F1l7LWZi%2Fw%3D%3D")

	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	var apiResponse ApolloAPIResponse
	err = json.Unmarshal(body, &apiResponse)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	return &apiResponse, nil
}
