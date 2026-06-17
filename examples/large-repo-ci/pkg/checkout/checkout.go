package checkout

import "github.com/cervantesh/cervomutants-example-large-repo-ci/pkg/pricing"

func ReadyToSubmit(quantity int, threshold int, subtotal int, discount int) bool {
	return pricing.Billable(quantity, threshold) && pricing.DiscountedCents(subtotal, discount) > 0
}
