package common

import (
	"github.com/ProtoconNet/mitum2/base"
	"github.com/ProtoconNet/mitum2/util"
	"github.com/pkg/errors"
)

type BaseAuthentication struct {
	contract         base.Address
	authenticationID string
	proofData        string
}

func NewBaseAuthentication(contract base.Address, authenticationID, proofData string) BaseAuthentication {
	return BaseAuthentication{
		contract:         contract,
		authenticationID: authenticationID,
		proofData:        proofData,
	}
}

func (ba BaseAuthentication) Contract() (base.Address, bool) {
	if ba.contract == nil {
		return nil, false
	}
	return ba.contract, true
}

func (ba BaseAuthentication) AuthenticationID() string {
	return ba.authenticationID
}

func (ba BaseAuthentication) ProofData() string {
	return ba.proofData
}

func (ba BaseAuthentication) Bytes() []byte {
	if ba.Equal(BaseAuthentication{}) {
		return []byte{}
	}
	var bs [][]byte
	bs = append(bs, ba.contract.Bytes())
	bs = append(bs, []byte(ba.authenticationID))
	bs = append(bs, []byte(ba.proofData))
	return util.ConcatBytesSlice(bs...)
}

func (ba BaseAuthentication) IsValid([]byte) error {
	if err := util.CheckIsValiders(nil, false, ba.contract); err != nil {
		return ErrValueInvalid.Wrap(err)
	}

	if len(ba.authenticationID) < 1 {
		return ErrValueInvalid.Wrap(errors.Errorf("empty authentication id"))
	}

	if len(ba.proofData) < 1 {
		return ErrValueInvalid.Wrap(errors.Errorf("empty proof data"))
	}

	return nil
}

func (ba BaseAuthentication) Equal(b BaseAuthentication) bool {
	if !ba.contract.Equal(b.contract) {
		return false
	}

	if ba.authenticationID != b.authenticationID {
		return false
	}

	if ba.proofData != b.proofData {
		return false
	}

	return true
}

type BaseSettlement struct {
	opSender   base.Address
	proxyPayer base.Address
}

func NewBaseSettlement(opSender, proxyPayer base.Address) BaseSettlement {
	return BaseSettlement{
		opSender:   opSender,
		proxyPayer: proxyPayer,
	}
}

func (ba BaseSettlement) OpSender() (base.Address, bool) {
	if ba.opSender == nil {
		return nil, false
	}
	return ba.opSender, true
}

func (ba BaseSettlement) ProxyPayer() (base.Address, bool) {
	if ba.proxyPayer == nil {
		return nil, false
	}
	return ba.proxyPayer, true
}

func (ba BaseSettlement) Bytes() []byte {
	if ba.Equal(BaseSettlement{}) {
		return []byte{}
	}
	var bs [][]byte
	bs = append(bs, ba.opSender.Bytes())
	if ba.proxyPayer != nil {
		bs = append(bs, ba.proxyPayer.Bytes())
	}
	return util.ConcatBytesSlice(bs...)
}

func (ba BaseSettlement) IsValid([]byte) error {
	if err := util.CheckIsValiders(nil, false, ba.opSender); err != nil {
		return ErrValueInvalid.Wrap(err)
	}

	if err := util.CheckIsValiders(nil, true, ba.proxyPayer); err != nil {
		return ErrValueInvalid.Wrap(err)
	}

	return nil
}

func (ba BaseSettlement) Equal(b BaseSettlement) bool {
	if !ba.opSender.Equal(b.opSender) {
		return false
	}

	if !ba.proxyPayer.Equal(b.proxyPayer) {
		return false
	}

	return true
}
