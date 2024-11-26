package extension

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

var UpdateRecipientProcessorPool = sync.Pool{
	New: func() interface{} {
		return new(UpdateRecipientProcessor)
	},
}

func (UpdateRecipient) Process(
	_ context.Context, _ base.GetStateFunc,
) ([]base.StateMergeValue, base.OperationProcessReasonError, error) {
	// NOTE Process is nil func
	return nil, nil, nil
}

type UpdateRecipientProcessor struct {
	*base.BaseOperationProcessor
	ca  base.StateMergeValue
	sb  base.StateMergeValue
	fee common.Big
}

func NewUpdateRecipientProcessor() types.GetNewProcessor {
	return func(
		height base.Height,
		getStateFunc base.GetStateFunc,
		newPreProcessConstraintFunc base.NewOperationProcessorProcessFunc,
		newProcessConstraintFunc base.NewOperationProcessorProcessFunc,
	) (base.OperationProcessor, error) {
		e := util.StringError("create new UpdateRecipientProcessor")

		nopp := UpdateRecipientProcessorPool.Get()
		opp, ok := nopp.(*UpdateRecipientProcessor)
		if !ok {
			return nil, errors.Errorf("expected UpdateRecipientProcessor, not %T", nopp)
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

func (opp *UpdateRecipientProcessor) PreProcess(
	ctx context.Context, op base.Operation, getStateFunc base.GetStateFunc,
) (context.Context, base.OperationProcessReasonError, error) {
	fact, ok := op.Fact().(UpdateRecipientFact)
	if !ok {
		return ctx, base.NewBaseOperationProcessReasonError(
			common.ErrMPreProcess.
				Wrap(common.ErrMTypeMismatch).
				Errorf("expected UpdateRecipientFact, not %T", op.Fact())), nil
	}

	_, err := state.ExistsCurrencyPolicy(fact.Currency(), getStateFunc)
	if err != nil {
		return ctx, base.NewBaseOperationProcessReasonError(
			common.ErrMPreProcess.
				Errorf("%v", err),
		), nil
	}

	if _, _, aErr, cErr := state.ExistsCAccount(fact.Sender(), "sender", true, false, getStateFunc); aErr != nil {
		return ctx, base.NewBaseOperationProcessReasonError(
			common.ErrMPreProcess.
				Errorf("%v", aErr)), nil
	} else if cErr != nil {
		return ctx, base.NewBaseOperationProcessReasonError(
			common.ErrMPreProcess.
				Wrap(common.ErrMCAccountNA).
				Errorf("%v: sender %v is contract account", cErr, fact.Sender())), nil
	}

	if _, cSt, aErr, cErr := state.ExistsCAccount(fact.Contract(), "contract", true, true, getStateFunc); aErr != nil {
		return ctx, base.NewBaseOperationProcessReasonError(
			common.ErrMPreProcess.
				Errorf("%v", aErr)), nil
	} else if cErr != nil {
		return ctx, base.NewBaseOperationProcessReasonError(
			common.ErrMPreProcess.
				Errorf("%v", cErr)), nil
	} else if status, err := extension.StateContractAccountValue(cSt); err != nil {
		return ctx, base.NewBaseOperationProcessReasonError(
			common.ErrMPreProcess.
				Wrap(common.ErrMStateValInvalid).
				Errorf("%v", cErr)), nil
	} else if !status.Owner().Equal(fact.Sender()) {
		return ctx, base.NewBaseOperationProcessReasonError(
			common.ErrMPreProcess.
				Wrap(common.ErrMAccountNAth).
				Errorf("sender %v is not owner of contract account", fact.Sender())), nil
	}

	for i := range fact.Recipients() {
		if _, _, _, cErr := state.ExistsCAccount(
			fact.Recipients()[i], "recipient", true, false, getStateFunc); cErr != nil {
			return ctx, base.NewBaseOperationProcessReasonError(
				common.ErrMPreProcess.
					Wrap(common.ErrMCAccountNA).
					Errorf("%v: recipient %v is contract account", cErr, fact.Recipients()[i])), nil
		}
	}

	//if err := state.CheckFactSignsByState(fact.Sender(), op.Signs(), getStateFunc); err != nil {
	//	return ctx, base.NewBaseOperationProcessReasonError(
	//		common.ErrMPreProcess.
	//			Wrap(common.ErrMSignInvalid).
	//			Errorf("%v", err)), nil
	//}

	return ctx, nil, nil
}

func (opp *UpdateRecipientProcessor) Process( // nolint:dupl
	_ context.Context, op base.Operation, getStateFunc base.GetStateFunc) (
	[]base.StateMergeValue, base.OperationProcessReasonError, error,
) {
	e := util.StringError("process UpdateRecipient")

	fact, ok := op.Fact().(UpdateRecipientFact)
	if !ok {
		return nil, nil, e.Errorf("expected UpdateRecipientFact, not %T", op.Fact())
	}

	var ctAccSt base.State
	var err error
	ctAccSt, err = state.ExistsState(extension.StateKeyContractAccount(fact.Contract()), "contract account status", getStateFunc)
	if err != nil {
		return nil, base.NewBaseOperationProcessReasonError("check existence of contract account status %v ; %w", fact.Contract(), err), nil
	}

	var fee common.Big
	policy, err := state.ExistsCurrencyPolicy(fact.Currency(), getStateFunc)
	if err != nil {
		return nil, base.NewBaseOperationProcessReasonError("check existence of currency id %q: %w", fact.Currency(), err), nil
	} else if fee, err = policy.Feeer().Fee(common.ZeroBig); err != nil {
		return nil, base.NewBaseOperationProcessReasonError("check fee of currency id %q: %w", fact.Currency(), err), nil
	}

	var stmvs []base.StateMergeValue // nolint:prealloc

	for _, recipient := range fact.Recipients() {
		smv, err := state.CreateNotExistAccount(recipient, getStateFunc)
		if err != nil {
			return nil, base.NewBaseOperationProcessReasonError("%w", err), nil
		} else if smv != nil {
			stmvs = append(stmvs, smv)
		}
	}

	var sdBalSt base.State
	if sdBalSt, err = state.ExistsState(currency.BalanceStateKey(fact.Sender(), fact.Currency()), "balance of sender", getStateFunc); err != nil {
		return nil, base.NewBaseOperationProcessReasonError("check existence of sender balance %v ; %w", fact.Sender(), err), nil
	} else if b, err := currency.StateBalanceValue(sdBalSt); err != nil {
		return nil, base.NewBaseOperationProcessReasonError("check existence of sender balance %v, %v ; %w", fact.Currency(), fact.Sender(), err), nil
	} else if b.Big().Compare(fee) < 0 {
		return nil, base.NewBaseOperationProcessReasonError("insufficient balance with fee %v ,%v", fact.Currency(), fact.Sender()), nil
	}

	v, ok := sdBalSt.Value().(currency.BalanceStateValue)
	if !ok {
		return nil, base.NewBaseOperationProcessReasonError("expected BalanceStateValue, not %T", sdBalSt.Value()), nil
	}

	if policy.Feeer().Receiver() != nil {
		if err := state.CheckExistsState(currency.AccountStateKey(policy.Feeer().Receiver()), getStateFunc); err != nil {
			return nil, nil, errors.Errorf("feeer receiver %s not found", policy.Feeer().Receiver())
		} else if feeRcvrSt, found, err := getStateFunc(currency.BalanceStateKey(policy.Feeer().Receiver(), fact.Currency())); err != nil {
			return nil, nil, errors.Errorf("feeer receiver %s balance of %s not found", policy.Feeer().Receiver(), fact.Currency())
		} else if !found {
			return nil, nil, errors.Errorf("feeer receiver %s balance of %s not found", policy.Feeer().Receiver(), fact.Currency())
		} else if feeRcvrSt.Key() != sdBalSt.Key() {
			r, ok := feeRcvrSt.Value().(currency.BalanceStateValue)
			if !ok {
				return nil, nil, errors.Errorf("invalid BalanceState value found, %T", feeRcvrSt.Value())
			}
			stmvs = append(stmvs, common.NewBaseStateMergeValue(
				feeRcvrSt.Key(),
				currency.NewAddBalanceStateValue(r.Amount.WithBig(fee)),
				func(height base.Height, st base.State) base.StateValueMerger {
					return currency.NewBalanceStateValueMerger(height, feeRcvrSt.Key(), fact.Currency(), st)
				},
			))

			stmvs = append(stmvs, common.NewBaseStateMergeValue(
				sdBalSt.Key(),
				currency.NewDeductBalanceStateValue(v.Amount.WithBig(fee)),
				func(height base.Height, st base.State) base.StateValueMerger {
					return currency.NewBalanceStateValueMerger(height, sdBalSt.Key(), fact.Currency(), st)
				},
			))
		}
	}

	ctsv := ctAccSt.Value()
	if ctsv == nil {
		return nil, nil, util.ErrNotFound.Errorf("contract account status not found in State")
	}

	sv, ok := ctsv.(extension.ContractAccountStateValue)
	if !ok {
		return nil, nil, errors.Errorf("invalid contract account value found, %T", ctsv)
	}

	status := sv.Status()
	err = status.SetRecipients(fact.Recipients())
	if err != nil {
		return nil, nil, err
	}

	stmvs = append(stmvs, state.NewStateMergeValue(ctAccSt.Key(), extension.NewContractAccountStateValue(status)))

	return stmvs, nil, nil
}

func (opp *UpdateRecipientProcessor) Close() error {
	UpdateRecipientProcessorPool.Put(opp)

	return nil
}
