package currency

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
	CreateAccountFactHint = hint.MustNewHint("mitum-currency-create-account-operation-fact-v0.0.1")
	CreateAccountHint     = hint.MustNewHint("mitum-currency-create-account-operation-v0.0.1")
)

var MaxCreateAccountItems uint = 100

type FeeBaser interface {
	FeeBase() (map[types.CurrencyID][]common.Big, base.Address)
}

type AmountsItem interface {
	Amounts() []types.Amount
}

type CreateAccountItem interface {
	hint.Hinter
	util.IsValider
	AmountsItem
	Bytes() []byte
	Keys() types.AccountKeys
	Address() (base.Address, error)
	Rebuild() CreateAccountItem
}

type CreateAccountFact struct {
	base.BaseFact
	sender base.Address
	user   base.Address
	items  []CreateAccountItem
}

func NewCreateAccountFact(
	token []byte,
	sender base.Address,
	items []CreateAccountItem,
) CreateAccountFact {
	bf := base.NewBaseFact(CreateAccountFactHint, token)
	fact := CreateAccountFact{
		BaseFact: bf,
		sender:   sender,
		items:    items,
	}
	fact.SetHash(fact.GenerateHash())

	return fact
}

func (fact CreateAccountFact) Hash() util.Hash {
	return fact.BaseFact.Hash()
}

func (fact CreateAccountFact) GenerateHash() util.Hash {
	return valuehash.NewSHA256(fact.Bytes())
}

func (fact CreateAccountFact) Bytes() []byte {
	is := make([][]byte, len(fact.items))
	for i := range fact.items {
		is[i] = fact.items[i].Bytes()
	}

	return util.ConcatBytesSlice(
		fact.Token(),
		fact.sender.Bytes(),
		util.ConcatBytesSlice(is...),
	)
}

func (fact CreateAccountFact) IsValid(b []byte) error {
	if err := fact.BaseHinter.IsValid(nil); err != nil {
		return common.ErrFactInvalid.Wrap(err)
	}

	if n := len(fact.items); n < 1 {
		return common.ErrFactInvalid.Wrap(common.ErrArrayLen.Wrap(errors.Errorf("Empty items")))
	} else if n > int(MaxCreateAccountItems) {
		return common.ErrFactInvalid.Wrap(common.ErrArrayLen.Wrap(errors.Errorf("Items, %d over max, %d", n, MaxCreateAccountItems)))
	}

	if err := util.CheckIsValiders(nil, false, fact.sender); err != nil {
		return common.ErrFactInvalid.Wrap(err)
	}

	foundKeys := map[string]struct{}{}
	for i := range fact.items {
		it := fact.items[i]
		if err := util.CheckIsValiders(nil, false, it); err != nil {
			return common.ErrFactInvalid.Wrap(err)
		}

		k := it.Keys().Hash().String()
		if _, found := foundKeys[k]; found {
			return common.ErrFactInvalid.Wrap(common.ErrDupVal.Wrap(errors.Errorf("account Keys, %s", k)))
		}

		switch a, err := it.Address(); {
		case err != nil:
			return common.ErrFactInvalid.Wrap(err)
		case fact.sender.Equal(a):
			return common.ErrFactInvalid.Wrap(common.ErrSelfTarget.Wrap(errors.Errorf("Target account is same with sender account, %v", fact.sender)))
		default:
			foundKeys[k] = struct{}{}
		}
	}

	if err := common.IsValidOperationFact(fact, b); err != nil {
		return common.ErrFactInvalid.Wrap(err)
	}

	return nil
}

func (fact CreateAccountFact) Token() base.Token {
	return fact.BaseFact.Token()
}

func (fact CreateAccountFact) Sender() base.Address {
	return fact.sender
}

func (fact CreateAccountFact) User() base.Address {
	return fact.user
}

func (fact CreateAccountFact) Items() []CreateAccountItem {
	return fact.items
}

func (fact CreateAccountFact) Targets() ([]base.Address, error) {
	as := make([]base.Address, len(fact.items))
	for i := range fact.items {
		a, err := fact.items[i].Address()
		if err != nil {
			return nil, err
		}
		as[i] = a
	}

	return as, nil
}

func (fact CreateAccountFact) Addresses() ([]base.Address, error) {
	as := make([]base.Address, len(fact.items)+1)

	tas, err := fact.Targets()
	if err != nil {
		return nil, err
	}
	copy(as, tas)

	as[len(fact.items)] = fact.Sender()

	return as, nil
}

func (fact CreateAccountFact) Rebuild() CreateAccountFact {
	items := make([]CreateAccountItem, len(fact.items))
	for i := range fact.items {
		it := fact.items[i]
		items[i] = it.Rebuild()
	}

	fact.items = items
	fact.SetHash(fact.Hash())

	return fact
}

func (fact CreateAccountFact) FeeBase() (map[types.CurrencyID][]common.Big, base.Address) {
	required := make(map[types.CurrencyID][]common.Big)
	items := make([]AmountsItem, len(fact.items))
	for i := range fact.items {
		items[i] = fact.items[i]
	}

	for i := range items {
		it := items[i]
		amounts := it.Amounts()
		for j := range amounts {
			am := amounts[j]
			cid := am.Currency()
			big := am.Big()
			var k []common.Big
			if arr, found := required[cid]; found {
				arr = append(arr, big)
				copy(k, arr)
			} else {
				k = append(k, big)
			}
			required[cid] = k
		}
	}

	return required, fact.Sender()
}

type CreateAccount struct {
	common.BaseOperation
}

func NewCreateAccount(fact CreateAccountFact) (CreateAccount, error) {
	return CreateAccount{BaseOperation: common.NewBaseOperation(CreateAccountHint, fact)}, nil
}

func (op *CreateAccount) HashSign(priv base.Privatekey, networkID base.NetworkID) error {
	err := op.Sign(priv, networkID)
	if err != nil {
		return err
	}
	return nil
}

//func (op *CreateAccount) Error(v error, err error) error {
//	var nerr error
//
//	switch {
//	case errors.Is(v, common.ErrDecodeBson):
//		nerr = common.ErrDecodeBson.WithMessage(err, "%T", *op)
//	case errors.Is(v, common.ErrDecodeJson):
//		nerr = common.ErrDecodeJson.WithMessage(err, "%T", *op)
//	default:
//		nerr = err
//	}
//
//	return nerr
//}
