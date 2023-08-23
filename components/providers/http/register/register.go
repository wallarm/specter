package register

import (
	"github.com/wallarm/specter/components/providers/http/middleware"
	"github.com/wallarm/specter/core/register"
)

func HTTPMW(name string, mwConstructor interface{}, defaultConfigOptional ...interface{}) {
	var ptr *middleware.Middleware
	register.RegisterPtr(ptr, name, mwConstructor, defaultConfigOptional...)
}
