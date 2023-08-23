package example

import (
	"github.com/spf13/afero"
	"github.com/wallarm/specter/components/guns/grpc"
	"github.com/wallarm/specter/components/providers/grpc/grpcjson"
	"github.com/wallarm/specter/core"
	"github.com/wallarm/specter/core/register"
)

func Import(fs afero.Fs) {

	register.Provider("grpc/json", func(conf grpcjson.Config) core.Provider {
		return grpcjson.NewProvider(fs, conf)
	})

	register.Gun("grpc", grpc.NewGun, grpc.DefaultGunConfig)
}
