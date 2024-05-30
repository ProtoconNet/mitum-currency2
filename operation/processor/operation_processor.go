package processor

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/ProtoconNet/mitum-currency/v3/operation/currency"
	"github.com/ProtoconNet/mitum-currency/v3/operation/extension"
	"github.com/ProtoconNet/mitum-currency/v3/types"
	"github.com/ProtoconNet/mitum2/base"

	"github.com/ProtoconNet/mitum2/util"
	"github.com/ProtoconNet/mitum2/util/hint"
	"github.com/ProtoconNet/mitum2/util/logging"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

var operationProcessorPool = sync.Pool{
	New: func() interface{} {
		return new(OperationProcessor)
	},
}

type GetLastBlockFunc func() (base.BlockMap, bool, error)

const (
	DuplicationTypeSender   types.DuplicationType = "sender"
	DuplicationTypeCurrency types.DuplicationType = "currency"
	DuplicationTypeContract types.DuplicationType = "contract"
)

type BaseOperationProcessor interface {
	PreProcess(base.Operation, base.GetStateFunc) (base.OperationProcessReasonError, error)
	Process(base.Operation, base.GetStateFunc) ([]base.StateMergeValue, base.OperationProcessReasonError, error)
	Close() error
}

type OperationProcessor struct {
	// id string
	sync.RWMutex
	*logging.Logging
	*base.BaseOperationProcessor
	processorHintSet     *hint.CompatibleSet[types.GetNewProcessor]
	Duplicated           map[string]struct{}
	duplicatedNewAddress map[string]struct{}
	processorClosers     *sync.Map
	GetStateFunc         base.GetStateFunc
	CollectFee           func(*OperationProcessor, types.AddFee) error
	CheckDuplicationFunc func(*OperationProcessor, base.Operation) error
	GetNewProcessorFunc  func(*OperationProcessor, base.Operation) (base.OperationProcessor, bool, error)
}

func NewOperationProcessor() *OperationProcessor {
	m := sync.Map{}
	return &OperationProcessor{
		// id: util.UUID().String(),
		Logging: logging.NewLogging(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "mitum-currency-operations-processor")
		}),
		processorHintSet:     hint.NewCompatibleSet[types.GetNewProcessor](1 << 9),
		Duplicated:           map[string]struct{}{},
		duplicatedNewAddress: map[string]struct{}{},
		processorClosers:     &m,
	}
}

func (opr *OperationProcessor) New(
	height base.Height,
	getStateFunc base.GetStateFunc,
	newPreProcessConstraintFunc base.NewOperationProcessorProcessFunc,
	newProcessConstraintFunc base.NewOperationProcessorProcessFunc) (*OperationProcessor, error) {
	e := util.StringError("create new OperationProcessor")

	nopr := operationProcessorPool.Get().(*OperationProcessor)
	if nopr.processorHintSet == nil {
		nopr.processorHintSet = opr.processorHintSet
	}

	if nopr.Duplicated == nil {
		nopr.Duplicated = make(map[string]struct{})
	}

	if nopr.duplicatedNewAddress == nil {
		nopr.duplicatedNewAddress = make(map[string]struct{})
	}

	if nopr.Logging == nil {
		nopr.Logging = opr.Logging
	}

	b, err := base.NewBaseOperationProcessor(
		height, getStateFunc, newPreProcessConstraintFunc, newProcessConstraintFunc)
	if err != nil {
		return nil, e.Wrap(err)
	}

	nopr.BaseOperationProcessor = b
	nopr.GetStateFunc = getStateFunc
	nopr.CheckDuplicationFunc = opr.CheckDuplicationFunc
	nopr.GetNewProcessorFunc = opr.GetNewProcessorFunc
	return nopr, nil
}

func (opr *OperationProcessor) SetProcessor(
	hint hint.Hint,
	newProcessor types.GetNewProcessor,
) error {
	if err := opr.processorHintSet.Add(hint, newProcessor); err != nil {
		if !errors.Is(err, util.ErrFound) {
			return err
		}
	}

	return nil
}

func (opr *OperationProcessor) SetCheckDuplicationFunc(
	f func(*OperationProcessor, base.Operation) error,
) error {
	if f == nil {
		return errors.Errorf("Set nil func to CheckDuplicationFunc")
	}
	opr.CheckDuplicationFunc = f

	return nil
}

func (opr *OperationProcessor) SetGetNewProcessorFunc(
	f func(*OperationProcessor, base.Operation) (base.OperationProcessor, bool, error),
) error {
	if f == nil {
		return errors.Errorf("Set nil func to GetNewProcessorFunc")
	}
	opr.GetNewProcessorFunc = f

	return nil
}

func (opr *OperationProcessor) PreProcess(ctx context.Context, op base.Operation, getStateFunc base.GetStateFunc) (context.Context, base.OperationProcessReasonError, error) {
	e := util.StringError("preprocess for OperationProcessor")

	if opr.processorClosers == nil {
		opr.processorClosers = &sync.Map{}
	}

	var sp base.OperationProcessor

	if opr.GetNewProcessorFunc == nil {
		return ctx, nil, e.Errorf("GetNewProcessorFunc is nil")
	}
	switch i, known, err := opr.GetNewProcessorFunc(opr, op); {
	case err != nil:
		return ctx, base.NewBaseOperationProcessReasonError(err.Error()), nil
	case !known:
		return ctx, nil, e.Errorf("getNewProcessor, %T", op)
	default:
		sp = i
	}

	switch _, reasonErr, err := sp.PreProcess(ctx, op, getStateFunc); {
	case err != nil:
		return ctx, nil, e.Wrap(err)
	case reasonErr != nil:
		return ctx, reasonErr, nil
	}

	return ctx, nil, nil
}

func (opr *OperationProcessor) Process(ctx context.Context, op base.Operation, getStateFunc base.GetStateFunc) ([]base.StateMergeValue, base.OperationProcessReasonError, error) {
	e := util.StringError("process for OperationProcessor")

	if err := opr.CheckDuplicationFunc(opr, op); err != nil {
		return nil, base.NewBaseOperationProcessReasonError("duplication found; %w", err), nil
	}

	var sp base.OperationProcessor
	if opr.GetNewProcessorFunc == nil {
		return nil, nil, e.Errorf("GetNewProcessorFunc is nil")
	}

	switch i, known, err := opr.GetNewProcessorFunc(opr, op); {
	case err != nil:
		return nil, nil, e.Wrap(err)
	case !known:
		return nil, nil, e.Errorf("getNewProcessor")
	default:
		sp = i
	}

	stateMergeValues, reasonErr, err := sp.Process(ctx, op, getStateFunc)
	return stateMergeValues, reasonErr, err
}

func DuplicationKey(key string, duplType types.DuplicationType) string {
	return fmt.Sprintf("%s:%s", key, duplType)
}

func CheckDuplication(opr *OperationProcessor, op base.Operation) error {
	opr.Lock()
	defer opr.Unlock()

	var duplicationTypeSenderID string
	var duplicationTypeCurrencyID string
	var duplicationTypeContractID string
	var newAddresses []base.Address

	switch t := op.(type) {
	case currency.CreateAccount:
		fact, ok := t.Fact().(currency.CreateAccountFact)
		if !ok {
			return errors.Errorf("expected CreateAccountFact, not %T", t.Fact())
		}
		as, err := fact.Targets()
		if err != nil {
			return errors.Errorf("failed to get Addresses")
		}
		newAddresses = as
		duplicationTypeSenderID = DuplicationKey(fact.Sender().String(), DuplicationTypeSender)
	case currency.UpdateKey:
		fact, ok := t.Fact().(currency.UpdateKeyFact)
		if !ok {
			return errors.Errorf("expected UpdateKeyFact, not %T", t.Fact())
		}
		duplicationTypeSenderID = DuplicationKey(fact.Target().String(), DuplicationTypeSender)
	case currency.Transfer:
		fact, ok := t.Fact().(currency.TransferFact)
		if !ok {
			return errors.Errorf("expected TransferFact, not %T", t.Fact())
		}
		duplicationTypeSenderID = DuplicationKey(fact.Sender().String(), DuplicationTypeSender)
	case currency.RegisterCurrency:
		fact, ok := t.Fact().(currency.RegisterCurrencyFact)
		if !ok {
			return errors.Errorf("expected RegisterCurrencyFact, not %T", t.Fact())
		}
		duplicationTypeCurrencyID = DuplicationKey(fact.Currency().Currency().String(), DuplicationTypeCurrency)
	case currency.UpdateCurrency:
		fact, ok := t.Fact().(currency.UpdateCurrencyFact)
		if !ok {
			return errors.Errorf("expected UpdateCurrencyFact, not %T", t.Fact())
		}
		duplicationTypeCurrencyID = DuplicationKey(fact.Currency().String(), DuplicationTypeCurrency)
	case currency.Mint:
	case extension.CreateContractAccount:
		fact, ok := t.Fact().(extension.CreateContractAccountFact)
		if !ok {
			return errors.Errorf("expected CreateContractAccountFact, not %T", t.Fact())
		}
		as, err := fact.Targets()
		if err != nil {
			return errors.Errorf("failed to get Addresses")
		}
		newAddresses = as
		duplicationTypeSenderID = DuplicationKey(fact.Sender().String(), DuplicationTypeSender)
		duplicationTypeContractID = DuplicationKey(fact.Sender().String(), DuplicationTypeContract)
	case extension.Withdraw:
		fact, ok := t.Fact().(extension.WithdrawFact)
		if !ok {
			return errors.Errorf("expected WithdrawFact, not %T", t.Fact())
		}
		duplicationTypeSenderID = DuplicationKey(fact.Sender().String(), DuplicationTypeSender)
	default:
		return nil
	}

	if len(duplicationTypeSenderID) > 0 {
		fmt.Println(opr.Duplicated)
		if _, found := opr.Duplicated[duplicationTypeSenderID]; found {
			return errors.Errorf("proposal cannot have duplicated sender, %v", duplicationTypeSenderID)
		}

		opr.Duplicated[duplicationTypeSenderID] = struct{}{}
	}

	if len(duplicationTypeCurrencyID) > 0 {
		if _, found := opr.Duplicated[duplicationTypeCurrencyID]; found {
			return errors.Errorf(
				"cannot register duplicated currency id, %v within a proposal",
				duplicationTypeCurrencyID,
			)
		}

		opr.Duplicated[duplicationTypeCurrencyID] = struct{}{}
	}
	if len(duplicationTypeContractID) > 0 {
		if _, found := opr.Duplicated[duplicationTypeContractID]; found {
			return errors.Errorf(
				"cannot use a duplicated contract, %v within a proposal",
				duplicationTypeContractID,
			)
		}

		opr.Duplicated[duplicationTypeContractID] = struct{}{}
	}

	if len(newAddresses) > 0 {
		if err := opr.CheckNewAddressDuplication(newAddresses); err != nil {
			return err
		}
	}

	return nil
}

func (opr *OperationProcessor) CheckNewAddressDuplication(as []base.Address) error {
	for i := range as {
		if _, found := opr.duplicatedNewAddress[as[i].String()]; found {
			return errors.Errorf("new address already processed")
		}
	}

	for i := range as {
		opr.duplicatedNewAddress[as[i].String()] = struct{}{}
	}

	return nil
}

func (opr *OperationProcessor) Close() error {
	opr.Lock()
	defer opr.Unlock()

	defer opr.close()

	return nil
}

func (opr *OperationProcessor) Cancel() error {
	opr.Lock()
	defer opr.Unlock()

	defer opr.close()

	return nil
}

func GetNewProcessor(opr *OperationProcessor, op base.Operation) (base.OperationProcessor, bool, error) {
	switch i, err := opr.GetNewProcessorFromHintset(op); {
	case err != nil:
		return nil, false, err
	case i != nil:
		return i, true, nil
	}

	switch t := op.(type) {
	case currency.CreateAccount,
		currency.UpdateKey,
		currency.Transfer,
		currency.RegisterCurrency,
		currency.UpdateCurrency,
		currency.Mint,
		extension.CreateContractAccount,
		extension.UpdateOperator,
		extension.Withdraw:
		return nil, false, errors.Errorf("%T needs SetProcessor", t)
	default:
		return nil, false, nil
	}
}

func (opr *OperationProcessor) GetNewProcessorFromHintset(op base.Operation) (base.OperationProcessor, error) {
	var f types.GetNewProcessor
	if hinter, ok := op.(hint.Hinter); !ok {
		return nil, nil
	} else if i, found := opr.processorHintSet.Find(hinter.Hint()); !found {
		return nil, nil
	} else {
		f = i
	}

	opp, err := f(opr.Height(), opr.GetStateFunc, nil, nil)
	if err != nil {
		return nil, err
	}

	h := op.(util.Hasher).Hash().String()
	_, isCloser := opp.(io.Closer)
	if isCloser {
		opr.processorClosers.Store(h, opp)
		isCloser = true
	}

	opr.Log().Debug().
		Str("operation", h).
		Str("processor", fmt.Sprintf("%T", opp)).
		Bool("is_closer", isCloser).
		Msg("operation processor created")

	return opp, nil
}

func (opr *OperationProcessor) close() {
	opr.processorClosers.Range(func(_, v interface{}) bool {
		err := v.(io.Closer).Close()
		if err != nil {
			opr.Log().Error().Err(err).Str("op", fmt.Sprintf("%T", v)).Msg("close operation processor")
		} else {
			opr.Log().Debug().Str("processor", fmt.Sprintf("%T", v)).Msg("operation processor closed")
		}

		return true
	})

	// opr.pool = nil
	opr.Duplicated = nil
	opr.duplicatedNewAddress = nil
	opr.processorClosers = &sync.Map{}

	operationProcessorPool.Put(opr)

	opr.Log().Debug().Msg("operation processors closed")
}
