package did_registry

import (
	"github.com/ProtoconNet/mitum-currency/v3/common"
	"github.com/ProtoconNet/mitum-currency/v3/types"
	currencytypes "github.com/ProtoconNet/mitum-currency/v3/types"
	mitumbase "github.com/ProtoconNet/mitum2/base"
	"github.com/ProtoconNet/mitum2/util"
	"github.com/ProtoconNet/mitum2/util/hint"
	"github.com/ProtoconNet/mitum2/util/valuehash"
	"github.com/pkg/errors"
)

var (
	UpdateDIDDocumentFactHint = hint.MustNewHint("mitum-did-update-did-document-operation-fact-v0.0.1")
	UpdateDIDDocumentHint     = hint.MustNewHint("mitum-did-update-did-document-operation-v0.0.1")
)

type UpdateDIDDocumentFact struct {
	mitumbase.BaseFact
	sender   mitumbase.Address
	contract mitumbase.Address
	did      string
	document types.DIDDocument
	currency currencytypes.CurrencyID
}

func NewUpdateDIDDocumentFact(
	token []byte, sender, contract mitumbase.Address,
	did string, doc types.DIDDocument, currency currencytypes.CurrencyID) UpdateDIDDocumentFact {
	bf := mitumbase.NewBaseFact(UpdateDIDDocumentFactHint, token)
	fact := UpdateDIDDocumentFact{
		BaseFact: bf,
		sender:   sender,
		contract: contract,
		did:      did,
		document: doc,
		currency: currency,
	}

	fact.SetHash(fact.GenerateHash())
	return fact
}

func (fact UpdateDIDDocumentFact) IsValid(b []byte) error {
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

func (fact UpdateDIDDocumentFact) Hash() util.Hash {
	return fact.BaseFact.Hash()
}

func (fact UpdateDIDDocumentFact) GenerateHash() util.Hash {
	return valuehash.NewSHA256(fact.Bytes())
}

func (fact UpdateDIDDocumentFact) Bytes() []byte {
	return util.ConcatBytesSlice(
		fact.Token(),
		fact.sender.Bytes(),
		fact.contract.Bytes(),
		[]byte(fact.did),
		fact.document.Bytes(),
		fact.currency.Bytes(),
	)
}

func (fact UpdateDIDDocumentFact) Token() mitumbase.Token {
	return fact.BaseFact.Token()
}

func (fact UpdateDIDDocumentFact) Sender() mitumbase.Address {
	return fact.sender
}

func (fact UpdateDIDDocumentFact) Signer() mitumbase.Address {
	return fact.sender
}

func (fact UpdateDIDDocumentFact) Contract() mitumbase.Address {
	return fact.contract
}

func (fact UpdateDIDDocumentFact) DID() string {
	return fact.did
}

func (fact UpdateDIDDocumentFact) Document() types.DIDDocument {
	return fact.document
}

func (fact UpdateDIDDocumentFact) Currency() currencytypes.CurrencyID {
	return fact.currency
}

func (fact UpdateDIDDocumentFact) Addresses() ([]mitumbase.Address, error) {
	as := []mitumbase.Address{fact.sender}

	return as, nil
}

func (fact UpdateDIDDocumentFact) FeeBase() (map[types.CurrencyID][]common.Big, mitumbase.Address) {
	required := make(map[types.CurrencyID][]common.Big)
	required[fact.Currency()] = []common.Big{common.ZeroBig}

	return required, fact.sender
}

type UpdateDIDDocument struct {
	common.BaseOperation
}

func NewUpdateDIDDocument(fact UpdateDIDDocumentFact) (UpdateDIDDocument, error) {
	return UpdateDIDDocument{BaseOperation: common.NewBaseOperation(UpdateDIDDocumentHint, fact)}, nil
}
