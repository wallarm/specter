package http

import (
	"github.com/spf13/afero"

	"github.com/wallarm/specter/components/providers/http/config"
	"github.com/wallarm/specter/components/providers/http/middleware"
	"github.com/wallarm/specter/components/providers/http/middleware/headerdate"
	"github.com/wallarm/specter/core"
	"github.com/wallarm/specter/core/register"

	httpRegister "github.com/wallarm/specter/components/providers/http/register"
)

func Import(fs afero.Fs) {
	register.Provider("http", func(cfg config.Config) (core.Provider, error) {
		return NewProvider(fs, cfg)
	})

	register.Provider("http/json", func(cfg config.Config) (core.Provider, error) {
		cfg.Decoder = config.DecoderJSONLine
		return NewProvider(fs, cfg)
	})

	register.Provider("uri", func(cfg config.Config) (core.Provider, error) {
		cfg.Decoder = config.DecoderURI
		return NewProvider(fs, cfg)
	})

	register.Provider("uripost", func(cfg config.Config) (core.Provider, error) {
		cfg.Decoder = config.DecoderURIPost
		return NewProvider(fs, cfg)
	})

	register.Provider("raw", func(cfg config.Config) (core.Provider, error) {
		cfg.Decoder = config.DecoderRaw
		return NewProvider(fs, cfg)
	})

	httpRegister.HTTPMW("header/date", func(cfg headerdate.Config) (middleware.Middleware, error) {
		return headerdate.NewMiddleware(cfg)
	})

}
