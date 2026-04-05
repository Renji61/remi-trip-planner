package trips

import (
	"context"
	"errors"
)

// ErrTripAccessDenied is returned when a user may not read or mutate a trip.
var ErrTripAccessDenied = errors.New("trip access denied")

// ErrAuthRequired is returned when an operation requires a logged-in user.
var ErrAuthRequired = errors.New("authentication required")

// ErrAdminRequired is returned when an operation requires an instance administrator.
var ErrAdminRequired = errors.New("administrator access required")

// TripAccess describes how the current user relates to a trip.
type TripAccess struct {
	TripID    string
	IsOwner   bool
	CanManage bool // archive/delete / stop sharing / invites
}

// AccessRepository is implemented by the storage layer for authorization checks.
type AccessRepository interface {
	IsTripOwner(ctx context.Context, tripID, userID string) (bool, error)
	IsActiveCollaborator(ctx context.Context, tripID, userID string) (bool, error)
}

// ResolveTripAccess returns whether the user may access the trip and their role.
func ResolveTripAccess(ctx context.Context, repo AccessRepository, tripID, userID string) (TripAccess, error) {
	if userID == "" {
		return TripAccess{}, ErrAuthRequired
	}
	owner, err := repo.IsTripOwner(ctx, tripID, userID)
	if err != nil {
		return TripAccess{}, err
	}
	if owner {
		return TripAccess{TripID: tripID, IsOwner: true, CanManage: true}, nil
	}
	collab, err := repo.IsActiveCollaborator(ctx, tripID, userID)
	if err != nil {
		return TripAccess{}, err
	}
	if collab {
		return TripAccess{TripID: tripID, IsOwner: false, CanManage: false}, nil
	}
	return TripAccess{}, ErrTripAccessDenied
}
