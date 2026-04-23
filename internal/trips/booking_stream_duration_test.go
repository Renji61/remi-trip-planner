package trips

import "testing"

func TestFormatVehicleRentalDurationLabel(t *testing.T) {
	cases := []struct {
		name   string
		v      VehicleRental
		expect string
	}{
		{
			name: "four hours same calendar day",
			v: VehicleRental{
				PickUpAt:  "2026-03-26T10:00",
				DropOffAt: "2026-03-26T14:00",
			},
			expect: "4 hrs",
		},
		{
			name: "one hour",
			v: VehicleRental{
				PickUpAt:  "2026-03-26T10:00",
				DropOffAt: "2026-03-26T11:00",
			},
			expect: "1 hr",
		},
		{
			name: "minutes only",
			v: VehicleRental{
				PickUpAt:  "2026-03-26T10:00",
				DropOffAt: "2026-03-26T10:45",
			},
			expect: "45 min",
		},
		{
			name: "hours and minutes under 24h",
			v: VehicleRental{
				PickUpAt:  "2026-03-26T10:00",
				DropOffAt: "2026-03-26T13:15",
			},
			expect: "3 hrs 15 min",
		},
		{
			name: "exactly 24 hours",
			v: VehicleRental{
				PickUpAt:  "2026-03-26T10:00",
				DropOffAt: "2026-03-27T10:00",
			},
			expect: "1 day",
		},
		{
			name: "25 hours",
			v: VehicleRental{
				PickUpAt:  "2026-03-26T10:00",
				DropOffAt: "2026-03-27T11:00",
			},
			expect: "1 day & 1 hr",
		},
		{
			name: "two days four hours",
			v: VehicleRental{
				PickUpAt:  "2026-03-26T10:00",
				DropOffAt: "2026-03-28T14:00",
			},
			expect: "2 days & 4 hrs",
		},
		{
			name: "one day and minutes only remainder",
			v: VehicleRental{
				PickUpAt:  "2026-03-26T10:00",
				DropOffAt: "2026-03-27T10:30",
			},
			expect: "1 day & 30 min",
		},
		{
			name: "invalid drop before pick",
			v: VehicleRental{
				PickUpAt:  "2026-03-27T10:00",
				DropOffAt: "2026-03-26T10:00",
			},
			expect: "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := FormatVehicleRentalDurationLabel(tc.v)
			if got != tc.expect {
				t.Fatalf("got %q want %q", got, tc.expect)
			}
		})
	}
}
