package did_registry

import (
	"context"
	"github.com/ProtoconNet/mitum-currency/v3/common"
	"github.com/ProtoconNet/mitum-currency/v3/state"
	didstate "github.com/ProtoconNet/mitum-currency/v3/state/did-registry"
	"github.com/ProtoconNet/mitum-currency/v3/types"
	crtypes "github.com/ProtoconNet/mitum-currency/v3/types"
	"sync"

	statecurrency "github.com/ProtoconNet/mitum-currency/v3/state/currency"
	stateextension "github.com/ProtoconNet/mitum-currency/v3/state/extension"
	mitumbase "github.com/ProtoconNet/mitum2/base"
	"github.com/ProtoconNet/mitum2/util"
)

var deactivateDIDProcessorPool = sync.Pool{
	New: func() interface{} {
		return new(DeactivateDIDProcessor)
	},
}

func (DeactivateDID) Process(
	_ context.Context, _ mitumbase.GetStateFunc,
) ([]mitumbase.StateMergeValue, mitumbase.OperationProcessReasonError, error) {
	return nil, nil, nil
}

type DeactivateDIDProcessor struct {
	*mitumbase.BaseOperationProcessor
}

func NewDeactivateDIDProcessor() crtypes.GetNewProcessor {
	return func(
		height mitumbase.Height,
		getStateFunc mitumbase.GetStateFunc,
		newPreProcessConstraintFunc mitumbase.NewOperationProcessorProcessFunc,
		newProcessConstraintFunc mitumbase.NewOperationProcessorProcessFunc,
	) (mitumbase.OperationProcessor, error) {
		e := util.StringError("failed to create new DeactivateDIDProcessor")

		nOpp := deactivateDIDProcessorPool.Get()
		opp, ok := nOpp.(*DeactivateDIDProcessor)
		if !ok {
			return nil, e.Errorf("expected %T, not %T", DeactivateDIDProcessor{}, nOpp)
		}

		b, err := mitumbase.NewBaseOperationProcessor(
			height, getStateFunc, newPreProcessConstraintFunc, newProcessConstraintFunc)
		if err != nil {
			return nil, e.Wrap(err)
		}

		opp.BaseOperationProcessor = b

		return opp, nil
	}
}

func (opp *DeactivateDIDProcessor) PreProcess(
	ctx context.Context, op mitumbase.Operation, getStateFunc mitumbase.GetStateFunc,
) (context.Context, mitumbase.OperationProcessReasonError, error) {
	fact, ok := op.Fact().(DeactivateDIDFact)
	if !ok {
		return ctx, mitumbase.NewBaseOperationProcessReasonError(
			common.ErrMPreProcess.
				Wrap(common.ErrMTypeMismatch).
				Errorf("expected %T, not %T", DeactivateDIDFact{}, op.Fact())), nil
	}

	if err := fact.IsValid(nil); err != nil {
		return ctx, mitumbase.NewBaseOperationProcessReasonError(
			common.ErrMPreProcess.
				Errorf("%v", err)), nil
	}

	if err := state.CheckExistsState(statecurrency.DesignStateKey(fact.Currency()), getStateFunc); err != nil {
		return ctx, mitumbase.NewBaseOperationProcessReasonError(
			common.ErrMPreProcess.Wrap(common.ErrMCurrencyNF).Errorf("currency id %v", fact.Currency())), nil
	}

	if _, _, aErr, cErr := state.ExistsCAccount(fact.Sender(), "sender", true, false, getStateFunc); aErr != nil {
		return ctx, mitumbase.NewBaseOperationProcessReasonError(
			common.ErrMPreProcess.
				Errorf("%v", aErr)), nil
	} else if cErr != nil {
		return ctx, mitumbase.NewBaseOperationProcessReasonError(
			common.ErrMPreProcess.Wrap(common.ErrMCAccountNA).
				Errorf("%v", cErr)), nil
	}

	//if err := state.CheckFactSignsByState(fact.Sender(), op.Signs(), getStateFunc); err != nil {
	//	return ctx, mitumbase.NewBaseOperationProcessReasonError(
	//		common.ErrMPreProcess.
	//			Wrap(common.ErrMSignInvalid).
	//			Errorf("%v", err)), nil
	//}

	_, cSt, aErr, cErr := state.ExistsCAccount(fact.Contract(), "contract", true, true, getStateFunc)
	if aErr != nil {
		return ctx, mitumbase.NewBaseOperationProcessReasonError(
			common.ErrMPreProcess.
				Errorf("%v", aErr)), nil
	} else if cErr != nil {
		return ctx, mitumbase.NewBaseOperationProcessReasonError(
			common.ErrMPreProcess.
				Errorf("%v", cErr)), nil
	}

	_, err := stateextension.CheckCAAuthFromState(cSt, fact.Sender())
	if err != nil {
		return ctx, mitumbase.NewBaseOperationProcessReasonError(
			common.ErrMPreProcess.
				Errorf("%v", err)), nil
	}

	if err := state.CheckExistsState(didstate.DesignStateKey(fact.Contract()), getStateFunc); err != nil {
		return nil, mitumbase.NewBaseOperationProcessReasonError(
			common.ErrMPreProcess.
				Wrap(common.ErrMServiceNF).Errorf("DID service for contract account %v",
				fact.Contract(),
			)), nil
	}

	_, id, err := types.ParseDIDScheme(fact.DID())
	if err != nil {
		return nil, mitumbase.NewBaseOperationProcessReasonError(
			common.ErrMPreProcess.
				Wrap(common.ErrMValueInvalid).Errorf("did scheme is invalid %v",
				fact.DID(),
			)), nil
	}

	if st, err := state.ExistsState(didstate.DataStateKey(fact.Contract(), id), "did data", getStateFunc); err != nil {
		return nil, mitumbase.NewBaseOperationProcessReasonError(
			common.ErrMPreProcess.
				Wrap(common.ErrMStateNF).Errorf("DID Data for DID %v in contract account %v", fact.DID(),
				fact.Contract(),
			)), nil
	} else if d, err := didstate.GetDataFromState(st); err != nil {
		return nil, mitumbase.NewBaseOperationProcessReasonError(
			common.ErrMPreProcess.
				Wrap(common.ErrMStateValInvalid).Errorf(
				"DID Data for DID %v in contract account %v", fact.DID(),
				fact.Contract(),
			)), nil
	} else if !d.Address().Equal(fact.Sender()) {
		return nil, mitumbase.NewBaseOperationProcessReasonError(
			common.ErrMPreProcess.
				Wrap(common.ErrMStateValInvalid).Errorf(
				"sender %v not matched with DID account address for DID %v in contract account %v", fact.Sender(), fact.DID(), fact.Contract(),
			)), nil
	}

	if st, err := state.ExistsState(didstate.DocumentStateKey(fact.Contract(), fact.DID()), "did document", getStateFunc); err != nil {
		return nil, mitumbase.NewBaseOperationProcessReasonError(
			common.ErrMPreProcess.
				Wrap(common.ErrMStateNF).Errorf("DID document for DID %v in contract account %v", fact.DID(),
				fact.Contract(),
			)), nil
	} else if d, err := didstate.GetDocumentFromState(st); err != nil {
		return nil, mitumbase.NewBaseOperationProcessReasonError(
			common.ErrMPreProcess.
				Wrap(common.ErrMStateValInvalid).Errorf(
				"DID document for DID %v in contract account %v", fact.DID(),
				fact.Contract(),
			)), nil
	} else if d.Status() != "1" {
		return nil, mitumbase.NewBaseOperationProcessReasonError(
			common.ErrMPreProcess.
				Wrap(common.ErrMValueInvalid).Errorf(
				"DID document for DID %v in contract account %v is not in active status",
				fact.DID(), fact.Contract(),
			)), nil
	}

	return ctx, nil, nil
}

func (opp *DeactivateDIDProcessor) Process( // nolint:dupl
	_ context.Context, op mitumbase.Operation, getStateFunc mitumbase.GetStateFunc) (
	[]mitumbase.StateMergeValue, mitumbase.OperationProcessReasonError, error,
) {
	e := util.StringError("failed to process DeleteData")

	fact, ok := op.Fact().(DeactivateDIDFact)
	if !ok {
		return nil, nil, e.Errorf("expected DeleteDataFact, not %T", op.Fact())
	}

	st, _ := state.ExistsState(didstate.DocumentStateKey(fact.Contract(), fact.DID()), "did document", getStateFunc)
	d, _ := didstate.GetDocumentFromState(st)
	d.SetStatus("0")

	if err := d.IsValid(nil); err != nil {
		return nil, mitumbase.NewBaseOperationProcessReasonError("invalid DID document; %w", err), nil
	}

	var sts []mitumbase.StateMergeValue // nolint:prealloc
	sts = append(sts, state.NewStateMergeValue(
		didstate.DocumentStateKey(fact.Contract(), fact.DID()),
		didstate.NewDocumentStateValue(d),
	))

	return sts, nil, nil
}

func (opp *DeactivateDIDProcessor) Close() error {
	deactivateDIDProcessorPool.Put(opp)

	return nil
}
