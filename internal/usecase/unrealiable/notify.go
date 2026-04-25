package unrealiable

import (
	"context"
	"errors"
	"math/rand"

	"github.com/candidate-ingestion/service/internal/domain/model"
	"github.com/candidate-ingestion/service/internal/domain/service"
)

var _ service.Notifier = new(UnreliableNotifier)

type UnreliableNotifier struct{}

func NewUnreliableNotifier() *UnreliableNotifier {
	return &UnreliableNotifier{}
}

func (p *UnreliableNotifier) Notify(ctx context.Context, candidate *model.Candidate) error {
	rng := rand.Intn(10)
	if rng < 1 {
		return errors.New("notifier failed")
	}
	return nil
}
