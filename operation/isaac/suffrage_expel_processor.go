package isaacoperation

import (
	"context"
	"github.com/ProtoconNet/mitum-currency/v3/common"
	"github.com/ProtoconNet/mitum2/base"
	"github.com/ProtoconNet/mitum2/isaac"
	"github.com/ProtoconNet/mitum2/util"
)

var ExpelPreProcessedContextKey = util.ContextKey("expel-preprocessed")

type SuffrageExpelProcessor struct {
	*base.BaseOperationProcessor
	sufstv       base.SuffrageNodesStateValue
	suffrage     base.Suffrage
	preprocessed map[string]struct{} //revive:disable-line:nested-structs
}

func NewSuffrageExpelProcessor(
	height base.Height,
	getStateFunc base.GetStateFunc,
	newPreProcessConstraintFunc base.NewOperationProcessorProcessFunc,
	newProcessConstraintFunc base.NewOperationProcessorProcessFunc,
) (*SuffrageExpelProcessor, error) {
	e := util.StringError("create new SuffrageExpelProcessor")

	b, err := base.NewBaseOperationProcessor(
		height, getStateFunc, newPreProcessConstraintFunc, newProcessConstraintFunc)
	if err != nil {
		return nil, e.Wrap(err)
	}

	p := &SuffrageExpelProcessor{
		BaseOperationProcessor: b,
		preprocessed:           map[string]struct{}{},
	}

	switch i, found, err := getStateFunc(isaac.SuffrageStateKey); {
	case err != nil:
		return nil, e.Wrap(err)
	case !found, i == nil:
		return nil, e.Errorf("Empty state")
	default:
		p.sufstv = i.Value().(base.SuffrageNodesStateValue) //nolint:forcetypeassert //...

		suf, err := p.sufstv.Suffrage()
		if err != nil {
			return nil, e.Errorf("get suffrage from state")
		}

		p.suffrage = suf
	}

	return p, nil
}

func (p *SuffrageExpelProcessor) Close() error {
	if err := p.BaseOperationProcessor.Close(); err != nil {
		return err
	}

	p.sufstv = nil
	p.suffrage = nil
	p.preprocessed = nil

	return nil
}

func (p *SuffrageExpelProcessor) PreProcess(ctx context.Context, op base.Operation, getStateFunc base.GetStateFunc) (
	context.Context, base.OperationProcessReasonError, error,
) {
	e := util.StringError("preprocess for SuffrageExpel")

	fact := op.Fact().(base.SuffrageExpelFact) //nolint:forcetypeassert //...

	switch {
	case fact.ExpelStart() > p.Height():
		return ctx, base.NewBaseOperationProcessReasonError("wrong start height"), nil
	case fact.ExpelEnd() < p.Height():
		return ctx, base.NewBaseOperationProcessReasonError("expired"), nil
	}

	n := fact.Node()

	if _, found := p.preprocessed[n.String()]; found {
		return ctx, base.NewBaseOperationProcessReasonError("already preprocessed, %v", n), nil
	}

	if !p.suffrage.Exists(n) {
		return ctx, base.NewBaseOperationProcessReasonError("not in suffrage, %v", n), nil
	}

	switch reasonerr, err := p.PreProcessConstraintFunc(ctx, op, getStateFunc); {
	case err != nil:
		return ctx, nil, e.Wrap(err)
	case reasonerr != nil:
		return ctx, reasonerr, nil
	}

	p.preprocessed[n.String()] = struct{}{}

	var preprocessed []base.Address

	_ = util.LoadFromContext(ctx, ExpelPreProcessedContextKey, &preprocessed)
	preprocessed = append(preprocessed, n)

	ctx = context.WithValue(ctx, ExpelPreProcessedContextKey, preprocessed) //revive:disable-line:modifies-parameter

	return ctx, nil, nil
}

func (p *SuffrageExpelProcessor) Process(ctx context.Context, op base.Operation, getStateFunc base.GetStateFunc) (
	[]base.StateMergeValue, base.OperationProcessReasonError, error,
) {
	e := util.StringError("process for SuffrageWithdraw")

	switch reasonerr, err := p.ProcessConstraintFunc(ctx, op, getStateFunc); {
	case err != nil:
		return nil, nil, e.Wrap(err)
	case reasonerr != nil:
		return nil, reasonerr, nil
	}

	fact := op.Fact().(base.SuffrageExpelFact) //nolint:forcetypeassert //...

	return []base.StateMergeValue{
		common.NewBaseStateMergeValue(
			isaac.SuffrageStateKey,
			newSuffrageDisjoinNodeStateValue(fact.Node()),
			func(height base.Height, st base.State) base.StateValueMerger {
				return NewSuffrageJoinStateValueMerger(height, st)
			},
		),
	}, nil, nil
}
