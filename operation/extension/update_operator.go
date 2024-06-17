package extension

import (
	"github.com/ProtoconNet/mitum-currency/v3/common"
	"github.com/ProtoconNet/mitum-currency/v3/types"
	"github.com/ProtoconNet/mitum2/base"
	"github.com/ProtoconNet/mitum2/util"
	"github.com/ProtoconNet/mitum2/util/hint"
	"github.com/ProtoconNet/mitum2/util/valuehash"
	"github.com/pkg/errors"
)

var (
	UpdateOperatorFactHint = hint.MustNewHint("mitum-currency-contract-account-update-operator-operation-fact-v0.0.1")
	UpdateOperatorHint     = hint.MustNewHint("mitum-currency-contract-account-update-operator-operation-v0.0.1")
)

const MaxOperators = 10

type UpdateOperatorFact struct {
	base.BaseFact
	sender    base.Address
	contract  base.Address
	operators []base.Address
	currency  types.CurrencyID
}

func NewUpdateOperatorFact(
	token []byte,
	sender,
	contract base.Address,
	operators []base.Address,
	currency types.CurrencyID,
) UpdateOperatorFact {
	fact := UpdateOperatorFact{
		BaseFact:  base.NewBaseFact(UpdateOperatorFactHint, token),
		sender:    sender,
		contract:  contract,
		operators: operators,
		currency:  currency,
	}

	fact.SetHash(fact.GenerateHash())

	return fact
}

func (fact UpdateOperatorFact) Hash() util.Hash {
	return fact.BaseFact.Hash()
}

func (fact UpdateOperatorFact) Bytes() []byte {
	bs := make([][]byte, len(fact.operators)+4)
	bs[0] = fact.Token()
	bs[1] = fact.sender.Bytes()
	bs[2] = fact.contract.Bytes()
	bs[3] = fact.currency.Bytes()
	for i := range fact.operators {
		bs[4+i] = fact.operators[i].Bytes()
	}

	return util.ConcatBytesSlice(bs...)
}

func (fact UpdateOperatorFact) IsValid(b []byte) error {
	if err := fact.BaseHinter.IsValid(nil); err != nil {
		return common.ErrFactInvalid.Wrap(err)
	}

	if err := util.CheckIsValiders(nil, false, fact.sender, fact.contract, fact.currency); err != nil {
		return common.ErrFactInvalid.Wrap(err)
	}

	if n := len(fact.operators); n < 1 {
		return common.ErrFactInvalid.Wrap(common.ErrArrayLen.Wrap(errors.Errorf("empty operators")))
	} else if n > MaxOperators {
		return common.ErrFactInvalid.Wrap(common.ErrArrayLen.Wrap(errors.Errorf("operators, %d over max, %d", n, MaxOperators)))
	}

	operatorsMap := make(map[string]struct{})
	for i := range fact.operators {
		_, found := operatorsMap[fact.operators[i].String()]
		if found {
			return common.ErrFactInvalid.Wrap(common.ErrDupVal.Wrap(errors.Errorf("operator, %v", fact.operators[i])))
		} else {
			operatorsMap[fact.operators[i].String()] = struct{}{}
		}
		if err := fact.operators[i].IsValid(nil); err != nil {
			return common.ErrFactInvalid.Wrap(common.ErrValOOR.Wrap(errors.Errorf("invalid operator address, %v", err)))
		}
	}

	if err := common.IsValidOperationFact(fact, b); err != nil {
		return common.ErrFactInvalid.Wrap(err)
	}

	return nil
}

func (fact UpdateOperatorFact) GenerateHash() util.Hash {
	return valuehash.NewSHA256(fact.Bytes())
}

func (fact UpdateOperatorFact) Token() base.Token {
	return fact.BaseFact.Token()
}

func (fact UpdateOperatorFact) Currency() types.CurrencyID {
	return fact.currency
}

func (fact UpdateOperatorFact) Sender() base.Address {
	return fact.sender
}

func (fact UpdateOperatorFact) Contract() base.Address {
	return fact.contract
}

func (fact UpdateOperatorFact) Operators() []base.Address {
	return fact.operators
}

func (fact UpdateOperatorFact) Addresses() ([]base.Address, error) {
	as := make([]base.Address, len(fact.operators)+2)

	oprs := fact.operators
	copy(as, oprs)

	as[len(fact.operators)] = fact.sender
	as[len(fact.operators)+1] = fact.contract

	return as, nil
}

type UpdateOperator struct {
	common.BaseOperation
}

func NewUpdateOperator(fact UpdateOperatorFact) (UpdateOperator, error) {
	return UpdateOperator{BaseOperation: common.NewBaseOperation(UpdateOperatorHint, fact)}, nil
}

func (op *UpdateOperator) HashSign(priv base.Privatekey, networkID base.NetworkID) error {
	err := op.Sign(priv, networkID)
	if err != nil {
		return err
	}
	return nil
}
