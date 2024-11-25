package common

import (
	"github.com/ProtoconNet/mitum2/base"
	"github.com/ProtoconNet/mitum2/util"
	"github.com/pkg/errors"
)

type ExtendedFact struct {
	base.Fact
	user base.Address
}

func NewExtendedFact(fact base.Fact, user base.Address) ExtendedFact {
	return ExtendedFact{
		Fact: fact,
		user: user,
	}
}

func (fact ExtendedFact) User() base.Address {
	return fact.user
}

func (fact ExtendedFact) HashBytes() []byte {
	var bs []util.Byter
	b, _ := fact.Fact.(util.Byter)
	bs = append(bs, b)
	bs = append(bs, fact.user)
	return util.ConcatByters(bs...)
}

func (fact ExtendedFact) IsValid(networkID []byte) error {
	_, ok := fact.Fact.(util.Byter)
	if !ok {
		return errors.Errorf("Fact not implemented Byter")
	}

	return nil
}
