package users

type User struct {
	ID       string
	Verified bool
	Quota    int
}

func CanAccessBilling(user User) bool {
	return user.Verified && user.Quota > 0
}

func RemainingQuota(user User, used int) int {
	if used >= user.Quota {
		return 0
	}
	return user.Quota - used
}
