package core

import (
	"testing"

	"github.com/wallarm/specter/components/guns/grpc"
	"github.com/wallarm/specter/core/warmup"
)

func TestGrpcGunImplementsWarmedUp(t *testing.T) {
	_ = warmup.WarmedUp(&grpc.Gun{})
}
