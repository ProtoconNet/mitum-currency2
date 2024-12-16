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
	ReactivateDIDFactHint = hint.MustNewHint("mitum-did-reactivate-did-operation-fact-v0.0.1")
	ReactivateDIDHint     = hint.MustNewHint("mitum-did-reactivate-did-operation-v0.0.1")
)

type ReactivateDIDFact struct {
	mitumbase.BaseFact
	sender   mitumbase.Address
	contract mitumbase.Address
	did      string
	currency currencytypes.CurrencyID
}

func NewReactivateDIDFact(
	token []byte, sender, contract mitumbase.Address,
	did string, currency currencytypes.CurrencyID) ReactivateDIDFact {
	bf := mitumbase.NewBaseFact(ReactivateDIDFactHint, token)
	fact := ReactivateDIDFact{
		BaseFact: bf,
		sender:   sender,
		contract: contract,
		did:      did,
		currency: currency,
	}

	fact.SetHash(fact.GenerateHash())
	return fact
}

func (fact ReactivateDIDFact) IsValid(b []byte) error {
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

func (fact ReactivateDIDFact) Hash() util.Hash {
	return fact.BaseFact.Hash()
}

func (fact ReactivateDIDFact) GenerateHash() util.Hash {
	return valuehash.NewSHA256(fact.Bytes())
}

func (fact ReactivateDIDFact) Bytes() []byte {
	return util.ConcatBytesSlice(
		fact.Token(),
		fact.sender.Bytes(),
		fact.contract.Bytes(),
		[]byte(fact.did),
		fact.currency.Bytes(),
	)
}

func (fact ReactivateDIDFact) Token() mitumbase.Token {
	return fact.BaseFact.Token()
}

func (fact ReactivateDIDFact) Sender() mitumbase.Address {
	return fact.sender
}

func (fact ReactivateDIDFact) Signer() mitumbase.Address {
	return fact.sender
}

func (fact ReactivateDIDFact) Contract() mitumbase.Address {
	return fact.contract
}

func (fact ReactivateDIDFact) DID() string {
	return fact.did
}

func (fact ReactivateDIDFact) Currency() currencytypes.CurrencyID {
	return fact.currency
}

func (fact ReactivateDIDFact) Addresses() ([]mitumbase.Address, error) {
	as := []mitumbase.Address{fact.sender}

	return as, nil
}

func (fact ReactivateDIDFact) FeeBase() map[currencytypes.CurrencyID][]common.Big {
	required := make(map[currencytypes.CurrencyID][]common.Big)
	required[fact.Currency()] = []common.Big{common.ZeroBig}

	return required
}

func (fact ReactivateDIDFact) FeePayer() mitumbase.Address {
	return fact.sender
}

func (fact ReactivateDIDFact) FactUser() mitumbase.Address {
	return fact.sender
}

func (fact ReactivateDIDFact) ActiveContract() mitumbase.Address {
	return fact.contract
}

type ReactivateDID struct {
	extras.ExtendedOperation
	//common.BaseOperation
	//*extras.BaseOperationExtensions
}

func NewReactivateDID(fact ReactivateDIDFact) (ReactivateDID, error) {
	return ReactivateDID{
		ExtendedOperation: extras.NewExtendedOperation(ReactivateDIDHint, fact),
		//BaseOperation:           common.NewBaseOperation(ReactivateDIDHint, fact),
		//BaseOperationExtensions: extras.NewBaseOperationExtensions(),
	}, nil
}

//func (op ReactivateDID) IsValid(networkID []byte) error {
//	if err := op.BaseOperation.IsValid(networkID); err != nil {
//		return err
//	}
//	if err := op.BaseOperationExtensions.IsValid(networkID); err != nil {
//		return err
//	}
//
//	return nil
//}
