package main

import (
	"github.com/spf13/afero"
	"github.com/wallarm/specter/cli"
	grpc "github.com/wallarm/specter/components/grpc/import"
	phttp "github.com/wallarm/specter/components/phttp/import"
	coreimport "github.com/wallarm/specter/core/import"
)

func main() {
	fs := afero.NewOsFs()
	coreimport.Import(fs)
	phttp.Import(fs)
	grpc.Import(fs)

	cli.Run()
}
