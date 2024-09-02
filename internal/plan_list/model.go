package planlist

type Planlist struct {
	Id     string `json:"id"`
	Name   string `json:"name"`
	Price  int    `json:"price"`
	Status string `json:"status"`
	StripesPlanid string `json:"stripes_plan_id"`
}




