package did_registry

import (
	"github.com/ProtoconNet/mitum-currency/v3/common"
	currencytypes "github.com/ProtoconNet/mitum-currency/v3/types"
	mitumbase "github.com/ProtoconNet/mitum2/base"
	"github.com/ProtoconNet/mitum2/util"
	"github.com/ProtoconNet/mitum2/util/hint"
	"github.com/ProtoconNet/mitum2/util/valuehash"
	"github.com/pkg/errors"
)

var (
	CreateDIDFactHint = hint.MustNewHint("mitum-did-create-did-operation-fact-v0.0.1")
	CreateDIDHint     = hint.MustNewHint("mitum-did-create-did-operation-v0.0.1")
)

type CreateDIDFact struct {
	mitumbase.BaseFact
	sender          mitumbase.Address
	contract        mitumbase.Address
	address         mitumbase.Address
	authType        string
	publicKey       mitumbase.Publickey
	serviceType     string
	serviceEndpoint string
	currency        currencytypes.CurrencyID
}

func NewCreateDIDFact(
	token []byte, sender, contract, address mitumbase.Address,
	authType string, publicKey mitumbase.Publickey, serviceType, serviceEndpoint string, currency currencytypes.CurrencyID) CreateDIDFact {
	bf := mitumbase.NewBaseFact(CreateDIDFactHint, token)
	fact := CreateDIDFact{
		BaseFact:        bf,
		sender:          sender,
		contract:        contract,
		address:         address,
		authType:        authType,
		publicKey:       publicKey,
		serviceType:     serviceType,
		serviceEndpoint: serviceEndpoint,
		currency:        currency,
	}

	fact.SetHash(fact.GenerateHash())
	return fact
}

func (fact CreateDIDFact) IsValid(b []byte) error {
	if fact.sender.Equal(fact.contract) {
		return common.ErrFactInvalid.Wrap(
			common.ErrSelfTarget.Wrap(errors.Errorf("sender %v is same with contract account", fact.sender)))
	}

	if err := util.CheckIsValiders(nil, false,
		fact.BaseHinter,
		fact.sender,
		fact.contract,
		fact.address,
		fact.currency,
	); err != nil {
		return common.ErrFactInvalid.Wrap(err)
	}

	if err := common.IsValidOperationFact(fact, b); err != nil {
		return common.ErrFactInvalid.Wrap(err)
	}

	return nil
}

func (fact CreateDIDFact) Hash() util.Hash {
	return fact.BaseFact.Hash()
}

func (fact CreateDIDFact) GenerateHash() util.Hash {
	return valuehash.NewSHA256(fact.Bytes())
}

func (fact CreateDIDFact) Bytes() []byte {
	return util.ConcatBytesSlice(
		fact.Token(),
		fact.sender.Bytes(),
		fact.contract.Bytes(),
		fact.address.Bytes(),
		[]byte(fact.authType),
		fact.publicKey.Bytes(),
		[]byte(fact.serviceType),
		[]byte(fact.serviceEndpoint),
		fact.currency.Bytes(),
	)
}

func (fact CreateDIDFact) Token() mitumbase.Token {
	return fact.BaseFact.Token()
}

func (fact CreateDIDFact) Sender() mitumbase.Address {
	return fact.sender
}

func (fact CreateDIDFact) Contract() mitumbase.Address {
	return fact.contract
}

func (fact CreateDIDFact) Address() mitumbase.Address {
	return fact.address
}

func (fact CreateDIDFact) AuthType() string {
	return fact.authType
}

func (fact CreateDIDFact) PublicKey() mitumbase.Publickey {
	return fact.publicKey
}

func (fact CreateDIDFact) ServiceType() string {
	return fact.serviceType
}

func (fact CreateDIDFact) ServiceEndpoint() string {
	return fact.serviceEndpoint
}

func (fact CreateDIDFact) Currency() currencytypes.CurrencyID {
	return fact.currency
}

func (fact CreateDIDFact) Addresses() ([]mitumbase.Address, error) {
	as := []mitumbase.Address{fact.sender}

	return as, nil
}

func (fact CreateDIDFact) FeeBase() (map[currencytypes.CurrencyID][]common.Big, mitumbase.Address) {
	required := make(map[currencytypes.CurrencyID][]common.Big)
	required[fact.Currency()] = []common.Big{common.ZeroBig}

	return required, fact.sender
}

type CreateDID struct {
	common.BaseOperation
}

func NewCreateDID(fact CreateDIDFact) (CreateDID, error) {
	return CreateDID{BaseOperation: common.NewBaseOperation(CreateDIDHint, fact)}, nil
}
