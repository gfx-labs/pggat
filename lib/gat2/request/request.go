package request

import "gfx.cafe/gfx/pggat/lib/gat2/source"

type Request interface {
	Source() source.Source
}
