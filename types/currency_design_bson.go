package types

import (
	"go.mongodb.org/mongo-driver/bson"

	bsonenc "github.com/ProtoconNet/mitum-currency/v3/digest/util/bson"
	"github.com/ProtoconNet/mitum2/util"
	"github.com/ProtoconNet/mitum2/util/hint"
)

func (de CurrencyDesign) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(
		bson.M{
			"_hint":           de.Hint().String(),
			"initial_supply":  de.initialSupply,
			"genesis_account": de.genesisAccount,
			"policy":          de.policy,
			"total_supply":    de.totalSupply.String(),
		},
	)
}

type CurrencyDesignBSONUnmarshaler struct {
	Hint          string   `bson:"_hint"`
	InitialSupply bson.Raw `bson:"initial_supply"`
	Genesis       string   `bson:"genesis_account"`
	Policy        bson.Raw `bson:"policy"`
	TotalSupply   string   `bson:"total_supply"`
}

func (de *CurrencyDesign) DecodeBSON(b []byte, enc *bsonenc.Encoder) error {
	e := util.StringError("Decode bson of CurrencyDesign")

	var ude CurrencyDesignBSONUnmarshaler
	if err := enc.Unmarshal(b, &ude); err != nil {
		return e.Wrap(err)
	}

	ht, err := hint.ParseHint(ude.Hint)
	if err != nil {
		return e.Wrap(err)
	}

	return de.unpack(enc, ht, ude.InitialSupply, ude.Genesis, ude.Policy, ude.TotalSupply)
}
