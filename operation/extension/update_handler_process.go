package extension

import (
	"context"
	"sync"

	"github.com/ProtoconNet/mitum-currency/v3/common"
	"github.com/ProtoconNet/mitum-currency/v3/state"
	"github.com/ProtoconNet/mitum-currency/v3/state/extension"
	"github.com/ProtoconNet/mitum-currency/v3/types"
	"github.com/ProtoconNet/mitum2/base"

	"github.com/ProtoconNet/mitum2/util"
	"github.com/pkg/errors"
)

var UpdateHandlerProcessorPool = sync.Pool{
	New: func() interface{} {
		return new(UpdateHandlerProcessor)
	},
}

func (UpdateHandler) Process(
	_ context.Context, _ base.GetStateFunc,
) ([]base.StateMergeValue, base.OperationProcessReasonError, error) {
	// NOTE Process is nil func
	return nil, nil, nil
}

type UpdateHandlerProcessor struct {
	*base.BaseOperationProcessor
	ca  base.StateMergeValue
	sb  base.StateMergeValue
	fee common.Big
}

func NewUpdateHandlerProcessor() types.GetNewProcessor {
	return func(
		height base.Height,
		getStateFunc base.GetStateFunc,
		newPreProcessConstraintFunc base.NewOperationProcessorProcessFunc,
		newProcessConstraintFunc base.NewOperationProcessorProcessFunc,
	) (base.OperationProcessor, error) {
		e := util.StringError("create new UpdateHandlerProcessor")

		nopp := UpdateHandlerProcessorPool.Get()
		opp, ok := nopp.(*UpdateHandlerProcessor)
		if !ok {
			return nil, errors.Errorf("expected UpdateHandlerProcessor, not %T", nopp)
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

func (opp *UpdateHandlerProcessor) PreProcess(
	ctx context.Context, op base.Operation, getStateFunc base.GetStateFunc,
) (context.Context, base.OperationProcessReasonError, error) {
	fact, ok := op.Fact().(UpdateHandlerFact)
	if !ok {
		return ctx, base.NewBaseOperationProcessReasonError(
			common.ErrMPreProcess.
				Wrap(common.ErrMTypeMismatch).
				Errorf("expected UpdateHandlerFact, not %T", op.Fact())), nil
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

	for i := range fact.Handlers() {
		if _, _, _, cErr := state.ExistsCAccount(
			fact.Handlers()[i], "handler", true, false, getStateFunc); cErr != nil {
			return ctx, base.NewBaseOperationProcessReasonError(
				common.ErrMPreProcess.
					Wrap(common.ErrMCAccountNA).
					Errorf("%v: handler %v is contract account", cErr, fact.Handlers()[i])), nil
		}
	}

	return ctx, nil, nil
}

func (opp *UpdateHandlerProcessor) Process( // nolint:dupl
	_ context.Context, op base.Operation, getStateFunc base.GetStateFunc) (
	[]base.StateMergeValue, base.OperationProcessReasonError, error,
) {
	e := util.StringError("process UpdateHandler")

	fact, ok := op.Fact().(UpdateHandlerFact)
	if !ok {
		return nil, nil, e.Errorf("expected UpdateHandlerFact, not %T", op.Fact())
	}

	var ctAccSt base.State
	var err error
	ctAccSt, err = state.ExistsState(extension.StateKeyContractAccount(fact.Contract()), "contract account status", getStateFunc)
	if err != nil {
		return nil, base.NewBaseOperationProcessReasonError("check existence of contract account status %v ; %w", fact.Contract(), err), nil
	}

	var stmvs []base.StateMergeValue // nolint:prealloc

	for _, handler := range fact.Handlers() {
		smv, err := state.CreateNotExistAccount(handler, getStateFunc)
		if err != nil {
			return nil, base.NewBaseOperationProcessReasonError("%w", err), nil
		} else if smv != nil {
			stmvs = append(stmvs, smv)
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
	err = status.SetHandlers(fact.Handlers())
	if err != nil {
		return nil, nil, err
	}

	stmvs = append(stmvs, state.NewStateMergeValue(ctAccSt.Key(), extension.NewContractAccountStateValue(status)))

	return stmvs, nil, nil
}

func (opp *UpdateHandlerProcessor) Close() error {
	UpdateHandlerProcessorPool.Put(opp)

	return nil
}
