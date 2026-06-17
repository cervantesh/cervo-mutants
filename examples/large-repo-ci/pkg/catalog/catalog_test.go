package catalog

import "testing"

func TestVisible(t *testing.T) {
	items := []Item{
		{ID: "a", Active: true, Quantity: 2},
		{ID: "b", Active: false, Quantity: 2},
		{ID: "c", Active: true, Quantity: 0},
	}
	got := Visible(items)
	if len(got) != 1 || got[0].ID != "a" {
		t.Fatalf("Visible(...) = %+v, want only item a", got)
	}
}
