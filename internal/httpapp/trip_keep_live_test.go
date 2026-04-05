package httpapp

import "testing"

func TestKeepChangeAffectsKeepUI(t *testing.T) {
	if !keepChangeAffectsKeepUI("trip_note") {
		t.Fatal("trip_note should affect keep UI")
	}
	if !keepChangeAffectsKeepUI("checklist_item") {
		t.Fatal("checklist_item should affect keep UI")
	}
	if !keepChangeAffectsKeepUI("checklist_category_pin") {
		t.Fatal("checklist_category_pin should affect keep UI")
	}
	if keepChangeAffectsKeepUI("itinerary_item") {
		t.Fatal("itinerary_item should not use keep board polling entity set")
	}
	if keepChangeAffectsKeepUI("") {
		t.Fatal("empty entity should not match")
	}
}
