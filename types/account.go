package types

import (
	"github.com/ProtoconNet/mitum-currency/v3/common"
	"github.com/ProtoconNet/mitum2/base"
	"github.com/ProtoconNet/mitum2/util"
	"github.com/ProtoconNet/mitum2/util/hint"
	"github.com/ProtoconNet/mitum2/util/valuehash"
	"github.com/pkg/errors"
)

var (
	AccountHint = hint.MustNewHint("mitum-currency-account-v0.0.1")
)

type Account struct {
	hint.BaseHinter
	h       util.Hash
	address base.Address
	keys    AccountKeys
}

func NewAccount(address base.Address, keys AccountKeys) (Account, error) {
	if address == nil {
		return Account{}, common.ErrValueInvalid.Wrap(errors.Errorf("address is nil"))
	} else if err := address.IsValid(nil); err != nil {
		return Account{}, err
	}

	// keys can be nil (zero account)
	if keys != nil {
		if err := keys.IsValid(nil); err != nil {
			return Account{}, err
		}
	}

	ac := Account{BaseHinter: hint.NewBaseHinter(AccountHint), address: address, keys: keys}
	ac.h = ac.GenerateHash()

	return ac, nil
}

func NewAccountFromKeys(keys AccountKeys) (Account, error) {
	if a, err := NewAddressFromKeys(keys); err != nil {
		return Account{}, err
	} else if ac, err := NewAccount(a, keys); err != nil {
		return Account{}, err
	} else {
		return ac, nil
	}
}

func (ac Account) Bytes() []byte {
	bs := make([][]byte, 2)
	bs[0] = ac.address.Bytes()

	if ac.keys != nil {
		bs[1] = ac.keys.Bytes()
	}

	return util.ConcatBytesSlice(bs...)
}

func (ac Account) Hash() util.Hash {
	return ac.h
}

func (ac Account) IsValid([]byte) error {
	if err := ac.BaseHinter.IsValid(nil); err != nil {
		return err
	}

	if err := util.CheckIsValiders(nil, false, ac.address); err != nil {
		return err
	}

	if err := util.CheckIsValiders(nil, true, ac.keys); err != nil {
		return err
	}

	return nil
}

func (ac Account) GenerateHash() util.Hash {
	return valuehash.NewSHA256(ac.Bytes())
}

func (ac Account) Address() base.Address {
	return ac.address
}

func (ac Account) Keys() AccountKeys {
	return ac.keys
}

func (ac Account) SetKeys(keys AccountKeys) (Account, error) {
	if err := keys.IsValid(nil); err != nil {
		return Account{}, err
	}

	ac.keys = keys

	return ac, nil
}

func ZeroAccount(cid CurrencyID) (Account, error) {
	return NewAccount(ZeroAddress(cid), nil)
}
