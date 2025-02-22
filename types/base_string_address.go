package types

import (
	"regexp"

	"github.com/ProtoconNet/mitum2/base"
	"github.com/ProtoconNet/mitum2/util"
	"github.com/ProtoconNet/mitum2/util/hint"
)

var (
	MaxAddressSize             = 100
	MinAddressSize             = base.AddressTypeSize + 3
	reBlankStringAddressString = regexp.MustCompile(`[\s][\s]*`)
	REStringAddressString      = `[a-zA-Z0-9][\w]*[a-zA-Z0-9]`
	reStringAddressString      = regexp.MustCompile(`^` + REStringAddressString + `$`)
)

type BaseStringAddress struct {
	s string
	hint.BaseHinter
}

func NewBaseStringAddressWithHint(ht hint.Hint, s string) BaseStringAddress {
	ad := BaseStringAddress{BaseHinter: hint.NewBaseHinter(ht)}
	ad.s = s + ht.Type().String()

	return ad
}

func (ad BaseStringAddress) IsValid([]byte) error {
	switch l := len(ad.s); {
	case l < MinAddressSize:
		return util.ErrInvalid.Errorf("Too short string address")
	case l > MaxAddressSize:
		return util.ErrInvalid.Errorf("Too long string address")
	}

	p := ad.s[:len(ad.s)-base.AddressTypeSize]
	if reBlankStringAddressString.MatchString(p) {
		return util.ErrInvalid.Errorf("String address string, %v has blank", ad)
	}

	if !reStringAddressString.MatchString(p) {
		return util.ErrInvalid.Errorf("Invalid string address string, %v", ad)
	}

	switch {
	case len(ad.Hint().Type().String()) != base.AddressTypeSize:
		return util.ErrInvalid.Errorf("Wrong hint of string address")
	case ad.s[len(ad.s)-base.AddressTypeSize:] != ad.Hint().Type().String():
		return util.ErrInvalid.Errorf(
			"Wrong type of string address; %v != %v", ad.s[len(ad.s)-base.AddressTypeSize:], ad.Hint().Type())
	}

	return nil
}

func (ad BaseStringAddress) String() string {
	return ad.s
}

func (ad BaseStringAddress) Bytes() []byte {
	return []byte(ad.s)
}

func (ad BaseStringAddress) Equal(b base.Address) bool {
	if b == nil {
		return false
	}

	return ad.s == b.String()
}

func (ad BaseStringAddress) MarshalText() ([]byte, error) {
	return []byte(ad.s), nil
}
