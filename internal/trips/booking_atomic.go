package trips

import "context"

func (s *Service) AddLodgingWithItinerary(ctx context.Context, lodging Lodging, checkInItem, checkOutItem ItineraryItem) error {
	return s.withRepoTransaction(ctx, func(txs *Service) error {
		if err := txs.AddItineraryItem(ctx, checkInItem); err != nil {
			return err
		}
		if err := txs.AddItineraryItem(ctx, checkOutItem); err != nil {
			return err
		}
		lodging.CheckInItineraryID = checkInItem.ID
		lodging.CheckOutItineraryID = checkOutItem.ID
		return txs.AddLodging(ctx, lodging)
	})
}

func (s *Service) UpdateLodgingWithItinerary(
	ctx context.Context,
	lodging Lodging,
	previousName string,
	checkInDay int,
	checkInTime string,
	checkOutDay int,
	checkOutTime string,
	checkInNotes string,
	checkOutNotes string,
) error {
	return s.withRepoTransaction(ctx, func(txs *Service) error {
		trip, err := txs.repo.GetTrip(ctx, lodging.TripID)
		if err != nil {
			return err
		}
		lodging, err = txs.SyncLodgingItinerary(ctx, trip, lodging, previousName, checkInDay, checkInTime, checkOutDay, checkOutTime, checkInNotes, checkOutNotes)
		if err != nil {
			return err
		}
		return txs.UpdateLodging(ctx, lodging)
	})
}

func (s *Service) AddVehicleRentalWithItinerary(ctx context.Context, rental VehicleRental, pickUpItem, dropOffItem ItineraryItem) error {
	return s.withRepoTransaction(ctx, func(txs *Service) error {
		if err := txs.AddItineraryItem(ctx, pickUpItem); err != nil {
			return err
		}
		if err := txs.AddItineraryItem(ctx, dropOffItem); err != nil {
			return err
		}
		return txs.AddVehicleRental(ctx, rental)
	})
}

func (s *Service) UpdateVehicleRentalWithItinerary(ctx context.Context, rental VehicleRental, pickUpItem, dropOffItem ItineraryItem) error {
	return s.withRepoTransaction(ctx, func(txs *Service) error {
		if err := txs.UpdateItineraryItem(ctx, pickUpItem); err != nil {
			return err
		}
		if err := txs.UpdateItineraryItem(ctx, dropOffItem); err != nil {
			return err
		}
		return txs.UpdateVehicleRental(ctx, rental)
	})
}

func (s *Service) AddFlightWithItinerary(ctx context.Context, flight Flight, departItem, arriveItem ItineraryItem) error {
	return s.withRepoTransaction(ctx, func(txs *Service) error {
		if err := txs.AddItineraryItem(ctx, departItem); err != nil {
			return err
		}
		if err := txs.AddItineraryItem(ctx, arriveItem); err != nil {
			return err
		}
		return txs.AddFlight(ctx, flight)
	})
}

func (s *Service) UpdateFlightWithItinerary(ctx context.Context, flight Flight, departItem, arriveItem ItineraryItem) error {
	return s.withRepoTransaction(ctx, func(txs *Service) error {
		if err := txs.UpdateItineraryItem(ctx, departItem); err != nil {
			return err
		}
		if err := txs.UpdateItineraryItem(ctx, arriveItem); err != nil {
			return err
		}
		return txs.UpdateFlight(ctx, flight)
	})
}
