package ssl_endpoint

import (
	"pggat/lib/gat"
	"pggat/lib/util/strutil"
)

type Module struct{}

func NewModule() (*Module, error) {
	return &Module{}, nil
}

func (T *Module) GatModule() {}

func (T *Module) Endpoints() []gat.Endpoint {
	// TODO(garet) gen ssl keys

	return []gat.Endpoint{
		{
			Network: "tcp",
			Address: ":5432",
			AcceptOptions: gat.FrontendAcceptOptions{
				SSLRequired: false,
				AllowedStartupOptions: []strutil.CIString{
					strutil.MakeCIString("client_encoding"),
					strutil.MakeCIString("datestyle"),
					strutil.MakeCIString("timezone"),
					strutil.MakeCIString("standard_conforming_strings"),
					strutil.MakeCIString("application_name"),
					strutil.MakeCIString("extra_float_digits"),
					strutil.MakeCIString("options"),
				},
			},
		},
	}
}

var _ gat.Module = (*Module)(nil)
var _ gat.Listener = (*Module)(nil)
