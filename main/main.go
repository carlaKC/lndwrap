package main

import (
	"context"
	"flag"
	"log"

	"github.com/carlaKC/lndwrap"
)

var (
	lndAddr = flag.String("lnd_address", "localhost:10000",
		"address of LND grpc server")
	lndCert = flag.String("lnd_cert", "/Users/carla/dev/lightning/data/man/tls.cert",
		"location of LND tls certificate")
	lndMacaroon = flag.String("lnd_macaroon", "/Users/carla/dev/lightning/data/man/admin.macaroon",
		"location of LND macaroon (empty if none)")
)

func main() {
	flag.Parse()

	// opts contains the minimum options required to connect.
	opts := []lndwrap.Option{
		lndwrap.WithCert(*lndCert),
		lndwrap.WithAddress(*lndAddr),
	}

	// If a macaroon path has been provided, add the option.
	// It cannot be added by default because an empty path
	// would make the option fail
	if *lndMacaroon != "" {
		opts = append(opts, lndwrap.WithMacaroon(*lndMacaroon))
	}

	cl, err := lndwrap.Connect(opts...)
	if err != nil {
		log.Fatalf("Could not connect to LND: %v", err)
	}

	// Temp just subscribe to events and block.
	if err := cl.SubscribeHtlcEvents(context.Background()); err != nil {
		log.Fatalf(err.Error())
	}
}
