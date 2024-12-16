package extension

import (
	"context"
	"github.com/ProtoconNet/mitum-currency/v3/operation/extras"
	"sync"

	"github.com/ProtoconNet/mitum-currency/v3/common"
	"github.com/ProtoconNet/mitum-currency/v3/operation/currency"
	"github.com/ProtoconNet/mitum-currency/v3/state"
	statecurrency "github.com/ProtoconNet/mitum-currency/v3/state/currency"
	"github.com/ProtoconNet/mitum-currency/v3/state/extension"
	"github.com/ProtoconNet/mitum-currency/v3/types"
	"github.com/ProtoconNet/mitum2/base"

	"github.com/ProtoconNet/mitum2/util"
	"github.com/pkg/errors"
)

var withdrawItemProcessorPool = sync.Pool{
	New: func() interface{} {
		return new(WithdrawItemProcessor)
	},
}

var withdrawProcessorPool = sync.Pool{
	New: func() interface{} {
		return new(WithdrawProcessor)
	},
}

func (Withdraw) Process(
	_ context.Context, _ base.GetStateFunc,
) ([]base.StateMergeValue, base.OperationProcessReasonError, error) {
	// NOTE Process is nil func
	return nil, nil, nil
}

type WithdrawItemProcessor struct {
	h      util.Hash
	sender base.Address
	item   WithdrawItem
	tb     map[types.CurrencyID]base.StateMergeValue
}

func (opp *WithdrawItemProcessor) PreProcess(
	_ context.Context, _ base.Operation, getStateFunc base.GetStateFunc,
) error {
	e := util.StringError("preprocess WithdrawItemProcessor")

	_, cState, aErr, cErr := state.ExistsCAccount(opp.item.Target(), "target", true, true, getStateFunc)
	if aErr != nil {
		return e.Wrap(aErr)
	} else if cErr != nil {
		return e.Wrap(common.ErrAccTypeInvalid.Wrap(errors.Errorf("%v", cErr)))
	}

	status, err := extension.StateContractAccountValue(cState)
	if err != nil {
		return e.Wrap(common.ErrStateValInvalid.Wrap(err))
	}

	if !status.Owner().Equal(opp.sender) {
		return e.Wrap(common.ErrAccountNAth.Wrap(errors.Errorf("sender account is not contract account owner, %v", opp.sender)))

	}

	tb := map[types.CurrencyID]base.StateMergeValue{}
	for i := range opp.item.Amounts() {
		am := opp.item.Amounts()[i]

		_, err := state.ExistsCurrencyPolicy(am.Currency(), getStateFunc)
		if err != nil {
			return e.Wrap(err)
		}

		st, _, err := getStateFunc(statecurrency.BalanceStateKey(opp.item.Target(), am.Currency()))
		if err != nil {
			return e.Wrap(err)
		}

		balance, err := statecurrency.StateBalanceValue(st)
		if err != nil {
			return e.Wrap(err)
		}

		if balance.Big().Compare(am.Big()) < 0 {
			return errors.Errorf("insufficient contract account balance")
		}

		tb[am.Currency()] = common.NewBaseStateMergeValue(
			st.Key(),
			statecurrency.NewDeductBalanceStateValue(balance),
			func(height base.Height, st base.State) base.StateValueMerger {
				return statecurrency.NewBalanceStateValueMerger(
					height,
					st.Key(),
					am.Currency(),
					st,
				)
			},
		)
	}

	opp.tb = tb

	return nil
}

func (opp *WithdrawItemProcessor) Process(
	_ context.Context, _ base.Operation, _ base.GetStateFunc,
) ([]base.StateMergeValue, error) {
	sts := make([]base.StateMergeValue, len(opp.item.Amounts()))
	for i := range opp.item.Amounts() {
		am := opp.item.Amounts()[i]
		v, ok := opp.tb[am.Currency()].Value().(statecurrency.DeductBalanceStateValue)
		if !ok {
			return nil, errors.Errorf("expect DeductBalanceStateValue, not %T", opp.tb[am.Currency()].Value())
		}

		sts[i] = common.NewBaseStateMergeValue(
			opp.tb[am.Currency()].Key(),
			statecurrency.NewDeductBalanceStateValue(v.Amount.WithBig(am.Big())),
			func(height base.Height, st base.State) base.StateValueMerger {
				return statecurrency.NewBalanceStateValueMerger(height, opp.tb[am.Currency()].Key(), am.Currency(), st)
			},
		)
	}

	return sts, nil
}

func (opp *WithdrawItemProcessor) Close() {
	opp.h = nil
	opp.sender = nil
	opp.item = nil
	opp.tb = nil

	withdrawItemProcessorPool.Put(opp)
}

type WithdrawProcessor struct {
	*base.BaseOperationProcessor
	ns       []*WithdrawItemProcessor
	required map[types.CurrencyID][2]common.Big // required[0] : amount + fee, required[1] : fee
}

func NewWithdrawProcessor() types.GetNewProcessor {
	return func(
		height base.Height,
		getStateFunc base.GetStateFunc,
		newPreProcessConstraintFunc base.NewOperationProcessorProcessFunc,
		newProcessConstraintFunc base.NewOperationProcessorProcessFunc,
	) (base.OperationProcessor, error) {
		e := util.StringError("create new WithdrawProcessor")

		nopp := withdrawProcessorPool.Get()
		opp, ok := nopp.(*WithdrawProcessor)
		if !ok {
			return nil, e.WithMessage(nil, "expected WithdrawProcessor, not %T", nopp)
		}

		b, err := base.NewBaseOperationProcessor(
			height, getStateFunc, newPreProcessConstraintFunc, newProcessConstraintFunc)
		if err != nil {
			return nil, e.Wrap(err)
		}

		opp.BaseOperationProcessor = b

		return opp, nil
	}
}

func (opp *WithdrawProcessor) PreProcess(
	ctx context.Context, op base.Operation, getStateFunc base.GetStateFunc,
) (context.Context, base.OperationProcessReasonError, error) {
	fact, ok := op.Fact().(WithdrawFact)
	if !ok {
		return ctx, base.NewBaseOperationProcessReasonError(
			common.ErrMPreProcess.
				Wrap(common.ErrMTypeMismatch).
				Errorf("expected WithdrawFact, not %T", op.Fact())), nil
	}

	for i := range fact.items {
		cip := withdrawItemProcessorPool.Get()
		c, ok := cip.(*WithdrawItemProcessor)
		if !ok {
			return nil, base.NewBaseOperationProcessReasonError(
				common.ErrMPreProcess.
					Wrap(common.ErrMTypeMismatch).
					Errorf("expected WithdrawItemProcessor, not %T", cip)), nil
		}

		c.h = op.Hash()
		c.sender = fact.Sender()
		c.item = fact.items[i]

		if err := c.PreProcess(ctx, op, getStateFunc); err != nil {
			return nil, base.NewBaseOperationProcessReasonError(
				common.ErrMPreProcess.
					Errorf("%v", err)), nil
		}

		c.Close()
	}

	return ctx, nil, nil
}

func (opp *WithdrawProcessor) Process( // nolint:dupl
	ctx context.Context, op base.Operation, getStateFunc base.GetStateFunc) (
	[]base.StateMergeValue, base.OperationProcessReasonError, error,
) {
	fact, ok := op.Fact().(WithdrawFact)
	if !ok {
		return nil, base.NewBaseOperationProcessReasonError("expected WithdrawFact, not %T", op.Fact()), nil
	}

	ns := make([]*WithdrawItemProcessor, len(fact.items))
	for i := range fact.items {
		cip := withdrawItemProcessorPool.Get()
		c, ok := cip.(*WithdrawItemProcessor)
		if !ok {
			return nil, base.NewBaseOperationProcessReasonError("expected WithdrawItemProcessor, not %T", cip), nil
		}

		c.h = op.Hash()
		c.sender = fact.Sender()
		c.item = fact.items[i]

		if err := c.PreProcess(ctx, op, getStateFunc); err != nil {
			return nil, base.NewBaseOperationProcessReasonError("fail to preprocess WithdrawItem: %v", err), nil
		}

		ns[i] = c
	}

	var stateMergeValues []base.StateMergeValue // nolint:prealloc
	for i := range ns {
		s, err := ns[i].Process(ctx, op, getStateFunc)
		if err != nil {
			return nil, base.NewBaseOperationProcessReasonError("process WithdrawItem: %v", err), nil
		}
		stateMergeValues = append(stateMergeValues, s...)
	}

	var required map[types.CurrencyID][]common.Big
	switch i := op.Fact().(type) {
	case extras.FeeAble:
		required = i.FeeBase()
	default:
	}

	senderBalSts, totals, err := currency.PrepareSenderState(fact.Sender(), required, getStateFunc)
	if err != nil {
		return nil, base.NewBaseOperationProcessReasonError("process CreateAccount; %w", err), nil
	}

	for cid := range senderBalSts {
		v, ok := senderBalSts[cid].Value().(statecurrency.BalanceStateValue)
		if !ok {
			return nil, base.NewBaseOperationProcessReasonError(
				"expected %T, not %T",
				statecurrency.BalanceStateValue{},
				senderBalSts[cid].Value(),
			), nil
		}

		total, found := totals[cid]
		if found {
			stateMergeValues = append(
				stateMergeValues,
				common.NewBaseStateMergeValue(
					senderBalSts[cid].Key(),
					statecurrency.NewDeductBalanceStateValue(v.Amount.WithBig(total)),
					func(height base.Height, st base.State) base.StateValueMerger {
						return statecurrency.NewBalanceStateValueMerger(height, senderBalSts[cid].Key(), cid, st)
					}),
			)
		}
	}

	return stateMergeValues, nil, nil
}

func (opp *WithdrawProcessor) Close() error {
	for i := range opp.ns {
		opp.ns[i].Close()
	}

	opp.required = nil
	withdrawProcessorPool.Put(opp)

	return nil
}
