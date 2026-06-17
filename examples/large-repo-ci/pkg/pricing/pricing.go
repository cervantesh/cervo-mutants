package pricing

func DiscountedCents(base int, discount int) int {
	if discount <= 0 {
		return base
	}
	if discount >= 100 {
		return 0
	}
	return base - (base*discount)/100
}

func Billable(quantity int, threshold int) bool {
	return quantity >= threshold
}
