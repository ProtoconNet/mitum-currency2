package common

import (
	"github.com/ProtoconNet/mitum2/base"
	"github.com/ProtoconNet/mitum2/util"
)

type Settlement interface {
	ProxyPayer() base.Address
	VerifyPayment(base.GetStateFunc) error
	Bytes() []byte
}

type BaseSettlement struct {
	proxyPayer base.Address
}

func NewBaseSettlement(proxyPayer base.Address) BaseSettlement {
	return BaseSettlement{
		proxyPayer: proxyPayer,
	}
}

func (ba BaseSettlement) ProxyPayer() base.Address {
	return ba.proxyPayer
}

func (ba BaseSettlement) VerifyPayment(getStateFunc base.GetStateFunc) error {
	return nil
}

func (ba BaseSettlement) Bytes() []byte {
	var bs [][]byte
	bs = append(bs, ba.proxyPayer.Bytes())
	return util.ConcatBytesSlice(bs...)
}

func (ba BaseSettlement) Equal(b BaseSettlement) bool {
	if !ba.proxyPayer.Equal(b.proxyPayer) {
		return false
	}

	return true
}

type Authentication interface {
	Contract() base.Address
	AuthenticationID() string
	ProofData() string
	VerifyAuth(base.GetStateFunc) error
	Bytes() []byte
}

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

func (ba BaseAuthentication) Contract() base.Address {
	return ba.contract
}

func (ba BaseAuthentication) AuthenticationID() string {
	return ba.authenticationID
}

func (ba BaseAuthentication) ProofData() string {
	return ba.proofData
}

func (ba BaseAuthentication) Bytes() []byte {
	var bs [][]byte
	bs = append(bs, ba.contract.Bytes())
	bs = append(bs, []byte(ba.authenticationID))
	bs = append(bs, []byte(ba.proofData))
	return util.ConcatBytesSlice(bs...)
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

func (ba BaseAuthentication) VerifyAuth(getStateFunc base.GetStateFunc) error {
	//var authentication types.IAuthentication
	//var doc types.DIDDocument
	//dr, err := types.NewDIDResourceFromString(ba.authenticationID)
	//if err != nil {
	//	return err
	//}
	//
	//if st, err := state.ExistsState(didstate.DocumentStateKey(ba.contract, dr.DID()), "did document", getStateFunc); err != nil {
	//	return err
	//} else if doc, err = didstate.GetDocumentFromState(st); err != nil {
	//	return err
	//}
	//
	//authentication, err = doc.Authentication(ba.authenticationID)
	//
	//if authentication.ID() != dr.DID() {
	//	return errors.Errorf("did not matched")
	//}

	//pubKey := hex.Decode(authentication.PublicKey())
	//signature := base58.Decode(ba.proof)
	//
	//if !ed25519.Verify(pubKey, ba.message.Bytes(), signature) {
	//
	//}
	return nil
}

type OperationExtension interface {
	ExtType() string
	RunExtension(base.GetStateFunc) error
	Bytes() []byte
}

type UserAuthentication struct {
	authentication Authentication
	extType        string
}

func NewUserAuthentication(
	authentication Authentication,
) UserAuthentication {
	return UserAuthentication{
		authentication: authentication,
		extType:        "UserAuthentication",
	}
}

func (ba UserAuthentication) ExtType() string {
	return ba.extType
}

func (ba UserAuthentication) RunExtension(getStateFunc base.GetStateFunc) error {
	err := ba.authentication.VerifyAuth(getStateFunc)
	if err != nil {
		return err
	}
	return nil
}

func (ba UserAuthentication) Bytes() []byte {
	var bs [][]byte
	bs = append(bs, ba.authentication.Bytes())
	bs = append(bs, []byte(ba.extType))
	return util.ConcatBytesSlice(bs...)
}

//type ExtendedOperation interface {
//	Extension() OperationExtension
//}

//type ExtendedOperation struct {
//	BaseOperation
//	UserAuthentication
//	Settlement
//}
//
//func NewExtendedOperation(
//	op BaseOperation, userAuthentication UserAuthentication, settlement Settlement,
//) ExtendedOperation {
//	return ExtendedOperation{
//		BaseOperation:      op,
//		UserAuthentication: userAuthentication,
//		Settlement:         settlement,
//	}
//}
//
//func (op ExtendedOperation) HashBytes() []byte {
//	var bs []util.Byter
//	bs = append(bs, op.BaseOperation)
//	bs = append(bs, op.UserAuthentication)
//	bs = append(bs, op.Settlement)
//	return util.ConcatByters(bs...)
//}
//
//func (op ExtendedOperation) IsValid(networkID []byte) error {
//	return nil
//}
