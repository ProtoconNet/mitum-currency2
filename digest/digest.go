package digest

import (
	"context"
	"github.com/ProtoconNet/mitum2/isaac"
	"sort"
	"sync"
	"time"

	"github.com/ProtoconNet/mitum2/base"
	isaacblock "github.com/ProtoconNet/mitum2/isaac/block"
	"github.com/ProtoconNet/mitum2/util"
	"github.com/ProtoconNet/mitum2/util/fixedtree"
	"github.com/ProtoconNet/mitum2/util/logging"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

type DigestError struct {
	err    error
	height base.Height
}

func NewDigestError(err error, height base.Height) DigestError {
	if err == nil {
		return DigestError{height: height}
	}

	return DigestError{err: err, height: height}
}

func (de DigestError) Error() string {
	if de.err == nil {
		return ""
	}

	return de.err.Error()
}

func (de DigestError) Height() base.Height {
	return de.height
}

func (de DigestError) IsError() bool {
	return de.err != nil
}

type Digester struct {
	sync.RWMutex
	*util.ContextDaemon
	*logging.Logging
	database      *Database
	localfsRoot   string
	blockChan     chan base.BlockMap
	errChan       chan error
	sourceReaders *isaac.BlockItemReaders
	fromRemotes   isaac.RemotesBlockItemReadFunc
	networkID     base.NetworkID
	buildInfo     string
}

func NewDigester(
	st *Database,
	root string,
	sourceReaders *isaac.BlockItemReaders,
	fromRemotes isaac.RemotesBlockItemReadFunc,
	networkID base.NetworkID,
	vs string,
	errChan chan error,
) *Digester {
	di := &Digester{
		Logging: logging.NewLogging(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "digester")
		}),
		database:      st,
		localfsRoot:   root,
		blockChan:     make(chan base.BlockMap, 200),
		errChan:       errChan,
		sourceReaders: sourceReaders,
		fromRemotes:   fromRemotes,
		networkID:     networkID,
		buildInfo:     vs,
	}

	di.ContextDaemon = util.NewContextDaemon(di.start)

	return di
}

func (di *Digester) start(ctx context.Context) error {
	e := util.StringError("start Digester")

	errch := func(err DigestError) {
		if di.errChan == nil {
			return
		}

		di.errChan <- err
	}

end:
	for {
		select {
		case <-ctx.Done():
			di.Log().Debug().Msg("stopped")

			break end
		case blk := <-di.blockChan:
			err := util.Retry(ctx, func() (bool, error) {
				if err := di.digest(ctx, blk); err != nil {
					go errch(NewDigestError(err, blk.Manifest().Height()))
					if errors.Is(err, context.Canceled) {
						return false, e.Wrap(err)
					}

					return true, e.Wrap(err)
				}

				return false, nil
			}, 15, time.Second*1)
			if err != nil {
				di.Log().Error().Err(err).Int64("block", blk.Manifest().Height().Int64()).Msg("digest block")
			} else {
				di.Log().Info().Int64("block", blk.Manifest().Height().Int64()).Msg("block digested")
			}

			go errch(NewDigestError(err, blk.Manifest().Height()))
		}
	}

	return nil
}

func (di *Digester) Digest(blocks []base.BlockMap) {
	sort.Slice(blocks, func(i, j int) bool {
		return blocks[i].Manifest().Height() < blocks[j].Manifest().Height()
	})

	for i := range blocks {
		blk := blocks[i]
		di.Log().Debug().Int64("block", blk.Manifest().Height().Int64()).Msg("start to digest block")

		di.blockChan <- blk
	}
}

func (di *Digester) digest(ctx context.Context, blk base.BlockMap) error {
	e := util.StringError("digest block")

	di.Lock()
	defer di.Unlock()

	var bm base.BlockMap

	switch i, found, err := isaac.BlockItemReadersDecode[base.BlockMap](di.sourceReaders.Item, blk.Manifest().Height(), base.BlockItemMap, nil); {
	case err != nil:
		return e.Wrap(err)
	case !found:
		return e.Wrap(util.ErrNotFound.Errorf("blockmap"))
	default:
		if err := i.IsValid(di.networkID); err != nil {
			return e.Wrap(err)
		}

		bm = i
	}

	pr, ops, sts, opsTree, _, _, err := isaacblock.LoadBlockItemsFromReader(bm, di.sourceReaders.Item, blk.Manifest().Height())
	if err != nil {
		return e.Wrap(err)
	}

	if err := DigestBlock(ctx, di.database, blk, ops, opsTree, sts, pr, di.buildInfo); err != nil {
		return e.Wrap(err)
	}

	return di.database.SetLastBlock(blk.Manifest().Height())
}

func DigestBlock(
	ctx context.Context,
	st *Database,
	blk base.BlockMap,
	ops []base.Operation,
	opsTree fixedtree.Tree,
	sts []base.State,
	proposal base.ProposalSignFact,
	vs string,
) error {
	if m, _, _, _, _, _ := st.ManifestByHeight(blk.Manifest().Height()); m != nil {
		return nil
	}

	bs, err := NewBlockSession(st, blk, ops, opsTree, sts, proposal, vs)
	if err != nil {
		return err
	}
	defer func() {
		_ = bs.Close()
	}()

	if err := bs.Prepare(); err != nil {
		return err
	}

	return bs.Commit(ctx)
}
