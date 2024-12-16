package currency

import (
	"context"
	"fmt"
	"github.com/ProtoconNet/mitum-currency/v3/common"
	"github.com/ProtoconNet/mitum-currency/v3/operation/extras"
	"github.com/ProtoconNet/mitum-currency/v3/state"
	"github.com/ProtoconNet/mitum-currency/v3/state/currency"
	"github.com/ProtoconNet/mitum-currency/v3/types"
	"github.com/ProtoconNet/mitum2/base"
	"github.com/ProtoconNet/mitum2/util"
	"github.com/pkg/errors"
	"sync"
)

var createAccountItemProcessorPool = sync.Pool{
	New: func() interface{} {
		return new(CreateAccountItemProcessor)
	},
}

var createAccountProcessorPool = sync.Pool{
	New: func() interface{} {
		return new(CreateAccountProcessor)
	},
}

func (CreateAccount) Process(
	_ context.Context, _ base.GetStateFunc,
) ([]base.StateMergeValue, base.OperationProcessReasonError, error) {
	// NOTE Process is nil func
	return nil, nil, nil
}

type CreateAccountItemProcessor struct {
	h    util.Hash
	item CreateAccountItem
}

func (opp *CreateAccountItemProcessor) PreProcess(
	_ context.Context, _ base.Operation, getStateFunc base.GetStateFunc,
) error {
	e := util.StringError("preprocess CreateAccountItemProcessor")

	target, err := opp.item.Address()
	if err != nil {
		return e.Wrap(err)
	}

	_, err = state.ExistsAccount(target, "target", false, getStateFunc)
	if err != nil {
		return e.Wrap(err)
	}

	amounts := opp.item.Amounts()

	for i := range amounts {
		am := amounts[i]
		cid := am.Currency()
		k := currency.BalanceStateKey(target, cid)
		policy, err := state.ExistsCurrencyPolicy(cid, getStateFunc)
		if err != nil {
			return e.Wrap(err)
		}

		if am.Big().Compare(policy.MinBalance()) < 0 {
			return e.Wrap(
				common.ErrValOOR.Wrap(
					errors.Errorf(
						"amount under new account minimum balance, %v < %v", am.Big(), policy.MinBalance())))
		}

		switch _, found, err := getStateFunc(k); {
		case err != nil:
			return e.Wrap(err)
		case found:
			return e.Wrap(common.ErrAccountE.Wrap(errors.Errorf("target balance already exists, %v", target)))
		default:
		}
	}

	return nil
}

func (opp *CreateAccountItemProcessor) Process(
	_ context.Context, _ base.Operation, _ base.GetStateFunc,
) ([]base.StateMergeValue, error) {
	e := util.StringError("process CreateAccountItemProcessor")

	nac, err := types.NewAccountFromKeys(opp.item.Keys())
	if err != nil {
		return nil, e.Wrap(err)
	}

	if err = nac.IsValid(nil); err != nil {
		return nil, e.Wrap(err)
	}

	target, _ := opp.item.Address()

	sts := make([]base.StateMergeValue, len(opp.item.Amounts())+1)
	sts[0] = state.NewStateMergeValue(currency.AccountStateKey(target), currency.NewAccountStateValue(nac))

	amounts := opp.item.Amounts()
	for i := range amounts {
		am := amounts[i]
		cid := am.Currency()
		k := currency.BalanceStateKey(target, cid)

		sts[i+1] = common.NewBaseStateMergeValue(
			currency.BalanceStateKey(target, cid),
			currency.NewAddBalanceStateValue(types.NewAmount(am.Big(), cid)),
			func(height base.Height, st base.State) base.StateValueMerger {
				return currency.NewBalanceStateValueMerger(height, k, cid, st)
			},
		)
	}

	return sts, nil
}

func (opp *CreateAccountItemProcessor) Close() {
	opp.h = nil
	opp.item = nil

	createAccountItemProcessorPool.Put(opp)
}

type CreateAccountProcessor struct {
	*base.BaseOperationProcessor
	ns       []*CreateAccountItemProcessor
	required map[types.CurrencyID][2]common.Big // required[0] : amount + fee, required[1] : fee
}

func NewCreateAccountProcessor() types.GetNewProcessor {
	return func(
		height base.Height,
		getStateFunc base.GetStateFunc,
		newPreProcessConstraintFunc base.NewOperationProcessorProcessFunc,
		newProcessConstraintFunc base.NewOperationProcessorProcessFunc,
	) (base.OperationProcessor, error) {
		e := util.StringError("create new CreateAccountProcessor")

		nOpp := createAccountProcessorPool.Get()
		opp, ok := nOpp.(*CreateAccountProcessor)
		if !ok {
			return nil, errors.Errorf("expected %T, not %T", &CreateAccountProcessor{}, nOpp)
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

func (opp *CreateAccountProcessor) PreProcess(
	ctx context.Context, op base.Operation, getStateFunc base.GetStateFunc,
) (context.Context, base.OperationProcessReasonError, error) {
	fact, ok := op.Fact().(CreateAccountFact)
	if !ok {
		return ctx,
			base.NewBaseOperationProcessReasonError(
				common.ErrMPreProcess.
					Wrap(common.ErrMTypeMismatch).
					Errorf("expected %T, not %T", CreateAccountFact{}, op.Fact()),
			),
			nil
	}

	for i := range fact.items {
		cip := createAccountItemProcessorPool.Get()
		c, ok := cip.(*CreateAccountItemProcessor)
		if !ok {
			return nil, base.NewBaseOperationProcessReasonError(
				common.ErrMPreProcess.
					Wrap(common.ErrMTypeMismatch).
					Errorf("expected %T, not %T", &CreateAccountItemProcessor{}, cip),
			), nil
		}

		c.h = op.Hash()
		c.item = fact.items[i]

		if err := c.PreProcess(ctx, op, getStateFunc); err != nil {
			return nil, base.NewBaseOperationProcessReasonError(
				common.ErrMPreProcess.Errorf("%v", err)), nil
		}

		c.Close()
	}

	return ctx, nil, nil
}

func (opp *CreateAccountProcessor) Process( // nolint:dupl
	ctx context.Context, op base.Operation, getStateFunc base.GetStateFunc) (
	[]base.StateMergeValue, base.OperationProcessReasonError, error,
) {
	fact, _ := op.Fact().(CreateAccountFact)

	ns := make([]*CreateAccountItemProcessor, len(fact.items))
	for i := range fact.items {
		cip := createAccountItemProcessorPool.Get()
		c, ok := cip.(*CreateAccountItemProcessor)
		if !ok {
			return nil, base.NewBaseOperationProcessReasonError(
				"expected %T, not %T",
				&CreateAccountItemProcessor{},
				cip,
			), nil
		}

		c.h = op.Hash()
		c.item = fact.items[i]

		if err := c.PreProcess(ctx, op, getStateFunc); err != nil {
			return nil, base.NewBaseOperationProcessReasonError(
				"fail to preprocess CreateAccountItem; %w",
				err,
			), nil
		}

		ns[i] = c
	}
	opp.ns = ns

	var stateMergeValues []base.StateMergeValue // nolint:prealloc
	for i := range opp.ns {
		s, err := opp.ns[i].Process(ctx, op, getStateFunc)
		if err != nil {
			return nil, base.NewBaseOperationProcessReasonError("process CreateAccountItem; %w", err), nil
		}
		stateMergeValues = append(stateMergeValues, s...)
	}

	var required map[types.CurrencyID][]common.Big
	switch i := op.Fact().(type) {
	case extras.FeeAble:
		required = i.FeeBase()
	default:
	}

	senderBalSts, totals, err := PrepareSenderState(fact.Sender(), required, getStateFunc)
	if err != nil {
		return nil, base.NewBaseOperationProcessReasonError("process CreateAccount; %w", err), nil
	}

	for cid := range senderBalSts {
		v, ok := senderBalSts[cid].Value().(currency.BalanceStateValue)
		if !ok {
			return nil, base.NewBaseOperationProcessReasonError(
				"expected %T, not %T",
				currency.BalanceStateValue{},
				senderBalSts[cid].Value(),
			), nil
		}

		total, found := totals[cid]
		if found {
			stateMergeValues = append(
				stateMergeValues,
				common.NewBaseStateMergeValue(
					senderBalSts[cid].Key(),
					currency.NewDeductBalanceStateValue(v.Amount.WithBig(total)),
					func(height base.Height, st base.State) base.StateValueMerger {
						return currency.NewBalanceStateValueMerger(height, senderBalSts[cid].Key(), cid, st)
					}),
			)
		}
	}

	return stateMergeValues, nil, nil
}

func (opp *CreateAccountProcessor) Close() error {
	for i := range opp.ns {
		opp.ns[i].Close()
	}

	opp.ns = nil
	opp.required = nil

	createAccountProcessorPool.Put(opp)

	return nil
}

func PrepareSenderState(
	holder base.Address,
	required map[types.CurrencyID][]common.Big,
	getStateFunc base.GetStateFunc,
) (map[types.CurrencyID]base.State, map[types.CurrencyID]common.Big, error) {
	sbSts := map[types.CurrencyID]base.State{}
	totalMap := map[types.CurrencyID]common.Big{}

	for cid, rqs := range required {
		total := common.ZeroBig
		for i := range rqs {
			total = total.Add(rqs[i])
		}

		st, err := state.ExistsState(currency.BalanceStateKey(holder, cid), fmt.Sprintf("balance of account, %v", holder), getStateFunc)
		if err != nil {
			return nil, nil, err
		}

		totalMap[cid] = total
		sbSts[cid] = st
	}

	return sbSts, totalMap, nil
}
