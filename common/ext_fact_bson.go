package common

//type ExtendedFactBSONMarshaler struct {
//	base.Fact `bson:",inline"`
//	User      string `bson:"user"`
//}
//
//func (fact ExtendedFact) MarshalBSON() ([]byte, error) {
//	return bsonenc.Marshal(
//		ExtendedFactBSONMarshaler{
//			Fact: fact.Fact,
//			User: fact.user.String(),
//		},
//	)
//}
//
//type ExtendedFactBSONUnmarshaler struct {
//	base.Fact `bson:",inline"`
//	User      string `bson:"user"`
//}
//
//func (fact *ExtendedFact) DecodeBSON(b []byte, enc *bsonenc.Encoder) error {
//	var u ExtendedFactBSONUnmarshaler
//
//	err := enc.Unmarshal(b, &u)
//	if err != nil {
//		return DecorateError(err, ErrDecodeBson, *fact)
//	}
//
//	switch ad, err := base.DecodeAddress(u.User, enc); {
//	case err != nil:
//		return err
//	default:
//		fact.user = ad
//	}
//
//	fact.Fact = u.Fact
//
//	return nil
//}
