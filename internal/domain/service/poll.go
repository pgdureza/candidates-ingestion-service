package service

import (
	"context"
)

type PollHandler interface {
	Execute(ctx context.Context)
}
