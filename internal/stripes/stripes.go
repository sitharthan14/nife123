package stripes

import (
	"encoding/json"
	"strconv"

	"log"

	"net/http"
	"os"

	"github.com/nifetency/nife.io/helper"
	"github.com/stripe/stripe-go/v71"
	"github.com/stripe/stripe-go/v71/billingportal/session"
	"github.com/stripe/stripe-go/v71/customer"

	"github.com/stripe/stripe-go/v71/product"
	"github.com/stripe/stripe-go/v71/sub"
)

type StripeCustomerPortal struct {
	Customer string `json:"customer"`
}

func HandleCustomerPortal(w http.ResponseWriter, r *http.Request) {
	var dataBody StripeCustomerPortal
	err := json.NewDecoder(r.Body).Decode(&dataBody)
	if err != nil {
		helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
		return
	}

	stripe.Key = os.Getenv("STRIPE_KEY")
	teamBoardURL := os.Getenv("TEAMBOARD_URL")
	// Authenticate your user.
	params := &stripe.BillingPortalSessionParams{
		Customer:  stripe.String(dataBody.Customer),
		ReturnURL: stripe.String(teamBoardURL),
	}
	s, _ := session.New(params)

	helper.RespondwithJSON(w, http.StatusOK, map[string]interface{}{
		"customerPortal": s.URL,
	})

}

func GetStripeCustomerDetails(email string) string {
	stripe.Key = os.Getenv("STRIPE_KEY")

	params := &stripe.CustomerListParams{
		Email: &email,
	}
	params.Filters.AddFilter("limit", "", "3")
	i := customer.List(params)
	var customerStripesId string
	for i.Next() {
		c := i.Customer()
		customerStripesId = c.ID
		break
	}
	return customerStripesId
}

func ListSubscription(custStripesId string) (stripe.SubscriptionStatus, string) {
	stripe.Key = os.Getenv("STRIPE_KEY")

	params := &stripe.SubscriptionListParams{
		Customer: custStripesId,
		Status:   "active",
	}
	params.Filters.AddFilter("limit", "", "3")
	i := sub.List(params)

	var activePlan stripe.SubscriptionStatus
	var productId string
	for i.Next() {
		s := i.Subscription()
		activePlan = s.Status
		productId = s.Plan.Product.ID

	}
	return activePlan, productId
}

func GetCustPlan(productId string) (int, error) {
	stripe.Key = os.Getenv("STRIPE_KEY")

	p, err := product.Get(productId, nil)
	if err != nil {
		log.Println(err)
		return 0, err
	}
	metaData := p.Metadata["Allowed"]
	planMetaData, _ := strconv.Atoi(metaData)
	return planMetaData, nil
}

func GetCustPlanName(custStripesId string) (string, error) {
	stripe.Key = os.Getenv("STRIPE_KEY")

	params := &stripe.SubscriptionListParams{
		Customer: custStripesId,
		Status:   "active",
	}
	params.Filters.AddFilter("limit", "", "3")
	i := sub.List(params)

	var planName string
	for i.Next() {
		s := i.Subscription()
		planName = s.Plan.Nickname
	}

	if planName == "platinum" {
		planName = "Enterprise"
	} else if planName == "gold" {
		planName = "Premium"
	} else if planName == "silver" {
		planName = "Starter"
	} else if planName == "Hobby" {
		planName = "free plan"
	}
	return planName, nil
}
