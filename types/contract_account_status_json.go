package types

import (
	"github.com/ProtoconNet/mitum2/base"
	"github.com/ProtoconNet/mitum2/util"
	"github.com/ProtoconNet/mitum2/util/encoder"
	"github.com/ProtoconNet/mitum2/util/hint"
)

type ContractAccountStatusJSONMarshaler struct {
	hint.BaseHinter
	Owner      base.Address   `json:"owner"`
	IsActive   bool           `json:"is_active"`
	Handlers   []base.Address `json:"handlers"`
	Recipients []base.Address `json:"recipients"`
}

func (cs ContractAccountStatus) MarshalJSON() ([]byte, error) {
	return util.MarshalJSON(ContractAccountStatusJSONMarshaler{
		BaseHinter: cs.BaseHinter,
		Owner:      cs.owner,
		IsActive:   cs.isActive,
		Handlers:   cs.handlers,
		Recipients: cs.recipients,
	})
}

type ContractAccountStatusJSONUnmarshaler struct {
	Hint       hint.Hint `json:"_hint"`
	Owner      string    `json:"owner"`
	IsActive   bool      `json:"is_active"`
	Handlers   []string  `json:"handlers"`
	Recipients []string  `json:"recipients"`
}

func (cs *ContractAccountStatus) DecodeJSON(b []byte, enc encoder.Encoder) error {
	e := util.StringError("Decode json of ContractAccountStatus")

	var ucs ContractAccountStatusJSONUnmarshaler
	if err := enc.Unmarshal(b, &ucs); err != nil {
		return e.Wrap(err)
	}

	return cs.unpack(enc, ucs.Hint, ucs.Owner, ucs.IsActive, ucs.Handlers, ucs.Recipients)
}
