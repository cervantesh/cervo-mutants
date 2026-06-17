package main

import (
	"fmt"

	"github.com/cervantesh/cervomutants-example-medium-service/internal/plans"
	"github.com/cervantesh/cervomutants-example-medium-service/internal/users"
)

func main() {
	user := users.User{ID: "demo", Verified: true, Quota: 2}
	fmt.Printf("%t %s\n", users.CanAccessBilling(user), plans.EffectiveTier("pro", false))
}
