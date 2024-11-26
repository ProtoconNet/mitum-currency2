package common

//type ExtendedFactJSONMarshaler struct {
//	User base.Address `json:"user"`
//	base.Fact
//}
//
//func (fact ExtendedFact) JSONMarshaler() ExtendedFactJSONMarshaler {
//	return ExtendedFactJSONMarshaler{
//		Fact: fact.Fact,
//		User: fact.user,
//	}
//}
//
//func (fact ExtendedFact) MarshalJSON() ([]byte, error) {
//	return util.MarshalJSON(fact.JSONMarshaler())
//}
//
//type ExtendedFactJSONUnmarshaler struct {
//	Fact base.Fact
//	User string `json:"user"`
//}
//
//func (fact *ExtendedFact) DecodeJSON(b []byte, enc encoder.Encoder) error {
//	e := util.StringError("failed to decode json of ExtendedFact")
//
//	var u ExtendedFactJSONUnmarshaler
//	if err := enc.Unmarshal(b, &u); err != nil {
//		return e.Wrap(err)
//	}
//
//	var uf ExtendedFactJSONUnmarshaler
//	if err := json.Unmarshal(b, &uf); err != nil {
//		return e.Wrap(err)
//	}
//
//	switch ad, err := base.DecodeAddress(uf.User, enc); {
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
