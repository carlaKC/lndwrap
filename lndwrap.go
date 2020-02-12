// package lndwrap provides a wrapper to connect to the lnd grpc client.
package lndwrap

import (
	"context"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"strings"
	"time"

	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lightningnetwork/lnd/lnrpc/routerrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

type Client struct {
	macaroon       string
	lndGRPCAddress string
	walletPassword []byte
	cert           credentials.TransportCredentials
	unlockerConn   *grpc.ClientConn
	rpcConn        *grpc.ClientConn

	//TODO(carla): add all the subsystms
	rpcClient    lnrpc.LightningClient
	routerClient routerrpc.RouterClient

	//routingClient routerrpc.RouterClient

	enableRPCServer        bool
	enableChanFitSubServer bool
	enableRouter           bool

	// requireUnlock indicates that we should try to unlock the client.
	requireUnlock bool
}

type Option func(*Client)

func WithAddress(address string) Option {
	return func(c *Client) { c.lndGRPCAddress = address }
}

func WithMacaroon(macaroonPath string) Option {
	return func(c *Client) {
		if macaroonPath == "" {
			log.Fatal("no macaroon path provided")
		}

		dat, err := ioutil.ReadFile(macaroonPath)
		if err != nil {
			log.Fatalf("cannot read macaroon path: %v", err)
		}

		c.macaroon = hex.EncodeToString(dat)
	}
}

func WithCert(certPath string) Option {
	return func(c *Client) {
		cert, err := credentials.NewClientTLSFromFile(certPath, "")
		if err != nil {
			log.Fatalf("cannot read cert path: %v", err)
		}

		c.cert = cert
	}
}

func WithChanFitSubserver() Option {
	return func(c *Client) {
		c.enableChanFitSubServer = true
	}
}

// WithWalletUnlock should be used for nodes which have passwords on their
// wallets (hopefully most!). If you use this option, you must call UnlockWallet
// before attempting to use any of the other grpc calls. If your wallet
func WithWalletUnlock() Option {
	return func(c *Client) {
		c.requireUnlock = true
	}
}

// LightningRPC is an interface which combines LND's main
// rpc server with all of the available subsystems.
type LightningRPC interface {

	// UnlockWallet is a special function because it needs a separate grpc
	// connection. If the client is called with the WithWalletUnlock function,
	// this function must be called first, all other grpc calls will fail.
	UnlockWallet(ctx context.Context, password string) error

	// TODO(carla): remove after testing
	SubscribePeer(ctx context.Context) error

	// TrackPayment
	//TrackPayment(ctx context.Context) error

	SubscribeChannels(ctx context.Context) error

	SubscribeHtlcEvents(ctx context.Context) error

	// LND main RPC server
	LNDRPC

	// ChanFitnessRPC sub-server
	//	ChanFitnessRPC

	//TODO(carla): Add all the GRPC functions
}

// macCtx appends a macaroon to context if the client has one
// set. This function should *always* called to wrap grpc ctx
// for grpc calls.
func (cl *Client) macCtx(ctx context.Context) context.Context {
	if cl.macaroon == "" {
		return ctx
	}

	return metadata.AppendToOutgoingContext(
		ctx, "macaroon", cl.macaroon)
}

func (cl *Client) UnlockWallet(ctx context.Context, password string) error {
	unlockerConn, err := grpc.Dial(
		cl.lndGRPCAddress,
		grpc.WithTransportCredentials(cl.cert),
	)
	if err != nil {
		return err
	}

	// get unlocker client and try to unlock the wallet, if it is already
	// unlocked just ignore the error thrown. Note that this error is string
	// matched, so might need updating in future versions.
	_, err = lnrpc.NewWalletUnlockerClient(unlockerConn).UnlockWallet(
		cl.macCtx(ctx),
		&lnrpc.UnlockWalletRequest{
			WalletPassword: []byte(password),
		},
	)
	if err != nil && strings.Contains(err.Error(), " Wallet is already unlocked") {
		log.Println("Wallet is already unlocked")
	} else if err != nil {
		return err
	}

	// the wallet has been successfully unlocked, so this check is no longer needed.
	cl.requireUnlock = false

	// close the unlocker connection, because we now need to connect to the new
	// server which has been started by unlocking the wallet.
	if err := unlockerConn.Close(); err != nil {
		return err
	}

	// now that the wallet has been unlocked, we can connect to all the
	// subservers that the client requested we connect to.
	return cl.setupSubServers()
}

//TODO(carla): think about whether it is a good idea to combine all
// rpc servers into one interface or whether they should be spit out.
func Connect(options ...Option) (LightningRPC, error) {
	// default config has no macaroon and connects to LND
	// on localhost.
	cl := &Client{
		lndGRPCAddress:  "localhost:10009",
		enableRPCServer: true,
	}

	for _, o := range options {
		o(cl)
	}

	// The wallet unlocker rpc server is a separate grpc server to the rest
	// of the lightning sub-servers, so if the lightning server requires
	// unlocking, it is necessary to connect, unlock and then re-establish
	// a new connection to the lightning server. This may block, as time is
	// needed to tear down the connection and make a new connection.
	if cl.requireUnlock {
		return cl, nil
	}

	// if we do not need to unlock the wallet, we can proceed to setup
	// all the sub-servers that the client requested.
	return cl, cl.setupSubServers()
}

/*
func (cl *Client) TrackPayment(ctx context.Context) error {
	hashStr := "bb0403ede017a2f799351d82010d5bacb8629c41557f89deef1cc677bd4c2b65"

	hash, err := hex.DecodeString(hashStr)
	if err != nil {
		return err
	}

	track, err := cl.routingClient.TrackPayment(ctx, &routerrpc.TrackPaymentRequest{
		PaymentHash: hash,
	})
	if err != nil {
		return err
	}

	status, err := track.Recv()
	if err != nil {
		return err
	}
	log.Println("CKC ", status)

	return nil
}
*/
func (cl *Client) SubscribeChannels(ctx context.Context) error {
	sub, err := cl.rpcClient.SubscribeChannelEvents(
		ctx, &lnrpc.ChannelEventSubscription{},
	)
	if err != nil {
		return err
	}
	i := 0

	log.Println("Waiting")
	for i < 4 {
		u, err := sub.Recv()
		if err != nil {
			return err
		}

		i++

		log.Println("Event: ", u.Type)
	}

	return nil
}

func (cl *Client) SubscribeHtlcEvents(ctx context.Context) error {
	sub, err := cl.routerClient.SubscribeHtlcEvents(
		cl.macCtx(ctx), &routerrpc.SubscribeHtlcEventsRequest{},
	)
	if err != nil {
		return err
	}

	log.Println("Waiting for events")
	for {
		event, err := sub.Recv()
		if err != nil {
			return err
		}

		ts := time.Unix(int64(event.Timestamp), 0)

		fmt.Printf("Received htlc event ts: %v: (%v:%v) "+
			"-> (%v:%v), type: %v\n", ts, event.IncomingChannelId,
			event.IncomingHtlcId, event.OutgoingChannelId,
			event.OutgoingHtlcId, event.EventType)

		switch e := event.Event.(type) {
		case *routerrpc.HtlcEvent_ForwardEvent:
			fmt.Println("Forwarding event")

			info := e.ForwardEvent.Info
			fmt.Printf("Htlc info: (%v/%v) -> (%v/%v)\n",
				info.IncomingAmtMsat, info.IncomingTimelock,
				info.OutgoingAmtMsat, info.OutgoingTimelock)

		case *routerrpc.HtlcEvent_ForwardFailEvent:
			fmt.Println("Forwarding fail event")

		case *routerrpc.HtlcEvent_LinkFailEvent:
			fmt.Printf("Link fail event: wire: %v, detail: "+
				"%v, string: %v \n",
				e.LinkFailEvent.WireFailure,
				e.LinkFailEvent.FailureDetail,
				e.LinkFailEvent.FailureString)

			info := e.LinkFailEvent.Info
			fmt.Printf("Htlc info: Amount: (%v/%v), "+
				"Timelock (%v/%v)\n",
				info.IncomingAmtMsat, info.OutgoingAmtMsat,
				info.IncomingTimelock, info.OutgoingTimelock)

		case *routerrpc.HtlcEvent_SettleEvent:
			fmt.Println("Settle event")
		}

	}

	return nil
}

func (cl *Client) setupSubServers() error {
	if cl.requireUnlock {
		return errWalletLocked
	}

	conn, err := grpc.Dial(cl.lndGRPCAddress,
		grpc.WithTransportCredentials(cl.cert))
	if err != nil {
		return err
	}

	cl.rpcConn = conn

	if cl.enableRPCServer {
		cl.rpcClient = lnrpc.NewLightningClient(conn)
	}

	// TODO(add bool for this)
	cl.routerClient = routerrpc.NewRouterClient(conn)

	return nil
}
