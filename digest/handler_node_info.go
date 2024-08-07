package digest

import (
	"context"
	isaacnetwork "github.com/ProtoconNet/mitum2/isaac/network"
	"github.com/ProtoconNet/mitum2/launch"
	"github.com/ProtoconNet/mitum2/network/quicmemberlist"
	"github.com/ProtoconNet/mitum2/network/quicstream"
	quicstreamheader "github.com/ProtoconNet/mitum2/network/quicstream/header"
	"github.com/ProtoconNet/mitum2/util/encoder"
	"github.com/pkg/errors"
	"net/http"
	"time"
)

func (hd *Handlers) SetNodeInfoHandler(handler NodeInfoHandler) *Handlers {
	hd.nodeInfoHandler = handler

	return hd
}

func (hd *Handlers) handleNodeInfo(w http.ResponseWriter, r *http.Request) {

	//if hd.nodeInfoHandler == nil {
	//	HTTP2NotSupported(w, nil)
	//
	//	return
	//}

	cachekey := CacheKeyPath(r)
	if err := LoadFromCache(hd.cache, cachekey, w); err == nil {
		return
	}

	if v, err, shared := hd.rg.Do(cachekey, hd.handleNodeInfoInGroup); err != nil {
		hd.Log().Err(err).Msg("get node info")

		HTTP2HandleError(w, err)
	} else {
		HTTP2WriteHalBytes(hd.enc, w, v.([]byte), http.StatusOK)

		if !shared {
			HTTP2WriteCache(w, cachekey, time.Millisecond*100)
		}
	}
}

func (hd *Handlers) handleNodeInfoInGroup() (interface{}, error) {
	params, memberList, nodeList, err := hd.client()
	connectionPool, err := launch.NewConnectionPool(
		1<<9,
		params.ISAAC.NetworkID(),
		nil,
	)
	client := isaacnetwork.NewBaseClient( //nolint:gomnd //...
		hd.encs, hd.enc,
		connectionPool.Dial,
		connectionPool.CloseAll,
	)
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = client.Close()
	}()

	var nodeInfoList []isaacnetwork.NodeInfo
	switch {
	case err != nil:
		return nil, err

	default:
		connInfo := make(map[string]quicstream.ConnInfo)
		memberList.Members(func(node quicmemberlist.Member) bool {
			connInfo[node.ConnInfo().String()] = node.ConnInfo()
			return true
		})
		for _, c := range nodeList {
			connInfo[c.String()] = c
		}

		for i := range connInfo {
			nodeInfo, err := NodeInfo(client, connInfo[i])

			if err != nil {
				continue
			}

			nodeInfoList = append(nodeInfoList, *nodeInfo)
		}
	}

	if i, err := hd.buildNodeInfoHal(nodeInfoList); err != nil {
		return nil, err
	} else {
		return hd.enc.Marshal(i)
	}
}

func (hd *Handlers) buildNodeInfoHal(ni []isaacnetwork.NodeInfo) (Hal, error) {
	var hal Hal = NewBaseHal(ni, NewHalLink(HandlerPathNodeInfo, nil))

	return hal, nil
}

func NodeInfo(client *isaacnetwork.BaseClient, connInfo quicstream.ConnInfo) (*isaacnetwork.NodeInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	stream, _, err := client.Dial(ctx, connInfo)
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = client.Close()
	}()

	header := isaacnetwork.NewNodeInfoRequestHeader()

	var nodeInfo *isaacnetwork.NodeInfo
	err = stream(ctx, func(ctx context.Context, broker *quicstreamheader.ClientBroker) error {
		if err := broker.WriteRequestHead(ctx, header); err != nil {
			return err
		}

		var enc encoder.Encoder

		switch rEnc, rh, err := broker.ReadResponseHead(ctx); {
		case err != nil:
			return err
		case !rh.OK():
			return errors.Errorf("Not ok")
		case rh.Err() != nil:
			return rh.Err()
		default:
			enc = rEnc
		}

		switch bodyType, bodyLength, r, err := broker.ReadBodyErr(ctx); {
		case err != nil:
			return err
		case bodyType == quicstreamheader.EmptyBodyType,
			bodyType == quicstreamheader.FixedLengthBodyType && bodyLength < 1:
			return errors.Errorf("Empty body")
		default:
			var v interface{}
			if err := enc.StreamDecoder(r).Decode(&v); err != nil {
				return err
			}

			b, err := enc.Marshal(v)
			if err != nil {
				return err
			}

			h, err := enc.Decode(b)
			if err != nil {
			}

			ni, ok := h.(isaacnetwork.NodeInfo)
			if !ok {
				return errors.Errorf("expected isaacnetwork.NodeInfo, not %T", v)
			}

			nodeInfo = &ni

			return nil
		}
	})
	if err != nil {
		return nil, err
	}

	return nodeInfo, nil
}
