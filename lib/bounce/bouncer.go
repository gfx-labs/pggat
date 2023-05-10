package bounce

import "pggat2/lib/pnet"

type Bouncer interface {
	Bounce(client, server pnet.ReadWriter)
}
