package utils

import (
	"context"
	"net"

	"tailscale.com/client/tailscale"
	"tailscale.com/client/tailscale/apitype"
)

type IServer interface {
	Start() error
	Listen(network, addr string) (net.Listener, error)
	LocalClient() (*tailscale.LocalClient, error)
}

type IClient interface {
	WhoIs(ctx context.Context, addr string) (*apitype.WhoIsResponse, error)
}
