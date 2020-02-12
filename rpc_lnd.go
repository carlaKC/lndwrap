package lndwrap

import (
	"context"
	"log"

	"github.com/lightningnetwork/lnd/lnrpc"
)

type LNDRPC interface {
	//TODO(carla): implement the LND main rpc interface here.
	GetInfo(ctx context.Context) (*Info, error)
}

type Info struct {
	NodePubkey string
}

func (c *Client) GetInfo(ctx context.Context) (*Info, error) {
	if !c.enableRPCServer {
		return nil, errNotEnabled
	}

	resp, err := c.rpcClient.GetInfo(ctx, &lnrpc.GetInfoRequest{})
	if err != nil {
		return nil, err
	}

	return &Info{
		NodePubkey: resp.IdentityPubkey,
	}, nil
}

func (c *Client) SubscribePeer(ctx context.Context) error {
	cl, err := c.rpcClient.SubscribePeerEvents(ctx, &lnrpc.PeerEventSubscription{})
	if err != nil {
		return err
	}

	i := 0
	for {
		log.Printf("Waiting for event: %v", i)
		if i == 3 {
			return nil
		}

		resp, err := cl.Recv()
		if err != nil {
			return err
		}
		log.Printf("Got event: %v type: %v", resp.PubKey, resp.Type)
	}

	return nil
}
