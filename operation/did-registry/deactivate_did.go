package did_registry

import (
	"github.com/ProtoconNet/mitum-currency/v3/common"
	"github.com/ProtoconNet/mitum-currency/v3/operation/extras"
	currencytypes "github.com/ProtoconNet/mitum-currency/v3/types"
	mitumbase "github.com/ProtoconNet/mitum2/base"
	"github.com/ProtoconNet/mitum2/util"
	"github.com/ProtoconNet/mitum2/util/hint"
	"github.com/ProtoconNet/mitum2/util/valuehash"
	"github.com/pkg/errors"
)

var (
	DeactivateDIDFactHint = hint.MustNewHint("mitum-did-deactivate-did-operation-fact-v0.0.1")
	DeactivateDIDHint     = hint.MustNewHint("mitum-did-deactivate-did-operation-v0.0.1")
)

type DeactivateDIDFact struct {
	mitumbase.BaseFact
	sender   mitumbase.Address
	contract mitumbase.Address
	did      string
	currency currencytypes.CurrencyID
}

func NewDeactivateDIDFact(
	token []byte, sender, contract mitumbase.Address,
	did string, currency currencytypes.CurrencyID) DeactivateDIDFact {
	bf := mitumbase.NewBaseFact(DeactivateDIDFactHint, token)
	fact := DeactivateDIDFact{
		BaseFact: bf,
		sender:   sender,
		contract: contract,
		did:      did,
		currency: currency,
	}

	fact.SetHash(fact.GenerateHash())
	return fact
}

func (fact DeactivateDIDFact) IsValid(b []byte) error {
	if fact.sender.Equal(fact.contract) {
		return common.ErrFactInvalid.Wrap(
			common.ErrSelfTarget.Wrap(errors.Errorf("sender %v is same with contract account", fact.sender)))
	}

	if err := util.CheckIsValiders(nil, false,
		fact.BaseHinter,
		fact.sender,
		fact.contract,
		fact.currency,
	); err != nil {
		return common.ErrFactInvalid.Wrap(err)
	}

	if err := common.IsValidOperationFact(fact, b); err != nil {
		return common.ErrFactInvalid.Wrap(err)
	}

	return nil
}

func (fact DeactivateDIDFact) Hash() util.Hash {
	return fact.BaseFact.Hash()
}

func (fact DeactivateDIDFact) GenerateHash() util.Hash {
	return valuehash.NewSHA256(fact.Bytes())
}

func (fact DeactivateDIDFact) Bytes() []byte {
	return util.ConcatBytesSlice(
		fact.Token(),
		fact.sender.Bytes(),
		fact.contract.Bytes(),
		[]byte(fact.did),
		fact.currency.Bytes(),
	)
}

func (fact DeactivateDIDFact) Token() mitumbase.Token {
	return fact.BaseFact.Token()
}

func (fact DeactivateDIDFact) Sender() mitumbase.Address {
	return fact.sender
}

func (fact DeactivateDIDFact) Signer() mitumbase.Address {
	return fact.sender
}

func (fact DeactivateDIDFact) Contract() mitumbase.Address {
	return fact.contract
}

func (fact DeactivateDIDFact) DID() string {
	return fact.did
}

func (fact DeactivateDIDFact) Currency() currencytypes.CurrencyID {
	return fact.currency
}

func (fact DeactivateDIDFact) Addresses() ([]mitumbase.Address, error) {
	as := []mitumbase.Address{fact.sender}

	return as, nil
}

func (fact DeactivateDIDFact) FeeBase() map[currencytypes.CurrencyID][]common.Big {
	required := make(map[currencytypes.CurrencyID][]common.Big)
	required[fact.Currency()] = []common.Big{common.ZeroBig}

	return required
}

func (fact DeactivateDIDFact) FeePayer() mitumbase.Address {
	return fact.sender
}

func (fact DeactivateDIDFact) FactUser() mitumbase.Address {
	return fact.sender
}

func (fact DeactivateDIDFact) ActiveContract() mitumbase.Address {
	return fact.contract
}

type DeactivateDID struct {
	extras.ExtendedOperation
	//common.BaseOperation
	//*extras.BaseOperationExtensions
}

func NewDeactivateDID(fact DeactivateDIDFact) (DeactivateDID, error) {
	return DeactivateDID{
		ExtendedOperation: extras.NewExtendedOperation(DeactivateDIDHint, fact),
		//BaseOperation:           common.NewBaseOperation(DeactivateDIDHint, fact),
		//BaseOperationExtensions: extras.NewBaseOperationExtensions(),
	}, nil
}

//func (op DeactivateDID) IsValid(networkID []byte) error {
//	if err := op.BaseOperation.IsValid(networkID); err != nil {
//		return err
//	}
//	if err := op.BaseOperationExtensions.IsValid(networkID); err != nil {
//		return err
//	}
//
//	return nil
//}
