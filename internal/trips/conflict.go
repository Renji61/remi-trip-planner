package trips

import (
	"errors"
	"strings"
	"time"
)

type ConflictError struct {
	Resource        string
	Message         string
	LatestUpdatedAt time.Time
}

func (e *ConflictError) Error() string {
	if e == nil {
		return "conflict"
	}
	if msg := strings.TrimSpace(e.Message); msg != "" {
		return msg
	}
	if resource := strings.TrimSpace(e.Resource); resource != "" {
		return "conflict updating " + resource
	}
	return "conflict"
}

func IsConflictError(err error) bool {
	var conflict *ConflictError
	return errors.As(err, &conflict)
}
