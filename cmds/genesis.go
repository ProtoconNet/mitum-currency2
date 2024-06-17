package cmds

import (
	"context"
	"fmt"
	"math"
	"os"

	"github.com/ProtoconNet/mitum-currency/v3/operation/currency"
	isaacoperation "github.com/ProtoconNet/mitum-currency/v3/operation/isaac"
	currencytypes "github.com/ProtoconNet/mitum-currency/v3/types"
	"github.com/ProtoconNet/mitum2/base"
	"github.com/ProtoconNet/mitum2/isaac"
	"github.com/ProtoconNet/mitum2/launch"
	"github.com/ProtoconNet/mitum2/util"
	"github.com/ProtoconNet/mitum2/util/encoder"
	"github.com/ProtoconNet/mitum2/util/hint"
	"github.com/ProtoconNet/mitum2/util/logging"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

type GenesisBlockGenerator struct {
	local                base.LocalNode
	encs                 *encoder.Encoders
	db                   isaac.Database
	proposal             base.ProposalSignFact
	ivp                  base.INITVoteproof
	avp                  base.ACCEPTVoteproof
	loadImportedBlockMap func() (base.BlockMap, bool, error)
	*logging.Logging
	dataroot  string
	networkID base.NetworkID
	facts     []base.Fact
	ops       []base.Operation
	ctx       context.Context
}

func NewGenesisBlockGenerator(
	local base.LocalNode,
	networkID base.NetworkID,
	encs *encoder.Encoders,
	db isaac.Database,
	dataroot string,
	facts []base.Fact,
	loadImportedBlockMap func() (base.BlockMap, bool, error),
	ctx context.Context,
) *GenesisBlockGenerator {
	return &GenesisBlockGenerator{
		Logging: logging.NewLogging(func(zctx zerolog.Context) zerolog.Context {
			return zctx.Str("module", "genesis-block-generator")
		}),
		local:                local,
		networkID:            networkID,
		encs:                 encs,
		db:                   db,
		dataroot:             dataroot,
		facts:                facts,
		loadImportedBlockMap: loadImportedBlockMap,
		ctx:                  ctx,
	}
}

func (g *GenesisBlockGenerator) Generate() (base.BlockMap, error) {
	e := util.StringError("generate genesis block")

	if err := g.generateOperations(); err != nil {
		return nil, e.Wrap(err)
	}

	if err := g.newProposal(nil); err != nil {
		return nil, e.Wrap(err)
	}

	if err := g.process(); err != nil {
		return nil, e.Wrap(err)
	}

	switch blockmap, found, err := g.loadImportedBlockMap(); {
	case err != nil:
		return nil, e.Wrap(err)
	case !found:
		return nil, util.ErrNotFound.Errorf("Blockmap")
	default:
		if err := blockmap.IsValid(g.networkID); err != nil {
			return nil, e.Wrap(err)
		}

		g.Log().Info().Interface("blockmap", blockmap).Msg("genesis block generated")

		if err := g.closeDatabase(); err != nil {
			return nil, e.Wrap(err)
		}

		return blockmap, nil
	}
}

func (g *GenesisBlockGenerator) generateOperations() error {
	g.ops = make([]base.Operation, len(g.facts))

	types := map[string]struct{}{}

	for i := range g.facts {
		fact := g.facts[i]

		var err error

		hinter, ok := fact.(hint.Hinter)
		if !ok {
			return errors.Errorf("Fact does not support Hinter")
		}

		switch ht := hinter.Hint(); {
		case ht.IsCompatible(isaacoperation.SuffrageGenesisJoinFactHint):
			if _, found := types[ht.String()]; found {
				return errors.Errorf("Multiple join operation found")
			}

			g.ops[i], err = g.joinOperation(fact)
		case ht.IsCompatible(isaacoperation.GenesisNetworkPolicyFactHint):
			if _, found := types[ht.String()]; found {
				return errors.Errorf("Multiple network policy operation found")
			}

			g.ops[i], err = g.networkPolicyOperation(fact)
		case ht.IsCompatible(currency.RegisterGenesisCurrencyFactHint):
			if _, found := types[ht.String()]; found {
				return errors.Errorf("Multiple RegisterGenesisCurrency operation found")
			}

			g.ops[i], err = g.registerGenesisCurrencyOperation(fact, g.networkID)
		}

		if err != nil {
			return err
		}

		types[hinter.Hint().String()] = struct{}{}
	}

	return nil
}

func (g *GenesisBlockGenerator) joinOperation(i base.Fact) (base.Operation, error) {
	e := util.StringError("make join operation")

	basefact, ok := i.(isaacoperation.SuffrageGenesisJoinFact)
	if !ok {
		return nil, e.WithMessage(nil, "expected SuffrageGenesisJoinFact, not %T", i)
	}

	fact := isaacoperation.NewSuffrageGenesisJoinFact(basefact.Nodes(), g.networkID)

	if err := fact.IsValid(g.networkID); err != nil {
		return nil, e.Wrap(err)
	}

	op := isaacoperation.NewSuffrageGenesisJoin(fact)
	if err := op.Sign(g.local.Privatekey(), g.networkID); err != nil {
		return nil, e.Wrap(err)
	}

	g.Log().Debug().Interface("operation", op).Msg("genesis join operation created")

	return op, nil
}

func (g *GenesisBlockGenerator) networkPolicyOperation(i base.Fact) (base.Operation, error) {
	e := util.StringError("make networkPolicy operation")

	basefact, ok := i.(isaacoperation.GenesisNetworkPolicyFact)
	if !ok {
		return nil, e.WithMessage(nil, "expected GenesisNetworkPolicyFact, not %T", i)
	}

	fact := isaacoperation.NewGenesisNetworkPolicyFact(basefact.Policy())

	if err := fact.IsValid(nil); err != nil {
		return nil, e.Wrap(err)
	}

	op := isaacoperation.NewGenesisNetworkPolicy(fact)
	if err := op.Sign(g.local.Privatekey(), g.networkID); err != nil {
		return nil, e.Wrap(err)
	}

	g.Log().Debug().Interface("operation", op).Msg("genesis network policy operation created")

	return op, nil
}

func (g *GenesisBlockGenerator) registerGenesisCurrencyOperation(i base.Fact, token []byte) (base.Operation, error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Panic:", r)
			os.Exit(1)
		}
	}()
	e := util.StringError("make registerGenesisCurrency operation")

	basefact, ok := i.(currency.RegisterGenesisCurrencyFact)
	if !ok {
		return nil, e.WithMessage(nil, "expected RegisterGenesisCurrencyFact, not %T", i)
	}
	acks, err := currencytypes.NewBaseAccountKeys(basefact.Keys().Keys(), basefact.Keys().Threshold())
	if err != nil {
		return nil, e.Wrap(err)
	}

	var design launch.NodeDesign
	err = util.LoadFromContextOK(g.ctx,
		launch.DesignContextKey, &design,
	)
	if err != nil {
		return nil, e.Wrap(err)
	}

	if !basefact.GenesisNodeKey().Equal(design.Privatekey.Publickey()) {
		panic(errors.Errorf(
			"GenesisNodeKey, %v is not match with local node key, %v",
			basefact.GenesisNodeKey().String(),
			design.Privatekey.Publickey().String(),
		))
	}

	fact := currency.NewRegisterGenesisCurrencyFact(token, basefact.GenesisNodeKey(), acks, basefact.Currencies())
	if err := fact.IsValid(g.networkID); err != nil {
		return nil, e.Wrap(err)
	}
	op := currency.NewRegisterGenesisCurrency(fact)
	if err := op.Sign(g.local.Privatekey(), g.networkID); err != nil {
		return nil, e.Wrap(err)
	}
	g.Log().Debug().Interface("operation", op).Msg("genesis join operation created")

	return op, nil
}

func (g *GenesisBlockGenerator) newProposal(ops [][2]util.Hash) error {
	e := util.StringError("make genesis proposal")

	nops := make([][2]util.Hash, len(ops)+len(g.ops))
	copy(nops[:len(ops)], ops)

	for i := range g.ops {
		nops[i+len(ops)][0] = g.ops[i].Hash()
		nops[i+len(ops)][1] = g.ops[i].Fact().Hash()
	}

	fact := isaac.NewProposalFact(base.GenesisPoint, g.local.Address(), nil, nops)
	sign := isaac.NewProposalSignFact(fact)

	if err := sign.Sign(g.local.Privatekey(), g.networkID); err != nil {
		return e.Wrap(err)
	}

	if err := sign.IsValid(g.networkID); err != nil {
		return e.Wrap(err)
	}

	g.proposal = sign

	g.Log().Debug().Interface("proposal", sign).Msg("proposal created for genesis")

	return nil
}

func (g *GenesisBlockGenerator) initVoetproof() error {
	e := util.StringError("make genesis init voteproof")

	fact := isaac.NewINITBallotFact(base.GenesisPoint, nil, g.proposal.Fact().Hash(), nil)
	if err := fact.IsValid(nil); err != nil {
		return e.Wrap(err)
	}

	sf := isaac.NewINITBallotSignFact(fact)
	if err := sf.NodeSign(g.local.Privatekey(), g.networkID, g.local.Address()); err != nil {
		return e.Wrap(err)
	}

	if err := sf.IsValid(g.networkID); err != nil {
		return e.Wrap(err)
	}

	vp := isaac.NewINITVoteproof(fact.Point().Point)
	vp.
		SetMajority(fact).
		SetSignFacts([]base.BallotSignFact{sf}).
		SetThreshold(base.MaxThreshold).
		Finish()

	if err := vp.IsValid(g.networkID); err != nil {
		return e.Wrap(err)
	}

	g.ivp = vp

	g.Log().Debug().Interface("init_voteproof", vp).Msg("init voteproof created for genesis")

	return nil
}

func (g *GenesisBlockGenerator) acceptVoteproof(proposal, newblock util.Hash) error {
	e := util.StringError("make genesis accept voteproof")

	fact := isaac.NewACCEPTBallotFact(base.GenesisPoint, proposal, newblock, nil)
	if err := fact.IsValid(nil); err != nil {
		return e.Wrap(err)
	}

	sf := isaac.NewACCEPTBallotSignFact(fact)
	if err := sf.NodeSign(g.local.Privatekey(), g.networkID, g.local.Address()); err != nil {
		return e.Wrap(err)
	}

	if err := sf.IsValid(g.networkID); err != nil {
		return e.Wrap(err)
	}

	vp := isaac.NewACCEPTVoteproof(fact.Point().Point)
	vp.
		SetMajority(fact).
		SetSignFacts([]base.BallotSignFact{sf}).
		SetThreshold(base.MaxThreshold).
		Finish()

	if err := vp.IsValid(g.networkID); err != nil {
		return e.Wrap(err)
	}

	g.avp = vp

	g.Log().Debug().Interface("init_voteproof", vp).Msg("accept voteproof created for genesis")

	return nil
}

func (g *GenesisBlockGenerator) process() error {
	e := util.StringError("process blockgenerator")

	if err := g.initVoetproof(); err != nil {
		return e.Wrap(err)
	}

	pp, err := g.newProposalProcessor()
	if err != nil {
		return e.Wrap(err)
	}

	_ = pp.SetLogging(g.Logging)

	switch m, err := pp.Process(context.Background(), g.ivp); {
	case err != nil:
		return e.Wrap(err)
	default:
		if err := m.IsValid(g.networkID); err != nil {
			return e.Wrap(err)
		}

		g.Log().Info().Interface("manifest", m).Msg("genesis block generated")

		if err := g.acceptVoteproof(g.proposal.Fact().Hash(), m.Hash()); err != nil {
			return e.Wrap(err)
		}
	}

	if _, err := pp.Save(context.Background(), g.avp); err != nil {
		return e.Wrap(err)
	}

	return nil
}

func (g *GenesisBlockGenerator) closeDatabase() error {
	e := util.StringError("close database")

	if err := g.db.MergeAllPermanent(); err != nil {
		return e.WithMessage(err, "merge temps")
	}

	return nil
}

func (g *GenesisBlockGenerator) newProposalProcessor() (*isaac.DefaultProposalProcessor, error) {
	args := isaac.NewDefaultProposalProcessorArgs()
	args.NewWriterFunc = launch.NewBlockWriterFunc(
		g.local, g.networkID, g.dataroot, g.encs.JSON(), g.encs.Default(), g.db, math.MaxInt16, 0)
	args.GetStateFunc = func(key string) (base.State, bool, error) {
		return nil, false, nil
	}
	args.GetOperationFunc = func(_ context.Context, operationhash, _ util.Hash) (base.Operation, error) {
		for i := range g.ops {
			op := g.ops[i]
			if operationhash.Equal(op.Hash()) {
				return op, nil
			}
		}

		return nil, util.ErrNotFound.Errorf("Operation not found")
	}

	return isaac.NewDefaultProposalProcessor(g.proposal, nil, args)
}
