package common

import (
	"context"
	"encoding/json"
	"fmt"
	"golang.org/x/exp/slices"

	"github.com/ProtoconNet/mitum2/base"
	"github.com/ProtoconNet/mitum2/util"
	"github.com/ProtoconNet/mitum2/util/hint"
	"github.com/ProtoconNet/mitum2/util/valuehash"
	"github.com/pkg/errors"
)

type IExtendedOperation interface {
	Contract() (base.Address, bool)
	AuthenticationID() string
	ProofData() string
	VerifyAuth(base.GetStateFunc) error
	OpSender() (base.Address, bool)
	ProxyPayer() (base.Address, bool)
	VerifyPayment(base.GetStateFunc) error
	GetAuthentication() Authentication
	GetSettlement() Settlement
}

//type IAuthentication interface {
//	Contract() base.Address
//	AuthenticationID() string
//	ProofData() string
//	VerifyAuth(base.GetStateFunc) error
//	Bytes() []byte
//}

//type ISettlement interface {
//	ProxyPayer() base.Address
//	VerifyPayment(base.GetStateFunc) error
//	Bytes() []byte
//}

type ExtendedOperation struct {
	Authentication
	Settlement
}

func NewExtendedOperation(authentication Authentication, settlement Settlement) ExtendedOperation {
	return ExtendedOperation{
		Authentication: authentication,
		Settlement:     settlement,
	}
}

func (op *ExtendedOperation) SetAuthentication(authentication Authentication) {
	op.Authentication = authentication
}

func (op *ExtendedOperation) SetSettlement(settlement Settlement) {
	op.Settlement = settlement
}

func (op ExtendedOperation) HashBytes() []byte {
	var bs []util.Byter
	bs = append(bs, op.Authentication)
	bs = append(bs, op.Settlement)
	return util.ConcatByters(bs...)
}

type BaseOperation struct {
	MBaseOperation
	Authentication
	Settlement
}

func NewBaseOperation(ht hint.Hint, fact base.Fact) BaseOperation {
	return BaseOperation{
		MBaseOperation: NewMBaseOperation(ht, fact),
	}
}

func (op BaseOperation) GetAuthentication() Authentication {
	return op.Authentication
}

func (op *BaseOperation) SetAuthentication(authentication Authentication) {
	op.Authentication = authentication
}

func (op BaseOperation) GetSettlement() Settlement {
	return op.Settlement
}

func (op *BaseOperation) SetSettlement(settlement Settlement) {
	op.Settlement = settlement
}

func (op *BaseOperation) Sign(priv base.Privatekey, networkID base.NetworkID) error {
	err := op.MBaseOperation.Sign(priv, networkID)
	if err != nil {
		return err
	}

	op.h = op.hash()

	return nil
}

func (op BaseOperation) hash() util.Hash {
	return valuehash.NewSHA256(op.HashBytes())
}

func (op BaseOperation) HashBytes() []byte {
	var bs [][]byte
	bs = append(bs, op.MBaseOperation.HashBytes())
	if op.Authentication != nil {
		bs = append(bs, op.Authentication.Bytes())
	}
	if op.Settlement != nil {
		bs = append(bs, op.Settlement.Bytes())
	}

	return util.ConcatBytesSlice(bs...)
}

func (op BaseOperation) String() string {
	var b []byte
	b, _ = json.Marshal(op)

	return fmt.Sprintf("%s", string(b))
}

func (op BaseOperation) IsValid(networkID []byte) error {
	if err := util.CheckIsValiders(networkID, false, op.MBaseOperation); err != nil {
		return ErrOperationInvalid.Wrap(err)
	}
	if err := util.CheckIsValiders(networkID, true, op.Authentication, op.Settlement); err != nil {
		return ErrOperationInvalid.Wrap(err)
	}

	if !op.h.Equal(op.hash()) {
		return ErrOperationInvalid.Wrap(ErrValueInvalid.Wrap(errors.Errorf("hash does not match")))
	}
	return nil
}

type MBaseOperation struct {
	h     util.Hash
	fact  base.Fact
	signs []base.Sign
	hint.BaseHinter
}

func NewMBaseOperation(ht hint.Hint, fact base.Fact) MBaseOperation {
	return MBaseOperation{
		BaseHinter: hint.NewBaseHinter(ht),
		fact:       fact,
	}
}

func (op MBaseOperation) Hash() util.Hash {
	return op.h
}

func (op *MBaseOperation) SetHash(h util.Hash) {
	op.h = h
}

func (op MBaseOperation) Signs() []base.Sign {
	return op.signs
}

func (op MBaseOperation) Fact() base.Fact {
	return op.fact
}

func (op *MBaseOperation) SetFact(fact base.Fact) {
	op.fact = fact
}

func (op MBaseOperation) HashBytes() []byte {
	bs := make([]util.Byter, len(op.signs)+1)
	bs[0] = op.fact.Hash()

	for i := range op.signs {
		bs[i+1] = op.signs[i]
	}

	return util.ConcatByters(bs...)
}

func (op MBaseOperation) IsValid(networkID []byte) error {
	if len(op.signs) < 1 {
		return ErrOperationInvalid.Wrap(ErrSignInvalid.Wrap(errors.Errorf("empty signs")))
	}

	if err := util.CheckIsValiders(networkID, false, op.h); err != nil {
		return ErrOperationInvalid.Wrap(err)
	}

	sfs := op.Signs()
	var duplicatederr error

	switch duplicated := util.IsDuplicatedSlice(sfs, func(i base.Sign) (bool, string) {
		if i == nil {
			return true, ""
		}

		s, ok := i.(base.Sign)
		if !ok {
			duplicatederr = ErrTypeMismatch.Wrap(errors.Errorf("expected Sign got %T", i))
		}

		return duplicatederr == nil, s.Signer().String()
	}); {
	case duplicatederr != nil:
		return ErrOperationInvalid.Wrap(ErrSignInvalid.Wrap(duplicatederr))
	case duplicated:
		return ErrOperationInvalid.Wrap(ErrSignInvalid.Wrap(errors.Errorf("duplicated signs found")))
	}

	if err := IsValidSignFact(op, networkID); err != nil {
		return ErrOperationInvalid.Wrap(err)
	}

	return nil
}

func (op *MBaseOperation) Sign(priv base.Privatekey, networkID base.NetworkID) error {
	switch index, sign, err := op.sign(priv, networkID); {
	case err != nil:
		return err
	case index < 0:
		op.signs = append(op.signs, sign)
	default:
		op.signs[index] = sign
	}

	op.h = op.hash()

	return nil
}

func (op *MBaseOperation) sign(priv base.Privatekey, networkID base.NetworkID) (found int, sign base.BaseSign, _ error) {
	e := util.StringError("sign BaseOperation")

	found = -1

	for i := range op.signs {
		s := op.signs[i]
		if s == nil {
			continue
		}

		if s.Signer().Equal(priv.Publickey()) {
			found = i

			break
		}
	}

	newsign, err := base.NewBaseSignFromFact(priv, networkID, op.fact)
	if err != nil {
		return found, sign, e.Wrap(err)
	}

	return found, newsign, nil
}

func (MBaseOperation) PreProcess(ctx context.Context, _ base.GetStateFunc) (
	context.Context, base.OperationProcessReasonError, error,
) {
	return ctx, nil, errors.WithStack(util.ErrNotImplemented)
}

func (MBaseOperation) Process(context.Context, base.GetStateFunc) ([]base.StateMergeValue, base.OperationProcessReasonError, error) {
	return nil, nil, errors.WithStack(util.ErrNotImplemented)
}

func (op MBaseOperation) hash() util.Hash {
	return valuehash.NewSHA256(op.HashBytes())
}

func IsValidOperationFact(fact base.Fact, networkID []byte) error {
	if err := util.CheckIsValiders(networkID, false,
		fact.Hash(),
	); err != nil {
		return err
	}

	switch l := len(fact.Token()); {
	case l < 1:
		return errors.Errorf("operation has empty token")
	case l > base.MaxTokenSize:
		return errors.Errorf("operation token size too large: %d > %d", l, base.MaxTokenSize)
	}

	hg, ok := fact.(HashGenerator)
	if !ok {
		return nil
	}

	if !fact.Hash().Equal(hg.GenerateHash()) {
		return ErrValueInvalid.Wrap(errors.Errorf("wrong Fact hash"))
	}

	return nil
}

type BaseNodeOperation struct {
	MBaseOperation
}

func NewBaseNodeOperation(ht hint.Hint, fact base.Fact) BaseNodeOperation {
	return BaseNodeOperation{
		MBaseOperation: NewMBaseOperation(ht, fact),
	}
}

func (op BaseNodeOperation) IsValid(networkID []byte) error {
	if err := op.MBaseOperation.IsValid(networkID); err != nil {
		return ErrNodeOperationInvalid.Wrap(err)
	}

	sfs := op.Signs()

	var duplicatederr error

	switch duplicated := util.IsDuplicatedSlice(sfs, func(i base.Sign) (bool, string) {
		if i == nil {
			return true, ""
		}

		ns, ok := i.(base.NodeSign)
		if !ok {
			duplicatederr = errors.Errorf("expected NodeSign got %T", i)
		}

		return duplicatederr == nil, ns.Node().String()
	}); {
	case duplicatederr != nil:
		return ErrNodeOperationInvalid.Wrap(duplicatederr)
	case duplicated:
		return ErrNodeOperationInvalid.Wrap(errors.Errorf("Duplicated signs found"))
	}

	for i := range sfs {
		if _, ok := sfs[i].(base.NodeSign); !ok {
			return ErrNodeOperationInvalid.Wrap(errors.Errorf("expected NodeSign got %T", sfs[i]))
		}
	}

	return nil
}

func (op *BaseNodeOperation) NodeSign(priv base.Privatekey, networkID base.NetworkID, node base.Address) error {
	found := -1

	for i := range op.signs {
		s := op.signs[i].(base.NodeSign) //nolint:forcetypeassert //...
		if s == nil {
			continue
		}

		if s.Node().Equal(node) {
			found = i

			break
		}
	}

	ns, err := base.NewBaseNodeSignFromFact(node, priv, networkID, op.fact)
	if err != nil {
		return err
	}

	switch {
	case found < 0:
		op.signs = append(op.signs, ns)
	default:
		op.signs[found] = ns
	}

	op.h = op.hash()

	return nil
}

func (op *BaseNodeOperation) SetNodeSigns(signs []base.NodeSign) error {
	if duplicated := util.IsDuplicatedSlice(signs, func(i base.NodeSign) (bool, string) {
		if i == nil {
			return true, ""
		}

		return true, i.Node().String()
	}); duplicated {
		return errors.Errorf("Duplicated signs found")
	}

	op.signs = make([]base.Sign, len(signs))
	for i := range signs {
		op.signs[i] = signs[i]
	}

	op.h = op.hash()

	return nil
}

func (op *BaseNodeOperation) AddNodeSigns(signs []base.NodeSign) (added bool, _ error) {
	updates := util.FilterSlice(signs, func(sign base.NodeSign) bool {
		return slices.IndexFunc(op.signs, func(s base.Sign) bool {
			nodesign, ok := s.(base.NodeSign)
			if !ok {
				return false
			}

			return sign.Node().Equal(nodesign.Node())
		}) < 0
	})

	if len(updates) < 1 {
		return false, nil
	}

	mergedsigns := make([]base.Sign, len(op.signs)+len(updates))
	copy(mergedsigns, op.signs)

	for i := range updates {
		mergedsigns[len(op.signs)+i] = updates[i]
	}

	op.signs = mergedsigns
	op.h = op.hash()

	return true, nil
}

func (op BaseNodeOperation) NodeSigns() []base.NodeSign {
	ss := op.Signs()
	signs := make([]base.NodeSign, len(ss))

	for i := range ss {
		signs[i] = ss[i].(base.NodeSign) //nolint:forcetypeassert //...
	}

	return signs
}

type HashGenerator interface {
	GenerateHash() util.Hash
}
