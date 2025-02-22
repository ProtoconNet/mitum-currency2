package isaacoperation

import (
	"context"
	"github.com/ProtoconNet/mitum-currency/v3/common"
	"github.com/ProtoconNet/mitum-currency/v3/types"
	"github.com/ProtoconNet/mitum2/base"
	"github.com/ProtoconNet/mitum2/isaac"
	"github.com/ProtoconNet/mitum2/util"
	"github.com/ProtoconNet/mitum2/util/hint"
	"github.com/ProtoconNet/mitum2/util/localtime"
	"github.com/ProtoconNet/mitum2/util/valuehash"
)

var (
	GenesisNetworkPolicyFactHint = hint.MustNewHint("currency-genesis-network-policy-fact-v0.0.1")
	GenesisNetworkPolicyHint     = hint.MustNewHint("currency-genesis-network-policy-v0.0.1")
	NetworkPolicyFactHint        = hint.MustNewHint("currency-network-policy-fact-v0.0.1")
	NetworkPolicyHint            = hint.MustNewHint("currency-network-policy-operation-v0.0.1")
)

type GenesisNetworkPolicyFact struct {
	policy base.NetworkPolicy
	base.BaseFact
}

func NewGenesisNetworkPolicyFact(policy base.NetworkPolicy) GenesisNetworkPolicyFact {
	fact := GenesisNetworkPolicyFact{
		BaseFact: base.NewBaseFact(
			GenesisNetworkPolicyFactHint,
			base.Token(localtime.New(localtime.Now().UTC()).Bytes()),
		),
		policy: policy,
	}

	fact.SetHash(fact.hash())

	return fact
}

func (fact GenesisNetworkPolicyFact) IsValid([]byte) error {
	e := util.ErrInvalid.Errorf("invalid GenesisNetworkPolicyFact")

	if err := util.CheckIsValiders(nil, false, fact.BaseFact, fact.policy); err != nil {
		return e.Wrap(err)
	}

	if !fact.Hash().Equal(fact.hash()) {
		return e.Errorf("hash does not match")
	}

	return nil
}

func (fact GenesisNetworkPolicyFact) Policy() base.NetworkPolicy {
	return fact.policy
}

func (fact GenesisNetworkPolicyFact) hash() util.Hash {
	return valuehash.NewSHA256(util.ConcatByters(
		util.BytesToByter(fact.Token()),
		util.DummyByter(fact.policy.HashBytes),
	))
}

// GenesisNetworkPolicy is only for used for genesis block
type GenesisNetworkPolicy struct {
	common.BaseOperation
}

func NewGenesisNetworkPolicy(fact GenesisNetworkPolicyFact) GenesisNetworkPolicy {
	return GenesisNetworkPolicy{
		BaseOperation: common.NewBaseOperation(GenesisNetworkPolicyHint, fact),
	}
}

func (op GenesisNetworkPolicy) IsValid(networkID []byte) error {
	e := util.ErrInvalid.Errorf("invalid GenesisNetworkPolicy")

	if err := op.BaseOperation.IsValid(networkID); err != nil {
		return e.Wrap(err)
	}

	if len(op.Signs()) > 1 {
		return e.Errorf("multiple signs found")
	}

	if _, ok := op.Fact().(GenesisNetworkPolicyFact); !ok {
		return e.Errorf("not GenesisNetworkPolicyFact, %T", op.Fact())
	}

	return nil
}

func (GenesisNetworkPolicy) PreProcess(ctx context.Context, getStateFunc base.GetStateFunc) (
	context.Context, base.OperationProcessReasonError, error,
) {
	switch _, found, err := getStateFunc(isaac.NetworkPolicyStateKey); {
	case err != nil:
		return ctx, nil, err
	case found:
		return ctx, base.NewBaseOperationProcessReasonError("network policy state already exists"), nil
	default:
		return ctx, nil, nil
	}
}

func (op GenesisNetworkPolicy) Process(context.Context, base.GetStateFunc) (
	[]base.StateMergeValue, base.OperationProcessReasonError, error,
) {
	fact := op.Fact().(GenesisNetworkPolicyFact) //nolint:forcetypeassert //...

	return []base.StateMergeValue{
		common.NewBaseStateMergeValue(
			isaac.NetworkPolicyStateKey,
			types.NewNetworkPolicyStateValue(fact.Policy()),
			nil,
		),
	}, nil, nil
}

type NetworkPolicyFact struct {
	policy base.NetworkPolicy
	base.BaseFact
}

func NewNetworkPolicyFact(token base.Token, policy base.NetworkPolicy) NetworkPolicyFact {
	fact := NetworkPolicyFact{
		BaseFact: base.NewBaseFact(
			NetworkPolicyFactHint,
			token,
		),
		policy: policy,
	}

	fact.SetHash(fact.hash())

	return fact
}

func (fact NetworkPolicyFact) hash() util.Hash {
	return valuehash.NewSHA256(util.ConcatByters(
		util.BytesToByter(fact.Token()),
		util.DummyByter(fact.policy.HashBytes),
	))
}

func (fact NetworkPolicyFact) IsValid([]byte) error {
	e := util.ErrInvalid.Errorf("Invalid NetworkPolicyFact")

	if err := util.CheckIsValiders(nil, false, fact.BaseFact, fact.policy); err != nil {
		return e.Wrap(err)
	}

	if !fact.Hash().Equal(fact.hash()) {
		return e.Errorf("hash does not match")
	}

	return nil
}

func (fact NetworkPolicyFact) Policy() base.NetworkPolicy {
	return fact.policy
}

type NetworkPolicy struct {
	common.BaseNodeOperation
}

func NewNetworkPolicy(fact NetworkPolicyFact) NetworkPolicy {
	return NetworkPolicy{
		BaseNodeOperation: common.NewBaseNodeOperation(NetworkPolicyHint, fact),
	}
}

func (op NetworkPolicy) IsValid(networkID []byte) error {
	e := util.ErrInvalid.Errorf("Invalid NetworkPolicy")

	if err := op.BaseNodeOperation.IsValid(networkID); err != nil {
		return e.Wrap(err)
	}

	if _, ok := op.Fact().(NetworkPolicyFact); !ok {
		return e.Errorf("not NetworkPolicyFact, %T", op.Fact())
	}

	return nil
}

func (op *NetworkPolicy) SetToken(t base.Token) error {
	fact := op.Fact().(NetworkPolicyFact) //nolint:forcetypeassert //...

	if err := fact.SetToken(t); err != nil {
		return err
	}

	fact.SetHash(fact.hash())

	op.BaseNodeOperation.SetFact(fact)

	return nil
}
