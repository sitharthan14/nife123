package api

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/google/uuid"
	"github.com/nifetency/nife.io/api/model"
	"github.com/nifetency/nife.io/internal/auth"
	"github.com/nifetency/nife.io/internal/cli_session"
	database "github.com/nifetency/nife.io/internal/pkg/db/mysql"
	stripe "github.com/stripe/stripe-go/v71"
	"github.com/stripe/stripe-go/v71/paymentintent"
	"github.com/stripe/stripe-go/v71/paymentmethod"
	"github.com/stripe/stripe-go/v71/price"
	"github.com/stripe/stripe-go/v71/product"
	"github.com/stripe/stripe-go/v71/sub"
)

func (r *mutationResolver) CreatePaymentIntent(ctx context.Context, input model.CreatePaymentIntent) (*model.Payment, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return &model.Payment{}, fmt.Errorf("Access Denied")
	}
	stripe.Key = os.Getenv("STRIPE_KEY")
	params := &stripe.PaymentIntentParams{

		Amount:   stripe.Int64(int64(input.Amount)),
		Currency: stripe.String(string(stripe.CurrencyUSD)),
		Customer: stripe.String(input.CustomerID),
		PaymentMethodTypes: []*string{
			stripe.String("card"),
		},
	}
	pay, err := paymentintent.New(params)
	if err != nil {
		return nil, err
	}
	res := &model.Payment{
		ID:             pay.ID,
		Amount:         int(pay.Amount),
		Currency:       pay.Currency,
		ClientSecretID: pay.ClientSecret,
	}
	return res, nil
}

func (r *mutationResolver) CreateAttachPaymentMethod(ctx context.Context, input model.CreateAttachPaymentMethod) (*model.AttachPayment, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return &model.AttachPayment{}, fmt.Errorf("Access Denied")
	}
	stripe.Key = os.Getenv("STRIPE_KEY")

	params := &stripe.PaymentMethodAttachParams{
		Customer: stripe.String(input.CustomerID),
	}
	payMethod, err := paymentmethod.Attach(
		input.PaymentMethodID,
		params,
	)
	if err != nil {
		return nil, err
	}

	res := &model.AttachPayment{
		ID: payMethod.ID,
	}
	return res, nil
}

func (r *mutationResolver) CreateStripeSubscription(ctx context.Context, input model.CreateStripeSubscription) (*model.StripeSubscription, error) {
	user := auth.ForContext(ctx)
	if user == nil {
		return &model.StripeSubscription{}, fmt.Errorf("Access Denied")
	}

	stripe.Key = os.Getenv("STRIPE_KEY")

	params := &stripe.SubscriptionParams{
		Customer: stripe.String(input.CustomerID),
		Items: []*stripe.SubscriptionItemsParams{
			{
				Price: stripe.String(input.PriceID),
			},
		},

		DefaultPaymentMethod: stripe.String(input.DefaultPaymentMethodid),
		OffSession:           stripe.Bool(true),
	}
	params.AddExpand("latest_invoice.payment_intent")

	subscription, err := sub.New(params)
	if err != nil {
		return nil, err
	}

	res := &model.StripeSubscription{
		CustomerSubscriptionID: subscription.ID,
	}

	// user plan

	selDB, err := database.Db.Prepare("select name, price, status from plan where stripes_plan_id=?")
	if err != nil {
		return nil, err
	}
	row := selDB.QueryRow(input.PriceID)

	defer selDB.Close()
	var planDetails model.StripeSubscription
	err = row.Scan(&planDetails.Name, &planDetails.Price, &planDetails.Status)
	if err != nil {
		return nil, err
	}

	statement, err := database.Db.Prepare("INSERT INTO user_plans(id,user_id,plan_price,status, plan_name, subscription_id) VALUES(?,?,?,?,?,?)")
	if err != nil {
		return nil, err
	}

	id := uuid.NewString()

	_, err = statement.Exec(id, user.ID, planDetails.Price, planDetails.Status, planDetails.Name, res.CustomerSubscriptionID)
	if err != nil {
		return nil, err
	}
	userid, _ := strconv.Atoi(user.ID)

	if input.AccessToken != nil && input.SessionID != nil {
		_, err = session.UpdateCLISession(*input.AccessToken, userid, *input.SessionID)
		if err != nil {
			return nil, err
		}
	}
	return res, nil
}

func (r *queryResolver) Getpricelist(ctx context.Context) ([]*model.PriceList, error) {
	stripe.Key = os.Getenv("STRIPE_KEY")

	var getPlan []*model.PriceList

	params := &stripe.PriceListParams{}

	i := price.List(params)
	for i.Next() {
		p := i.Price()

		descript, err := product.Get(p.Product.ID, nil)
		if err != nil {
			return nil, err
		}
		p.Product.Description = descript.Description
		item := &model.PriceList{
			Priceid:     p.ID,
			Productid:   p.Product.ID,
			Nickname:    p.Nickname,
			Description: p.Product.Description,
			Unitamount:  int(p.UnitAmount),
		}
		getPlan = append(getPlan, item)

	}
	return getPlan, nil
}
