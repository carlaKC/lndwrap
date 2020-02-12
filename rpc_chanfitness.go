package lndwrap

/*
// ChanFitnessRPC is a golang typed interface which wraps LND's
// chanfitness RPC interface.
type ChanFitnessRPC interface {
	GetScore(ctx context.Context, chanID uint64) (*Score, error)
}

type Score struct {
	Uptime    time.Duration
	Lifetime  time.Duration
	FlapCount int32
}

func (cl *Client) GetScore(ctx context.Context, chanID uint64) (*Score, error) {
	if !cl.enableChanFitSubServer {
		return nil, errNotEnabled
	}

	s, err := cl.chanFitClient.GetScore(cl.macCtx(ctx),
		&chanfitrpc.GetScoreRequest{
			ChanId: chanID,
		})
	if err != nil {
		return nil, err
	}

	return &Score{
		Uptime:    time.Second * time.Duration(s.GetUptimeSeconds()),
		Lifetime:  time.Second * time.Duration(s.GetLifetimeSeconds()),
		FlapCount: s.GetFlapCount(),
	}, err
}
*/
