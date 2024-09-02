package planlist

import (
	"net/http"
	"os"

	"github.com/nifetency/nife.io/api/model"
	"github.com/nifetency/nife.io/helper"

	stripe "github.com/stripe/stripe-go/v71"
	"github.com/stripe/stripe-go/v71/price"
	"github.com/stripe/stripe-go/v71/product"
)

func GetPlanList(w http.ResponseWriter, r *http.Request) {

	stripe.Key = os.Getenv("STRIPE_KEY")

	var getPlan []*model.PriceList

	params := &stripe.PriceListParams{}

	i := price.List(params)
	for i.Next() {
		p := i.Price()

		descript, err := product.Get(p.Product.ID, nil)
		if err != nil {
			helper.RespondwithJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
			return
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

	helper.RespondwithJSON(w, http.StatusOK, getPlan)
}

