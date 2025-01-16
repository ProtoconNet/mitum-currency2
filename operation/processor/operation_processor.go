package processor

import (
	"context"
	"fmt"
	"github.com/ProtoconNet/mitum-currency/v3/common"
	"github.com/ProtoconNet/mitum-currency/v3/state"
	didstate "github.com/ProtoconNet/mitum-currency/v3/state/did-registry"
	"github.com/btcsuite/btcutil/base58"
	"io"
	"sync"

	"github.com/ProtoconNet/mitum-currency/v3/operation/currency"
	"github.com/ProtoconNet/mitum-currency/v3/operation/did-registry"
	"github.com/ProtoconNet/mitum-currency/v3/operation/extension"
	statecurrency "github.com/ProtoconNet/mitum-currency/v3/state/currency"
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
	DuplicationTypeSender    types.DuplicationType = "sender"
	DuplicationTypeCurrency  types.DuplicationType = "currency"
	DuplicationTypeContract  types.DuplicationType = "contract"
	DuplicationTypeDID       types.DuplicationType = "did"
	DuplicationTypeDIDPubKey types.DuplicationType = "didpubkey"
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
	processorHintSet             *hint.CompatibleSet[types.GetNewProcessor]
	processorHintSetWithProposal *hint.CompatibleSet[types.GetNewProcessorWithProposal]
	Duplicated                   map[string]struct{}
	duplicatedNewAddress         map[string]struct{}
	processorClosers             *sync.Map
	proposal                     *base.ProposalSignFact
	GetStateFunc                 base.GetStateFunc
	CollectFee                   func(*OperationProcessor, types.AddFee) error
	CheckDuplicationFunc         func(*OperationProcessor, base.Operation) error
	GetNewProcessorFunc          func(*OperationProcessor, base.Operation) (base.OperationProcessor, bool, error)
}

func NewOperationProcessor() *OperationProcessor {
	m := sync.Map{}
	return &OperationProcessor{
		// id: util.UUID().String(),
		Logging: logging.NewLogging(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "mitum-currency-operations-processor")
		}),
		processorHintSet:             hint.NewCompatibleSet[types.GetNewProcessor](1 << 9),
		processorHintSetWithProposal: hint.NewCompatibleSet[types.GetNewProcessorWithProposal](1 << 9),
		Duplicated:                   map[string]struct{}{},
		duplicatedNewAddress:         map[string]struct{}{},
		processorClosers:             &m,
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

	if nopr.processorHintSetWithProposal == nil {
		nopr.processorHintSetWithProposal = opr.processorHintSetWithProposal
	}

	if nopr.Duplicated == nil {
		nopr.Duplicated = make(map[string]struct{})
	}

	if nopr.proposal == nil && opr.proposal != nil {
		nopr.proposal = opr.proposal
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

func (opr *OperationProcessor) SetProcessorWithProposal(
	hint hint.Hint,
	newProcessor types.GetNewProcessorWithProposal,
) error {
	if err := opr.processorHintSetWithProposal.Add(hint, newProcessor); err != nil {
		if !errors.Is(err, util.ErrFound) {
			return err
		}
	}

	return nil
}

func (opr *OperationProcessor) SetProposal(
	proposal *base.ProposalSignFact,
) error {
	if proposal == nil {
		return errors.Errorf("Set nil proposal to OperationProcessor")
	}
	opr.proposal = proposal

	return nil
}

func (opr *OperationProcessor) GetProposal() *base.ProposalSignFact {
	return opr.proposal
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

	if err := opr.CheckDuplicationFunc(opr, op); err != nil {
		return ctx, base.NewBaseOperationProcessReasonError("duplication found; %w", err), nil
	}

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

	switch k := op.(type) {
	case common.IExtendedOperation:
		if k.GetAuthentication() != nil && k.GetSettlement() != nil {
			err := k.GetAuthentication().IsValid(nil)
			if err != nil {
				return ctx, base.NewBaseOperationProcessReasonError(
					common.ErrMPreProcess.Errorf("%v", err)), nil
			}
			err = k.GetSettlement().IsValid(nil)
			if err != nil {
				return ctx, base.NewBaseOperationProcessReasonError(
					common.ErrMPreProcess.Errorf("%v", err)), nil
			}

			opSender, _ := k.GetSettlement().OpSender()
			if err := state.CheckFactSignsByState(opSender, op.Signs(), getStateFunc); err != nil {
				return ctx,
					base.NewBaseOperationProcessReasonError(
						common.ErrMPreProcess.
							Wrap(common.ErrMSignInvalid).
							Errorf("%v", err),
					), nil
			}

			var authentication types.IAuthentication
			var doc types.DIDDocument
			dr, err := types.NewDIDResourceFromString(k.AuthenticationID())
			if err != nil {
				return ctx, nil, err
			}

			contract, _ := k.Contract()
			if st, err := state.ExistsState(didstate.DocumentStateKey(contract, dr.DID()), "did document", getStateFunc); err != nil {
				return ctx, nil, err
			} else if doc, err = didstate.GetDocumentFromState(st); err != nil {
				return ctx, nil, err
			}

			authentication, err = doc.Authentication(k.AuthenticationID())
			if err != nil {
				return ctx, base.NewBaseOperationProcessReasonError(
					common.ErrMPreProcess.Errorf("%v", err)), nil
			}

			if authentication.Controller() != dr.DID() {
				return ctx, base.NewBaseOperationProcessReasonError(
					common.ErrMPreProcess.Errorf(
						"%v", errors.Errorf(
							"Controller of authentication id, %v is not matched with DID in document, %v", authentication.Controller(), dr.DID()))), nil
			}

			switch authentication.AuthType() {
			case types.AuthTypeECDSASECP:
				details := authentication.Details()
				pubKey, ok := details.(base.Publickey)
				if !ok {
					return ctx, base.NewBaseOperationProcessReasonError(
						common.ErrMPreProcess.Errorf("%v", errors.Errorf("expected PublicKey, but %T", details))), nil
				}

				signature := base58.Decode(k.ProofData())
				err := pubKey.Verify(op.Fact().Hash().Bytes(), signature)
				if err != nil {
					return ctx, base.NewBaseOperationProcessReasonError(
						common.ErrMPreProcess.Errorf("%v", err)), nil
				}

				opSender, ok := k.OpSender()
				if !ok {
					return ctx, base.NewBaseOperationProcessReasonError(
						common.ErrMPreProcess.Errorf("%v", errors.Errorf("empty op sender"))), nil
				}

				if _, _, aErr, cErr := state.ExistsCAccount(opSender, "op sender", true, false, getStateFunc); aErr != nil {
					return ctx, base.NewBaseOperationProcessReasonError(
						common.ErrMPreProcess.Errorf("%v", aErr)), nil
				} else if cErr != nil {
					return ctx, base.NewBaseOperationProcessReasonError(
						common.ErrMPreProcess.Errorf("%v", cErr)), nil
				}

				proxyPayer, ok := k.ProxyPayer()
				if ok {
					if _, _, aErr, cErr := state.ExistsCAccount(proxyPayer, "proxy payer", true, true, getStateFunc); aErr != nil {
						return ctx, base.NewBaseOperationProcessReasonError(
							common.ErrMPreProcess.Errorf("%v", aErr)), nil
					} else if cErr != nil {
						return ctx, base.NewBaseOperationProcessReasonError(
							common.ErrMPreProcess.Errorf("%v", cErr)), nil
					}
				}
			case types.AuthTypeVC:
				details := authentication.Details()
				m, ok := details.(map[string]interface{})
				if !ok {
					return ctx, base.NewBaseOperationProcessReasonError(
						common.ErrMPreProcess.Errorf("%v", errors.Errorf("get authentication details"))), nil
				}
				p := m["proof"]
				proof, ok := p.(types.Proof)
				if !ok {
					return ctx, base.NewBaseOperationProcessReasonError(
						common.ErrMPreProcess.Errorf("%v", errors.Errorf("get vc proof"))), nil
				}
				vm := proof.VerificationMethod()
				dr, err := types.NewDIDResourceFromString(vm)
				if err != nil {
					return ctx, nil, err
				}

				var doc types.DIDDocument
				contract, _ := k.Contract()
				if st, err := state.ExistsState(didstate.DocumentStateKey(contract, dr.DID()), "did document", getStateFunc); err != nil {
					return ctx, nil, err
				} else if doc, err = didstate.GetDocumentFromState(st); err != nil {
					return ctx, nil, err
				}

				sAuthentication, err := doc.Authentication(vm)
				if err != nil {
					return ctx, base.NewBaseOperationProcessReasonError(
						common.ErrMPreProcess.Errorf("%v", err)), nil
				}

				if sAuthentication.Controller() != dr.DID() {
					return ctx, base.NewBaseOperationProcessReasonError(
						common.ErrMPreProcess.Errorf(
							"%v", errors.Errorf(
								"Controller of authentication id, %v is not matched with DID in document, %v", authentication.Controller(), dr.DID()))), nil
				}

				if sAuthentication.AuthType() != types.AuthTypeECDSASECP {
					return ctx, base.NewBaseOperationProcessReasonError(
						common.ErrMPreProcess.Errorf("%v", errors.Errorf("auth type must be EcdsaSecp256k1VerificationKey2019"))), nil
				}

				sDetails := sAuthentication.Details()
				pubKey, ok := sDetails.(base.Publickey)
				if !ok {
					return ctx, base.NewBaseOperationProcessReasonError(
						common.ErrMPreProcess.Errorf("%v", errors.Errorf("expected PublicKey, but %T", details))), nil
				}

				signature := base58.Decode(k.ProofData())

				err = pubKey.Verify(op.Fact().Hash().Bytes(), signature)
				if err != nil {
					return ctx, base.NewBaseOperationProcessReasonError(
						common.ErrMPreProcess.Errorf("signature verification failed, %v", err)), nil
				}

				opSender, ok := k.OpSender()
				if !ok {
					return ctx, base.NewBaseOperationProcessReasonError(
						common.ErrMPreProcess.Errorf("%v", errors.Errorf("empty op sender"))), nil
				}

				if _, _, aErr, cErr := state.ExistsCAccount(opSender, "op sender", true, false, getStateFunc); aErr != nil {
					return ctx, base.NewBaseOperationProcessReasonError(
						common.ErrMPreProcess.Errorf("%v", aErr)), nil
				} else if cErr != nil {
					return ctx, base.NewBaseOperationProcessReasonError(
						common.ErrMPreProcess.Errorf("%v", cErr)), nil
				}

				proxyPayer, ok := k.ProxyPayer()
				if ok {
					if _, _, aErr, cErr := state.ExistsCAccount(proxyPayer, "proxy payer", true, true, getStateFunc); aErr != nil {
						return ctx, base.NewBaseOperationProcessReasonError(
							common.ErrMPreProcess.Errorf("%v", aErr)), nil
					} else if cErr != nil {
						return ctx, base.NewBaseOperationProcessReasonError(
							common.ErrMPreProcess.Errorf("%v", cErr)), nil
					}
				}
			default:
			}
		} else {
			fact := op.Fact()
			signerFact, ok := fact.(currency.Signer)
			if ok {
				if err := state.CheckFactSignsByState(signerFact.Signer(), op.Signs(), getStateFunc); err != nil {
					return ctx,
						base.NewBaseOperationProcessReasonError(
							common.ErrMPreProcess.
								Wrap(common.ErrMSignInvalid).
								Errorf("%v", err),
						), nil
				}
			}
		}
	default:
	}

	return ctx, nil, nil
}

func (opr *OperationProcessor) Process(ctx context.Context, op base.Operation, getStateFunc base.GetStateFunc) ([]base.StateMergeValue, base.OperationProcessReasonError, error) {
	e := util.StringError("process for OperationProcessor")

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

	var isUserOperation bool
	switch i := op.Fact().(type) {
	case currency.FeeBaser:
		m, payer := i.FeeBase()
		switch k := op.(type) {
		case common.IExtendedOperation:
			if k.GetAuthentication() != nil && k.GetSettlement() != nil {
				isUserOperation = true
				opSender, _ := k.OpSender()
				payer = opSender
				if proxyPayer, ok := k.ProxyPayer(); ok {
					payer = proxyPayer
				}
				//
				//
				//if _, _, aErr, cErr := state.ExistsCAccount(payer, "payer", true, true, getStateFunc); aErr != nil {
				//	return nil, base.NewBaseOperationProcessReasonError(
				//		common.ErrMPreProcess.Errorf("%v", aErr)), nil
				//} else if cErr != nil {
				//	return nil, base.NewBaseOperationProcessReasonError(
				//		common.ErrMPreProcess.Errorf("%v", cErr)), nil
				//}
			}
		default:
		}

		feeReceiveSts := map[types.CurrencyID]base.State{}
		var sendAmount = make(map[types.CurrencyID]common.Big)
		var feeRequired = make(map[types.CurrencyID]common.Big)
		for cid, amounts := range m {
			policy, err := state.ExistsCurrencyPolicy(cid, getStateFunc)
			if err != nil {
				return nil, nil, err
			}
			receiver := policy.Feeer().Receiver()
			if receiver == nil {
				continue
			}

			if err := state.CheckExistsState(statecurrency.AccountStateKey(receiver), getStateFunc); err != nil {
				return nil, nil, errors.Errorf("Feeer receiver account not found, %s", receiver)
			} else if st, found, err := getStateFunc(statecurrency.BalanceStateKey(receiver, cid)); err != nil {
				return nil, nil, errors.Errorf("Feeer receiver account not found, %s", receiver)
			} else if !found {
				return nil, nil, errors.Errorf("Feeer receiver account not found, %s", receiver)
			} else {
				feeReceiveSts[cid] = st
			}

			total := common.ZeroBig
			rq := common.ZeroBig
			for _, big := range amounts {
				switch k, err := policy.Feeer().Fee(big); {
				case err != nil:
					return nil, nil, err
				default:
					rq = rq.Add(k)
				}
				total = total.Add(big)
			}
			if v, found := feeRequired[cid]; !found {
				feeRequired[cid] = rq
			} else {
				feeRequired[cid] = v.Add(rq)
			}
			sendAmount[cid] = total
		}

		for cid, rq := range feeRequired {
			st, err := state.ExistsState(statecurrency.BalanceStateKey(payer, cid), fmt.Sprintf("balance of fee payer, %v", payer), getStateFunc)
			if err != nil {
				return nil, nil, e.Wrap(err)
			}

			v, ok := st.Value().(statecurrency.BalanceStateValue)
			if !ok {
				return nil, base.NewBaseOperationProcessReasonError(
					"expected %T, not %T",
					statecurrency.BalanceStateValue{},
					st.Value(),
				), nil
			}

			am, err := statecurrency.StateBalanceValue(st)
			if err != nil {
				return nil, nil, e.Wrap(err)
			}

			if !isUserOperation {
				reg := sendAmount[cid].Add(rq)
				if am.Big().Compare(reg) < 0 {
					return nil, base.NewBaseOperationProcessReasonError(
						"account, %s balance insufficient; %d < required %d", payer.String(), am.Big(), rq), nil
				}
			} else {
				if am.Big().Compare(rq) < 0 {
					return nil, base.NewBaseOperationProcessReasonError(
						"account, %s balance insufficient; %d < required %d", payer.String(), am.Big(), rq), nil
				}
			}

			_, feeReceiverFound := feeReceiveSts[cid]
			if feeReceiverFound {
				if st.Key() != feeReceiveSts[cid].Key() {
					stateMergeValues = append(stateMergeValues, common.NewBaseStateMergeValue(
						st.Key(),
						statecurrency.NewDeductBalanceStateValue(v.Amount.WithBig(rq)),
						func(height base.Height, st base.State) base.StateValueMerger {
							return statecurrency.NewBalanceStateValueMerger(height, st.Key(), cid, st)
						},
					))
					r, ok := feeReceiveSts[cid].Value().(statecurrency.BalanceStateValue)
					if !ok {
						return nil, base.NewBaseOperationProcessReasonError(
							"expected %T, not %T",
							statecurrency.BalanceStateValue{},
							feeReceiveSts[cid].Value(),
						), nil
					}
					stateMergeValues = append(
						stateMergeValues,
						common.NewBaseStateMergeValue(
							feeReceiveSts[cid].Key(),
							statecurrency.NewAddBalanceStateValue(r.Amount.WithBig(rq)),
							func(height base.Height, st base.State) base.StateValueMerger {
								return statecurrency.NewBalanceStateValueMerger(height, feeReceiveSts[cid].Key(), cid, st)
							},
						),
					)
				}
			}
		}
	default:
	}

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
	var duplicationTypeDID string
	var duplicationTypeDIDPubKey []string
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
		duplicationTypeSenderID = DuplicationKey(fact.Sender().String(), DuplicationTypeSender)
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
	case did_registry.RegisterModel:
		fact, ok := t.Fact().(did_registry.RegisterModelFact)
		if !ok {
			return errors.Errorf("expected %T, not %T", did_registry.RegisterModelFact{}, t.Fact())
		}
		duplicationTypeSenderID = DuplicationKey(fact.Sender().String(), DuplicationTypeSender)
		duplicationTypeContractID = DuplicationKey(fact.Contract().String(), DuplicationTypeContract)
	case did_registry.CreateDID:
		fact, ok := t.Fact().(did_registry.CreateDIDFact)
		if !ok {
			return errors.Errorf("expected %T, not %T", did_registry.CreateDIDFact{}, t.Fact())
		}
		duplicationTypeDIDPubKey = []string{DuplicationKey(
			fmt.Sprintf("%s:%s", fact.Contract().String(), fact.Sender()), DuplicationTypeDIDPubKey)}
		duplicationTypeSenderID = DuplicationKey(fact.Sender().String(), DuplicationTypeSender)
	case did_registry.DeactivateDID:
		fact, ok := t.Fact().(did_registry.DeactivateDIDFact)
		if !ok {
			return errors.Errorf("expected %T, not %T", did_registry.DeactivateDIDFact{}, t.Fact())
		}
		duplicationTypeDID = DuplicationKey(
			fmt.Sprintf("%s:%s", fact.Contract().String(), fact.DID()), DuplicationTypeDID)
		duplicationTypeSenderID = DuplicationKey(fact.Sender().String(), DuplicationTypeSender)
	case did_registry.ReactivateDID:
		fact, ok := t.Fact().(did_registry.ReactivateDIDFact)
		if !ok {
			return errors.Errorf("expected %T, not %T", did_registry.ReactivateDIDFact{}, t.Fact())
		}
		duplicationTypeDID = DuplicationKey(
			fmt.Sprintf("%s:%s", fact.Contract().String(), fact.DID()), DuplicationTypeDID)
		duplicationTypeSenderID = DuplicationKey(fact.Sender().String(), DuplicationTypeSender)
	default:
		return nil
	}

	if len(duplicationTypeSenderID) > 0 {
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
		if len(duplicationTypeDID) > 0 {
			if _, found := opr.Duplicated[duplicationTypeDID]; found {
				return errors.Errorf(
					"cannot use a duplicated contract-did for DID, %v within a proposal",
					duplicationTypeDID,
				)
			}

			opr.Duplicated[duplicationTypeDID] = struct{}{}
		}
		if len(duplicationTypeDIDPubKey) > 0 {
			for _, v := range duplicationTypeDIDPubKey {
				if _, found := opr.Duplicated[v]; found {
					return errors.Errorf(
						"cannot use a duplicated contract-publickey for DID, %v within a proposal",
						v,
					)
				}
				opr.Duplicated[v] = struct{}{}
			}
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
		extension.UpdateHandler,
		extension.Withdraw,
		did_registry.RegisterModel,
		did_registry.CreateDID,
		did_registry.DeactivateDID,
		did_registry.UpdateDIDDocument,
		did_registry.ReactivateDID:
		return nil, false, errors.Errorf("%T needs SetProcessor", t)
	default:
		return nil, false, nil
	}
}

func (opr *OperationProcessor) GetNewProcessorFromHintset(op base.Operation) (base.OperationProcessor, error) {
	var fA types.GetNewProcessor
	var fB types.GetNewProcessorWithProposal
	var iA types.GetNewProcessor
	var iB types.GetNewProcessorWithProposal
	var foundA, foundB bool
	if hinter, ok := op.(hint.Hinter); !ok {
		return nil, nil
	} else if iA, foundA = opr.processorHintSet.Find(hinter.Hint()); foundA {
		fA = iA
	} else if iB, foundB = opr.processorHintSetWithProposal.Find(hinter.Hint()); foundB {
		fB = iB
	} else {
		return nil, nil
	}

	var opp base.OperationProcessor
	var err error
	if foundA {
		opp, err = fA(opr.Height(), opr.GetStateFunc, nil, nil)
	}
	if foundB {
		opp, err = fB(opr.Height(), opr.proposal, opr.GetStateFunc, nil, nil)
	}

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

	//opr.pool = nil
	opr.proposal = nil
	opr.Duplicated = nil
	opr.duplicatedNewAddress = nil
	opr.processorClosers = &sync.Map{}

	operationProcessorPool.Put(opr)

	opr.Log().Debug().Msg("operation processors closed")
}
