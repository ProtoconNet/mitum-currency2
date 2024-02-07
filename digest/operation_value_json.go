package digest

import (
	"encoding/json"
	"time"

	"github.com/ProtoconNet/mitum2/base"
	"github.com/ProtoconNet/mitum2/util"
	"github.com/ProtoconNet/mitum2/util/encoder"
	"github.com/ProtoconNet/mitum2/util/hint"
)

type OperationValueJSONMarshaler struct {
	hint.BaseHinter
	Hash        util.Hash      `json:"hash"`
	Operation   base.Operation `json:"operation"`
	Height      base.Height    `json:"height"`
	ConfirmedAt time.Time      `json:"confirmed_at"`
	Reason      string         `json:"reason"`
	InState     bool           `json:"in_state"`
	Index       uint64         `json:"index"`
	DigestedAt  time.Time      `json:"digested_at"`
}

func (va OperationValue) MarshalJSON() ([]byte, error) {
	return util.MarshalJSON(OperationValueJSONMarshaler{
		BaseHinter:  va.BaseHinter,
		Hash:        va.op.Fact().Hash(),
		Operation:   va.op,
		Height:      va.height,
		ConfirmedAt: va.confirmedAt,
		Reason:      va.reason,
		InState:     va.inState,
		Index:       va.index,
		DigestedAt:  va.digestedAt,
	})
}

type OperationValueJSONUnmarshaler struct {
	Operation   json.RawMessage `json:"operation"`
	Height      base.Height     `json:"height"`
	ConfirmedAt time.Time       `json:"confirmed_at"`
	InState     bool            `json:"in_state"`
	Reason      string          `json:"reason"`
	Index       uint64          `json:"index"`
	DigestedAt  time.Time       `json:"digested_at"`
}

func (va *OperationValue) DecodeJSON(b []byte, enc encoder.Encoder) error {
	var uva OperationValueJSONUnmarshaler
	if err := enc.Unmarshal(b, &uva); err != nil {
		return err
	}

	if err := enc.Unmarshal(uva.Operation, &va.op); err != nil {
		return err
	}

	va.reason = uva.Reason
	va.height = uva.Height
	va.confirmedAt = uva.ConfirmedAt
	va.inState = uva.InState
	va.index = uva.Index
	va.digestedAt = uva.DigestedAt

	return nil
}
