package service

import (
	"context"
)

type MessageHandler interface {
	Handle(ctx context.Context, data []byte) error
}
