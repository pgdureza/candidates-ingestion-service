package service

import (
	"context"
)

type Publisher interface {
	PublishJSON(ctx context.Context, topic string, msg []byte) error
}
