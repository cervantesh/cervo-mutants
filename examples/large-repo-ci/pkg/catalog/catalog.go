package catalog

type Item struct {
	ID       string
	Active   bool
	Quantity int
}

func Visible(items []Item) []Item {
	visible := make([]Item, 0, len(items))
	for _, item := range items {
		if item.Active && item.Quantity > 0 {
			visible = append(visible, item)
		}
	}
	return visible
}
