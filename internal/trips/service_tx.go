package trips

import "context"

type transactionCapableRepository interface {
	RunInTx(ctx context.Context, fn func(Repository) error) error
}

func (s *Service) withRepoTransaction(ctx context.Context, fn func(*Service) error) error {
	txRepo, ok := s.repo.(transactionCapableRepository)
	if !ok {
		return fn(s)
	}
	return txRepo.RunInTx(ctx, func(repo Repository) error {
		return fn(&Service{repo: repo})
	})
}
