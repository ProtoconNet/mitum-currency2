package common

import (
	"encoding/json"
	"golang.org/x/exp/slices"
	"sort"
	"strings"
	"sync"

	"github.com/ProtoconNet/mitum2/base"
	"github.com/ProtoconNet/mitum2/util"
	"github.com/ProtoconNet/mitum2/util/encoder"
	"github.com/ProtoconNet/mitum2/util/hint"
	"github.com/ProtoconNet/mitum2/util/valuehash"
	"github.com/pkg/errors"
)

var stateValueMergerPool = sync.Pool{
	New: func() any {
		return &BaseStateValueMerger{}
	},
}

var BaseStateHint = hint.MustNewHint("currency-base-state-v0.0.1")

type BaseState struct {
	h        util.Hash
	previous util.Hash
	v        base.StateValue
	k        string
	ops      []util.Hash
	hint.BaseHinter
	height base.Height
}

func NewBaseState(
	height base.Height,
	k string,
	v base.StateValue,
	previous util.Hash,
	ops []util.Hash,
) BaseState {
	s := BaseState{
		BaseHinter: hint.NewBaseHinter(BaseStateHint),
		height:     height,
		k:          k,
		v:          v,
		previous:   previous,
		ops:        ops,
	}

	s.h = s.generateHash()

	return s
}

func (s BaseState) IsValid([]byte) error {
	vs := make([]util.IsValider, len(s.ops)+5)
	vs[0] = s.BaseHinter
	vs[1] = s.h
	vs[2] = s.height
	vs[3] = util.DummyIsValider(func([]byte) error {
		if len(s.k) < 1 {
			return ErrStateInvalid.Wrap(errors.Errorf("Empty state key"))
		}

		return nil
	})
	vs[4] = s.v

	for i := range s.ops {
		vs[i+5] = s.ops[i]
	}

	if err := util.CheckIsValiders(nil, false, vs...); err != nil {
		return ErrStateInvalid.Wrap(err)
	}

	if s.previous != nil {
		if err := s.previous.IsValid(nil); err != nil {
			return ErrStateInvalid.Wrap(errors.Errorf("invalid previous state hash, %v", err))
		}
	}

	if !s.h.Equal(s.generateHash()) {
		return ErrStateInvalid.Wrap(errors.Errorf("Wrong hash"))
	}

	return nil
}

func (s BaseState) Hash() util.Hash {
	return s.h
}

func (s BaseState) Previous() util.Hash {
	return s.previous
}

func (s BaseState) Key() string {
	return s.k
}

func (s BaseState) Value() base.StateValue {
	return s.v
}

func (s BaseState) Height() base.Height {
	return s.height
}

func (s BaseState) Operations() []util.Hash {
	return s.ops
}

func (s BaseState) generateHash() util.Hash {
	return valuehash.NewSHA256(util.ConcatByters(
		util.DummyByter(func() []byte {
			if s.previous == nil {
				return nil
			}

			return s.previous.Bytes()
		}),
		util.BytesToByter([]byte(s.k)),
		util.DummyByter(func() []byte {
			if s.v == nil {
				return nil
			}

			return s.v.HashBytes()
		}),
		util.DummyByter(func() []byte {
			if len(s.ops) < 1 {
				return nil
			}

			bs := make([][]byte, len(s.ops))

			for i := range s.ops {
				if s.ops[i] == nil {
					continue
				}

				bs[i] = s.ops[i].Bytes()
			}

			return util.ConcatBytesSlice(bs...)
		}),
	))
}

type baseStateJSONMarshaler struct {
	Hash       util.Hash       `json:"hash"`
	Previous   util.Hash       `json:"previous"`
	Value      base.StateValue `json:"value"`
	Key        string          `json:"key"`
	Operations []util.Hash     `json:"operations"`
	hint.BaseHinter
	Height base.Height `json:"height"`
}

func (s BaseState) MarshalJSON() ([]byte, error) {
	return util.MarshalJSON(baseStateJSONMarshaler{
		BaseHinter: s.BaseHinter,
		Hash:       s.h,
		Previous:   s.previous,
		Height:     s.height,
		Key:        s.k,
		Value:      s.v,
		Operations: s.ops,
	})
}

type baseStateJSONUnmarshaler struct {
	Hash       valuehash.HashDecoder   `json:"hash"`
	Previous   valuehash.HashDecoder   `json:"previous"`
	Key        string                  `json:"key"`
	Value      json.RawMessage         `json:"value"`
	Operations []valuehash.HashDecoder `json:"operations"`
	Height     base.HeightDecoder      `json:"height"`
}

func (s *BaseState) DecodeJSON(b []byte, enc encoder.Encoder) error {
	e := util.StringError("Decode json BaseState")

	var u baseStateJSONUnmarshaler
	if err := enc.Unmarshal(b, &u); err != nil {
		return e.Wrap(err)
	}

	s.h = u.Hash.Hash()
	s.previous = u.Previous.Hash()
	s.height = u.Height.Height()
	s.k = u.Key

	s.ops = make([]util.Hash, len(u.Operations))

	for i := range u.Operations {
		s.ops[i] = u.Operations[i].Hash()
	}

	switch i, err := DecodeStateValue(u.Value, enc); {
	case err != nil:
		return e.Wrap(err)
	default:
		s.v = i
	}

	return nil
}

type BaseStateValueMerger struct {
	st     base.State
	value  base.StateValue
	key    string
	ops    []util.Hash
	height base.Height
	sync.RWMutex
}

func NewBaseStateValueMerger(height base.Height, key string, st base.State) *BaseStateValueMerger {
	s := stateValueMergerPool.Get().(*BaseStateValueMerger) //nolint:forcetypeassert //...

	s.Reset(height, key, st)

	return s
}

func (s *BaseStateValueMerger) Reset(height base.Height, key string, st base.State) {
	nkey := key

	if st != nil {
		nkey = st.Key()
	}

	s.st = st
	s.height = height
	s.key = nkey
	s.value = nil
	s.ops = nil
}

func (s *BaseStateValueMerger) Close() error {
	s.Lock()
	defer s.Unlock()

	s.height = 0
	s.st = nil
	s.key = ""
	s.value = nil
	s.ops = nil

	stateValueMergerPool.Put(s)

	return nil
}

func (s *BaseStateValueMerger) Key() string {
	return s.key
}

func (s *BaseStateValueMerger) Merge(value base.StateValue, op util.Hash) error {
	s.Lock()
	defer s.Unlock()

	s.value = value

	s.addOperation(op)

	return nil
}

func (s *BaseStateValueMerger) CloseValue() (base.State, error) {
	s.Lock()
	defer s.Unlock()

	if s.value == nil || len(s.ops) < 1 {
		return nil, base.ErrIgnoreStateValue.Errorf("Empty state value")
	}

	sort.Slice(s.ops, func(i, j int) bool {
		return strings.Compare(s.ops[i].String(), s.ops[j].String()) < 0
	})

	var previous util.Hash
	if s.st != nil {
		previous = s.st.Hash()
	}

	return NewBaseState(s.height, s.key, s.value, previous, s.ops), nil
}

func (s *BaseStateValueMerger) AddOperation(op util.Hash) {
	s.Lock()
	defer s.Unlock()

	s.addOperation(op)
}

func (s *BaseStateValueMerger) Height() base.Height {
	return s.height
}

func (s *BaseStateValueMerger) State() base.State {
	return s.st
}

func (s *BaseStateValueMerger) SetValue(v base.StateValue) {
	s.Lock()
	defer s.Unlock()

	s.value = v
}

func (s *BaseStateValueMerger) addOperation(op util.Hash) {
	if slices.IndexFunc(s.ops, func(i util.Hash) bool {
		return i.Equal(op)
	}) >= 0 {
		return
	}

	nops := make([]util.Hash, len(s.ops)+1)
	copy(nops[:len(s.ops)], s.ops)
	nops[len(s.ops)] = op

	s.ops = nops
}

type BaseStateMergeValue struct {
	base.StateValue
	merger func(base.Height, base.State) base.StateValueMerger
	key    string
}

func NewBaseStateMergeValue(
	key string,
	value base.StateValue,
	merger func(base.Height, base.State) base.StateValueMerger,
) BaseStateMergeValue {
	v := BaseStateMergeValue{StateValue: value, key: key, merger: merger}

	if merger == nil {
		v.merger = v.defaultMerger
	}

	return v
}

func (v BaseStateMergeValue) Key() string {
	return v.key
}

func (v BaseStateMergeValue) Value() base.StateValue {
	return v.StateValue
}

func (v BaseStateMergeValue) Merger(height base.Height, st base.State) base.StateValueMerger {
	return v.merger(height, st)
}

func (v BaseStateMergeValue) defaultMerger(height base.Height, st base.State) base.StateValueMerger {
	nst := st
	if st == nil {
		nst = NewBaseState(base.NilHeight, v.key, nil, nil, nil)
	}

	return NewBaseStateValueMerger(height, nst.Key(), nst)
}

func DecodeStateValue(b []byte, enc encoder.Encoder) (base.StateValue, error) {
	e := util.StringError("Decode StateValue")

	var s base.StateValue
	if err := encoder.Decode(enc, b, &s); err != nil {
		return nil, e.Wrap(err)
	}

	return s, nil
}
