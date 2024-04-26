package currency

import (
	"context"
	"sync"

	"github.com/ProtoconNet/mitum-currency/v3/common"
	"github.com/ProtoconNet/mitum-currency/v3/state"
	"github.com/ProtoconNet/mitum-currency/v3/state/currency"
	"github.com/ProtoconNet/mitum-currency/v3/state/extension"
	"github.com/ProtoconNet/mitum-currency/v3/types"
	"github.com/ProtoconNet/mitum2/base"

	"github.com/ProtoconNet/mitum2/util"
	"github.com/pkg/errors"
)

var transferItemProcessorPool = sync.Pool{
	New: func() interface{} {
		return new(TransferItemProcessor)
	},
}

var transferProcessorPool = sync.Pool{
	New: func() interface{} {
		return new(TransferProcessor)
	},
}

func (Transfer) Process(
	_ context.Context, _ base.GetStateFunc,
) ([]base.StateMergeValue, base.OperationProcessReasonError, error) {
	// NOTE Process is nil func
	return nil, nil, nil
}

type TransferItemProcessor struct {
	h    util.Hash
	item TransferItem
	rb   map[types.CurrencyID]base.StateMergeValue
}

func (opp *TransferItemProcessor) PreProcess(
	_ context.Context, _ base.Operation, getStateFunc base.GetStateFunc,
) error {
	rb := map[types.CurrencyID]base.StateMergeValue{}
	for i := range opp.item.Amounts() {
		am := opp.item.Amounts()[i]

		_, err := state.ExistsCurrencyPolicy(am.Currency(), getStateFunc)
		if err != nil {
			return err
		}

		st, _, err := getStateFunc(currency.StateKeyBalance(opp.item.Receiver(), am.Currency()))
		if err != nil {
			return err
		}

		var balance types.Amount
		if st == nil {
			balance = types.NewZeroAmount(am.Currency())
		} else {
			balance, err = currency.StateBalanceValue(st)
			if err != nil {
				return err
			}
		}

		rb[am.Currency()] = common.NewBaseStateMergeValue(
			currency.StateKeyBalance(opp.item.Receiver(), am.Currency()),
			currency.NewAddBalanceStateValue(balance),
			func(height base.Height, st base.State) base.StateValueMerger {
				return currency.NewBalanceStateValueMerger(
					height,
					currency.StateKeyBalance(opp.item.Receiver(), am.Currency()),
					am.Currency(),
					st,
				)
			},
		)
	}

	opp.rb = rb

	return nil
}

func (opp *TransferItemProcessor) Process(
	_ context.Context, _ base.Operation, getStateFunc base.GetStateFunc,
) ([]base.StateMergeValue, error) {
	e := util.StringError("process TransferItemProcessor")

	var sts []base.StateMergeValue
	k := currency.StateKeyAccount(opp.item.Receiver())
	switch _, found, err := getStateFunc(k); {
	case err != nil:
		return nil, e.Wrap(err)
	case !found:
		nilKys, err := types.NewNilAccountKeysFromAddress(opp.item.Receiver())
		if err != nil {
			return nil, e.Wrap(err)
		}
		acc, err := types.NewAccount(opp.item.Receiver(), nilKys)
		if err != nil {
			return nil, e.Wrap(err)
		}

		sts = append(sts, state.NewStateMergeValue(k, currency.NewAccountStateValue(acc)))
	default:
	}

	for i := range opp.item.Amounts() {
		am := opp.item.Amounts()[i]
		v, ok := opp.rb[am.Currency()].Value().(currency.AddBalanceStateValue)
		if !ok {
			return nil, e.Wrap(errors.Errorf("not AddBalanceStateValue, %T", opp.rb[am.Currency()].Value()))
		}
		//stv := currency.NewBalanceStateValue(v.Amount.WithBig(v.Amount.Big().Add(am.Big())))
		//sts[i] = state.NewStateMergeValue(opp.rb[am.Currency()].Key(), stv)
		sts = append(sts, common.NewBaseStateMergeValue(
			opp.rb[am.Currency()].Key(),
			currency.NewAddBalanceStateValue(v.Amount.WithBig(am.Big())),
			func(height base.Height, st base.State) base.StateValueMerger {
				return currency.NewBalanceStateValueMerger(height, opp.rb[am.Currency()].Key(), am.Currency(), st)
			},
		))
	}

	return sts, nil
}

func (opp *TransferItemProcessor) Close() {
	opp.h = nil
	opp.item = nil
	opp.rb = nil

	transferItemProcessorPool.Put(opp)
}

type TransferProcessor struct {
	*base.BaseOperationProcessor
	ns       []*TransferItemProcessor
	required map[types.CurrencyID][2]common.Big
}

func NewTransferProcessor() types.GetNewProcessor {
	return func(
		height base.Height,
		getStateFunc base.GetStateFunc,
		newPreProcessConstraintFunc base.NewOperationProcessorProcessFunc,
		newProcessConstraintFunc base.NewOperationProcessorProcessFunc,
	) (base.OperationProcessor, error) {
		e := util.StringError("create new TransferProcessor")

		nopp := transferProcessorPool.Get()
		opp, ok := nopp.(*TransferProcessor)
		if !ok {
			return nil, e.Wrap(errors.Errorf("expected TransferProcessor, not %T", nopp))
		}

		b, err := base.NewBaseOperationProcessor(
			height, getStateFunc, newPreProcessConstraintFunc, newProcessConstraintFunc)
		if err != nil {
			return nil, e.Wrap(err)
		}

		opp.BaseOperationProcessor = b
		opp.ns = nil
		opp.required = nil

		return opp, nil
	}
}

func (opp *TransferProcessor) PreProcess(
	ctx context.Context, op base.Operation, getStateFunc base.GetStateFunc,
) (context.Context, base.OperationProcessReasonError, error) {
	fact, ok := op.Fact().(TransferFact)
	if !ok {
		return ctx, base.NewBaseOperationProcessReasonError(
			"expected %T, not %T", TransferFact{}, op.Fact(),
		), nil
	}

	if err := state.CheckExistsState(currency.StateKeyAccount(fact.sender), getStateFunc); err != nil {
		return ctx, base.NewBaseOperationProcessReasonError("sender account not found, %v; %w", fact.sender, err), nil
	}

	if err := state.CheckNotExistsState(extension.StateKeyContractAccount(fact.Sender()), getStateFunc); err != nil {
		return ctx, base.NewBaseOperationProcessReasonError("contract account cannot transfer currency, %v; %w", fact.Sender(), err), nil
	}

	if err := state.CheckFactSignsByState(fact.sender, op.Signs(), getStateFunc); err != nil {
		return ctx, base.NewBaseOperationProcessReasonError("invalid signing :  %w", err), nil
	}

	for i := range fact.items {
		tip := transferItemProcessorPool.Get()
		t, ok := tip.(*TransferItemProcessor)
		if !ok {
			return nil, base.NewBaseOperationProcessReasonError("expected %T, not %T", &TransferItemProcessor{}, tip), nil
		}

		t.h = op.Hash()
		t.item = fact.items[i]

		if err := t.PreProcess(ctx, op, getStateFunc); err != nil {
			return nil, base.NewBaseOperationProcessReasonError("fail to preprocess transfer item; %w", err), nil
		}
		t.Close()
	}

	return ctx, nil, nil
}

func (opp *TransferProcessor) Process( // nolint:dupl
	ctx context.Context, op base.Operation, getStateFunc base.GetStateFunc) (
	[]base.StateMergeValue, base.OperationProcessReasonError, error,
) {
	fact, ok := op.Fact().(TransferFact)
	if !ok {
		return nil, base.NewBaseOperationProcessReasonError("expected %T, not %T", TransferFact{}, op.Fact()), nil
	}

	var (
		senderBalSts, feeReceiverBalSts map[types.CurrencyID]base.State
		required                        map[types.CurrencyID][2]common.Big
		err                             error
	)

	if feeReceiverBalSts, required, err = opp.calculateItemsFee(op, getStateFunc); err != nil {
		return nil, base.NewBaseOperationProcessReasonError("calculate fee; %w", err), nil
	} else if senderBalSts, err = CheckEnoughBalance(fact.sender, required, getStateFunc); err != nil {
		return nil, base.NewBaseOperationProcessReasonError("check enough balance; %w", err), nil
	} else {
		opp.required = required
	}

	ns := make([]*TransferItemProcessor, len(fact.items))
	for i := range fact.items {
		cip := transferItemProcessorPool.Get()
		c, ok := cip.(*TransferItemProcessor)
		if !ok {
			return nil, base.NewBaseOperationProcessReasonError("expected %T, not %T", &TransferItemProcessor{}, cip), nil
		}

		c.h = op.Hash()
		c.item = fact.items[i]

		if err := c.PreProcess(ctx, op, getStateFunc); err != nil {
			return nil, base.NewBaseOperationProcessReasonError("fail to preprocess transfer item; %w", err), nil
		}

		ns[i] = c
	}
	opp.ns = ns

	var stmvs []base.StateMergeValue // nolint:prealloc
	for i := range opp.ns {
		s, err := opp.ns[i].Process(ctx, op, getStateFunc)
		if err != nil {
			return nil, base.NewBaseOperationProcessReasonError("process transfer item; %w", err), nil
		}
		stmvs = append(stmvs, s...)
	}

	for cid := range senderBalSts {
		v, ok := senderBalSts[cid].Value().(currency.BalanceStateValue)
		if !ok {
			return nil, base.NewBaseOperationProcessReasonError("expected %T, not %T", currency.BalanceStateValue{}, senderBalSts[cid].Value()), nil
		}

		_, feeReceiverFound := feeReceiverBalSts[cid]

		var stmv base.StateMergeValue
		if feeReceiverFound && (senderBalSts[cid].Key() == feeReceiverBalSts[cid].Key()) {
			stmv = common.NewBaseStateMergeValue(
				senderBalSts[cid].Key(),
				currency.NewDeductBalanceStateValue(v.Amount.WithBig(opp.required[cid][0].Sub(opp.required[cid][1]))),
				func(height base.Height, st base.State) base.StateValueMerger {
					return currency.NewBalanceStateValueMerger(height, senderBalSts[cid].Key(), cid, st)
				},
			)
		} else {
			stmv = common.NewBaseStateMergeValue(
				senderBalSts[cid].Key(),
				currency.NewDeductBalanceStateValue(v.Amount.WithBig(opp.required[cid][0])),
				func(height base.Height, st base.State) base.StateValueMerger {
					return currency.NewBalanceStateValueMerger(height, senderBalSts[cid].Key(), cid, st)
				},
			)
			if feeReceiverFound {
				r, ok := feeReceiverBalSts[cid].Value().(currency.BalanceStateValue)
				if !ok {
					return nil, base.NewBaseOperationProcessReasonError("expected %T, not %T", currency.BalanceStateValue{}, feeReceiverBalSts[cid].Value()), nil
				}
				stmvs = append(
					stmvs,
					common.NewBaseStateMergeValue(
						feeReceiverBalSts[cid].Key(),
						currency.NewAddBalanceStateValue(r.Amount.WithBig(opp.required[cid][1])),
						func(height base.Height, st base.State) base.StateValueMerger {
							return currency.NewBalanceStateValueMerger(height, feeReceiverBalSts[cid].Key(), cid, st)
						},
					),
				)
			}
		}
		stmvs = append(stmvs, stmv)
	}

	return stmvs, nil, nil
}

func (opp *TransferProcessor) Close() error {
	for i := range opp.ns {
		opp.ns[i].Close()
	}

	opp.ns = nil
	opp.required = nil

	transferProcessorPool.Put(opp)

	return nil
}

func (opp *TransferProcessor) calculateItemsFee(op base.Operation, getStateFunc base.GetStateFunc) (map[types.CurrencyID]base.State, map[types.CurrencyID][2]common.Big, error) {
	fact, ok := op.Fact().(TransferFact)
	if !ok {
		return nil, nil, errors.Errorf("expected %T, not %T", TransferFact{}, op.Fact())
	}
	items := make([]AmountsItem, len(fact.items))
	for i := range fact.items {
		items[i] = fact.items[i]
	}

	return CalculateItemsFee(getStateFunc, items)
}
