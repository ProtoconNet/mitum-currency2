package digest

import (
	"github.com/ProtoconNet/mitum-currency/v3/types"
	"github.com/ProtoconNet/mitum2/base"
	"github.com/pkg/errors"
	"net/http"
	"time"
)

var (
	HandlerPathDIDDesign   = `/did-registry/{contract:(?i)` + types.REStringAddressString + `}`
	HandlerPathDIDData     = `/did-registry/{contract:(?i)` + types.REStringAddressString + `}/did/{method_specific_id:` + types.ReSpecialCh + `}`
	HandlerPathDIDDocument = `/did-registry/{contract:(?i)` + types.REStringAddressString + `}/document`
)

func (hd *Handlers) handleDIDDesign(w http.ResponseWriter, r *http.Request) {
	cacheKey := CacheKeyPath(r)
	if err := LoadFromCache(hd.cache, cacheKey, w); err == nil {
		return
	}

	contract, err, status := ParseRequest(w, r, "contract")
	if err != nil {
		HTTP2ProblemWithError(w, err, status)
		return
	}

	if v, err, shared := hd.rg.Do(cacheKey, func() (interface{}, error) {
		return hd.handleDIDDesignInGroup(contract)
	}); err != nil {
		HTTP2HandleError(w, err)
	} else {
		HTTP2WriteHalBytes(hd.enc, w, v.([]byte), http.StatusOK)

		if !shared {
			HTTP2WriteCache(w, cacheKey, time.Hour*3000)
		}
	}
}

func (hd *Handlers) handleDIDDesignInGroup(contract string) ([]byte, error) {
	var de types.Design
	var st base.State

	de, st, err := DIDDesign(hd.database, contract)
	if err != nil {
		return nil, err
	}

	i, err := hd.buildDIDDesign(contract, de, st)
	if err != nil {
		return nil, err
	}
	return hd.enc.Marshal(i)
}

func (hd *Handlers) buildDIDDesign(contract string, de types.Design, st base.State) (Hal, error) {
	h, err := hd.combineURL(HandlerPathDIDDesign, "contract", contract)
	if err != nil {
		return nil, err
	}

	var hal Hal
	hal = NewBaseHal(de, NewHalLink(h, nil))

	h, err = hd.combineURL(HandlerPathBlockByHeight, "height", st.Height().String())
	if err != nil {
		return nil, err
	}
	hal = hal.AddLink("block", NewHalLink(h, nil))

	for i := range st.Operations() {
		h, err := hd.combineURL(HandlerPathOperation, "hash", st.Operations()[i].String())
		if err != nil {
			return nil, err
		}
		hal = hal.AddLink("operations", NewHalLink(h, nil))
	}

	return hal, nil
}

func (hd *Handlers) handleDIDData(w http.ResponseWriter, r *http.Request) {
	cacheKey := CacheKeyPath(r)
	if err := LoadFromCache(hd.cache, cacheKey, w); err == nil {
		return
	}

	contract, err, status := ParseRequest(w, r, "contract")
	if err != nil {
		HTTP2ProblemWithError(w, err, status)
		return
	}

	key, err, status := ParseRequest(w, r, "method_specific_id")
	if err != nil {
		HTTP2ProblemWithError(w, err, status)
		return
	}

	if v, err, shared := hd.rg.Do(cacheKey, func() (interface{}, error) {
		return hd.handleDIDDataInGroup(contract, key)
	}); err != nil {
		HTTP2HandleError(w, err)
	} else {
		HTTP2WriteHalBytes(hd.enc, w, v.([]byte), http.StatusOK)

		if !shared {
			HTTP2WriteCache(w, cacheKey, time.Hour*3000)
		}
	}
}

func (hd *Handlers) handleDIDDataInGroup(contract, key string) ([]byte, error) {
	data, st, err := DIDData(hd.database, contract, key)
	if err != nil {
		return nil, err
	}

	i, err := hd.buildDIDDataHal(contract, *data, st)
	if err != nil {
		return nil, err
	}
	return hd.enc.Marshal(i)
}

func (hd *Handlers) buildDIDDataHal(
	contract string, data types.Data, st base.State) (Hal, error) {
	h, err := hd.combineURL(
		HandlerPathDIDData,
		"contract", contract, "method_specific_id", data.Address().String())
	if err != nil {
		return nil, err
	}

	var hal Hal
	hal = NewBaseHal(data, NewHalLink(h, nil))
	h, err = hd.combineURL(HandlerPathBlockByHeight, "height", st.Height().String())
	if err != nil {
		return nil, err
	}
	hal = hal.AddLink("block", NewHalLink(h, nil))

	for i := range st.Operations() {
		h, err := hd.combineURL(HandlerPathOperation, "hash", st.Operations()[i].String())
		if err != nil {
			return nil, err
		}
		hal = hal.AddLink("operations", NewHalLink(h, nil))
	}

	return hal, nil
}

func (hd *Handlers) handleDIDDocument(w http.ResponseWriter, r *http.Request) {
	cacheKey := CacheKeyPath(r)
	if err := LoadFromCache(hd.cache, cacheKey, w); err == nil {
		return
	}

	contract, err, status := ParseRequest(w, r, "contract")
	if err != nil {
		HTTP2ProblemWithError(w, err, status)
		return
	}

	did := ParseStringQuery(r.URL.Query().Get("did"))
	if len(did) < 1 {
		HTTP2ProblemWithError(w, errors.Errorf("invalid DID"), http.StatusBadRequest)
		return
	}

	if v, err, shared := hd.rg.Do(cacheKey, func() (interface{}, error) {
		return hd.handleDIDDocumentInGroup(contract, did)
	}); err != nil {
		HTTP2HandleError(w, err)
	} else {
		HTTP2WriteHalBytes(hd.enc, w, v.([]byte), http.StatusOK)

		if !shared {
			HTTP2WriteCache(w, cacheKey, time.Millisecond*100)
		}
	}
}

func (hd *Handlers) handleDIDDocumentInGroup(contract, key string) ([]byte, error) {
	doc, st, err := DIDDocument(hd.database, contract, key)
	if err != nil {
		return nil, err
	}

	i, err := hd.buildDIDDocumentHal(contract, *doc, st)
	if err != nil {
		return nil, err
	}
	return hd.enc.Marshal(i)
}

func (hd *Handlers) buildDIDDocumentHal(
	contract string, doc types.DIDDocument, st base.State) (Hal, error) {
	//h, err := hd.combineURL(
	//	HandlerPathDIDDocument,
	//	"contract", contract)
	//if err != nil {
	//	return nil, err
	//}

	var hal Hal
	hal = NewBaseHal(doc, NewHalLink("", nil))
	h, err := hd.combineURL(HandlerPathBlockByHeight, "height", st.Height().String())
	if err != nil {
		return nil, err
	}
	hal = hal.AddLink("block", NewHalLink(h, nil))

	for i := range st.Operations() {
		h, err := hd.combineURL(HandlerPathOperation, "hash", st.Operations()[i].String())
		if err != nil {
			return nil, err
		}
		hal = hal.AddLink("operations", NewHalLink(h, nil))
	}

	return hal, nil
}
