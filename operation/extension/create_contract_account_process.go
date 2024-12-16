package extension

import (
	"context"
	"github.com/ProtoconNet/mitum-currency/v3/operation/extras"
	"sync"

	"github.com/ProtoconNet/mitum-currency/v3/common"
	"github.com/ProtoconNet/mitum-currency/v3/operation/currency"
	"github.com/ProtoconNet/mitum-currency/v3/state"
	currencystate "github.com/ProtoconNet/mitum-currency/v3/state/currency"
	"github.com/ProtoconNet/mitum-currency/v3/state/extension"
	"github.com/ProtoconNet/mitum-currency/v3/types"
	"github.com/ProtoconNet/mitum2/base"
	"github.com/ProtoconNet/mitum2/util"
	"github.com/pkg/errors"
)

var createContractAccountItemProcessorPool = sync.Pool{
	New: func() interface{} {
		return new(CreateContractAccountItemProcessor)
	},
}

var createContractAccountProcessorPool = sync.Pool{
	New: func() interface{} {
		return new(CreateContractAccountProcessor)
	},
}

func (CreateContractAccount) Process(
	_ context.Context, _ base.GetStateFunc,
) ([]base.StateMergeValue, base.OperationProcessReasonError, error) {
	// NOTE Process is nil func
	return nil, nil, nil
}

type CreateContractAccountItemProcessor struct {
	h      util.Hash
	sender base.Address
	item   CreateContractAccountItem
	ns     base.StateMergeValue
	oas    base.StateMergeValue
	nb     map[types.CurrencyID]base.StateMergeValue
}

func (opp *CreateContractAccountItemProcessor) PreProcess(
	_ context.Context, _ base.Operation, getStateFunc base.GetStateFunc,
) error {
	e := util.StringError("preprocess CreateContractAccountItemProcessor")

	target, err := opp.item.Address()
	if err != nil {
		return e.Wrap(err)
	}

	ast, cst, aErr, cErr := state.ExistsCAccount(target, "target", false, false, getStateFunc)
	if aErr != nil {
		return e.Wrap(aErr)
	} else if cErr != nil {
		return e.Wrap(cErr)
	}

	opp.ns = state.NewStateMergeValue(ast.Key(), ast.Value())
	opp.oas = state.NewStateMergeValue(cst.Key(), cst.Value())

	nb := map[types.CurrencyID]base.StateMergeValue{}
	amounts := opp.item.Amounts()
	for i := range amounts {
		am := amounts[i]
		cid := am.Currency()
		policy, err := state.ExistsCurrencyPolicy(cid, getStateFunc)
		if err != nil {
			return e.Wrap(err)
		}
		if am.Big().Compare(policy.MinBalance()) < 0 {
			return e.Wrap(common.ErrValOOR.Wrap(errors.Errorf("amount under new account minimum balance, %v < %v", am.Big(), policy.MinBalance())))

		}

		switch _, found, err := getStateFunc(currencystate.BalanceStateKey(target, cid)); {
		case err != nil:
			return e.Wrap(err)
		case found:
			return e.Wrap(common.ErrAccountE.Wrap(errors.Errorf("target account balance already exists, %v", target)))

		default:
			nb[am.Currency()] = common.NewBaseStateMergeValue(
				currencystate.BalanceStateKey(target, cid),
				currencystate.NewAddBalanceStateValue(types.NewZeroAmount(cid)),
				func(height base.Height, st base.State) base.StateValueMerger {
					return currencystate.NewBalanceStateValueMerger(
						height,
						currencystate.BalanceStateKey(target, cid), cid, st)
				},
			)
		}
	}
	opp.nb = nb

	return nil
}

func (opp *CreateContractAccountItemProcessor) Process(
	_ context.Context, _ base.Operation, _ base.GetStateFunc,
) ([]base.StateMergeValue, error) {
	e := util.StringError("process for CreateContractAccountItemProcessor")

	sts := make([]base.StateMergeValue, len(opp.item.Amounts())+2)

	nac, err := types.NewAccountFromKeys(opp.item.Keys())

	if err != nil {
		return nil, e.Wrap(err)
	}

	ks, err := types.NewContractAccountKeys()
	if err != nil {
		return nil, e.Wrap(err)
	}

	ncac, err := nac.SetKeys(ks)
	if err != nil {
		return nil, e.Wrap(err)
	}
	sts[0] = state.NewStateMergeValue(opp.ns.Key(), currencystate.NewAccountStateValue(ncac))

	cas := types.NewContractAccountStatus(opp.sender, nil)
	sts[1] = state.NewStateMergeValue(opp.oas.Key(), extension.NewContractAccountStateValue(cas))

	amounts := opp.item.Amounts()
	for i := range amounts {
		am := amounts[i]
		cid := am.Currency()
		stv := opp.nb[cid]
		v, ok := stv.Value().(currencystate.AddBalanceStateValue)
		if !ok {
			return nil, errors.Errorf("expected AddBalanceStateValue, not %T", stv.Value())
		}
		sts[i+2] = common.NewBaseStateMergeValue(
			stv.Key(),
			currencystate.NewAddBalanceStateValue(v.Amount.WithBig(am.Big())),
			func(height base.Height, st base.State) base.StateValueMerger {
				return currencystate.NewBalanceStateValueMerger(height, stv.Key(), cid, st)
			},
		)
	}

	return sts, nil
}

func (opp *CreateContractAccountItemProcessor) Close() {
	opp.h = nil
	opp.item = nil
	opp.ns = nil
	opp.nb = nil
	opp.sender = nil
	opp.oas = nil

	createContractAccountItemProcessorPool.Put(opp)
}

type CreateContractAccountProcessor struct {
	*base.BaseOperationProcessor
	required map[types.CurrencyID][2]common.Big // required[0] : amount + fee, required[1] : fee
}

func NewCreateContractAccountProcessor() types.GetNewProcessor {
	return func(
		height base.Height,
		getStateFunc base.GetStateFunc,
		newPreProcessConstraintFunc base.NewOperationProcessorProcessFunc,
		newProcessConstraintFunc base.NewOperationProcessorProcessFunc,
	) (base.OperationProcessor, error) {
		e := util.StringError("create new CreateContractAccountProcessor")

		nopp := createContractAccountProcessorPool.Get()
		opp, ok := nopp.(*CreateContractAccountProcessor)
		if !ok {
			return nil, e.Errorf("expected CreateContractAccountProcessor, not %T", nopp)
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

func (opp *CreateContractAccountProcessor) PreProcess(
	ctx context.Context, op base.Operation, getStateFunc base.GetStateFunc,
) (context.Context, base.OperationProcessReasonError, error) {
	fact, ok := op.Fact().(CreateContractAccountFact)
	if !ok {
		return ctx, base.NewBaseOperationProcessReasonError(
			common.ErrMPreProcess.
				Wrap(common.ErrMTypeMismatch).
				Errorf("expected CreateContractAccountFact, not %T", op.Fact())), nil
	}

	items := fact.Items()
	var wg sync.WaitGroup
	errChan := make(chan *base.BaseOperationProcessReasonError, len(items))

	for i := range items {
		wg.Add(1)
		go func(item CreateContractAccountItem) {
			defer wg.Done()
			cip := createContractAccountItemProcessorPool.Get()
			c, ok := cip.(*CreateContractAccountItemProcessor)
			if !ok {
				err := base.NewBaseOperationProcessReasonError(common.ErrMPreProcess.
					Wrap(common.ErrMTypeMismatch).
					Errorf("expected CreateContractAccountItemProcessor, not %T", cip))
				errChan <- &err
				return
			}

			c.h = op.Hash()
			c.item = item
			c.sender = fact.Sender()

			if err := c.PreProcess(ctx, op, getStateFunc); err != nil {
				err := base.NewBaseOperationProcessReasonError(common.ErrMPreProcess.
					Errorf("%v", err))
				errChan <- &err
				return
			}

			c.Close()
		}(items[i])
	}
	go func() {
		wg.Wait()
		close(errChan)
	}()

	for err := range errChan {
		if err != nil {
			return nil, *err, nil
		}
	}

	return ctx, nil, nil
}

func (opp *CreateContractAccountProcessor) Process( // nolint:dupl
	ctx context.Context, op base.Operation, getStateFunc base.GetStateFunc) (
	[]base.StateMergeValue, base.OperationProcessReasonError, error,
) {
	fact, ok := op.Fact().(CreateContractAccountFact)
	if !ok {
		return nil, nil, base.NewBaseOperationProcessReasonError("expected CreateContractAccountFact, not %T", op.Fact())
	}

	//ns := make([]*CreateContractAccountItemProcessor, len(fact.items))
	var stateMergeValues []base.StateMergeValue // nolint:prealloc
	var wg sync.WaitGroup
	var mu sync.Mutex
	errChan := make(chan *base.BaseOperationProcessReasonError, len(fact.items))
	for i := range fact.items {
		wg.Add(1)
		go func(item CreateContractAccountItem) {
			defer wg.Done()
			cip := createContractAccountItemProcessorPool.Get()
			c, ok := cip.(*CreateContractAccountItemProcessor)
			if !ok {
				err := base.NewBaseOperationProcessReasonError("expected CreateContractAccountItemProcessor, not %T", cip)
				errChan <- &err
				return
			}

			c.h = op.Hash()
			c.item = item
			c.sender = fact.Sender()

			if err := c.PreProcess(ctx, op, getStateFunc); err != nil {
				err := base.NewBaseOperationProcessReasonError("fail to preprocess CreateContractAccountItem: %v", err)
				errChan <- &err
				return
			}

			s, err := c.Process(ctx, op, getStateFunc)
			if err != nil {
				err := base.NewBaseOperationProcessReasonError("process CreateContractAccountItem: %v", err)
				errChan <- &err
				return
			}
			mu.Lock()
			stateMergeValues = append(stateMergeValues, s...)
			mu.Unlock()

			c.Close()
		}(fact.items[i])
	}
	go func() {
		wg.Wait()
		close(errChan)
	}()

	for err := range errChan {
		if err != nil {
			return nil, *err, nil
		}
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
		v, ok := senderBalSts[cid].Value().(currencystate.BalanceStateValue)
		if !ok {
			return nil, base.NewBaseOperationProcessReasonError(
				"expected %T, not %T",
				currencystate.BalanceStateValue{},
				senderBalSts[cid].Value(),
			), nil
		}

		total, found := totals[cid]
		if found {
			stateMergeValues = append(
				stateMergeValues,
				common.NewBaseStateMergeValue(
					senderBalSts[cid].Key(),
					currencystate.NewDeductBalanceStateValue(v.Amount.WithBig(total)),
					func(height base.Height, st base.State) base.StateValueMerger {
						return currencystate.NewBalanceStateValueMerger(height, senderBalSts[cid].Key(), cid, st)
					}),
			)
		}
	}

	return stateMergeValues, nil, nil
}

func (opp *CreateContractAccountProcessor) Close() error {
	opp.required = nil

	createContractAccountProcessorPool.Put(opp)

	return nil
}
