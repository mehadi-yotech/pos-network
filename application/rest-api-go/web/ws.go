package web

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-protos-go/common"
	"golang.org/x/net/websocket"
)

type wsMessage struct {
	Type      string `json:"type"`
	Height    uint64 `json:"height"`
	Timestamp string `json:"timestamp"`
}

func (setup OrgSetup) ServeWS(w http.ResponseWriter, r *http.Request) {
	server := websocket.Server{
		Handshake: func(cfg *websocket.Config, req *http.Request) error {
			return nil
		},
		Handler: func(conn *websocket.Conn) {
			defer conn.Close()

			lastHeight, err := setup.getChainHeight()
			if err == nil {
				sendWS(conn, wsMessage{
					Type:      "chain_update",
					Height:    lastHeight,
					Timestamp: time.Now().UTC().Format(time.RFC3339),
				})
			}

			done := make(chan struct{})
			go func() {
				var noop string
				for {
					if err := websocket.Message.Receive(conn, &noop); err != nil {
						close(done)
						return
					}
				}
			}()

			ticker := time.NewTicker(5 * time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-done:
					return
				case <-ticker.C:
					height, err := setup.getChainHeight()
					if err != nil {
						continue
					}
					if height != lastHeight {
						lastHeight = height
						sendWS(conn, wsMessage{
							Type:      "chain_update",
							Height:    height,
							Timestamp: time.Now().UTC().Format(time.RFC3339),
						})
					}
				}
			}
		},
	}

	server.ServeHTTP(w, r)
}

func (setup OrgSetup) getChainHeight() (uint64, error) {
	network := setup.Gateway.GetNetwork("poschannel")
	qscc := network.GetContract("qscc")
	res, err := qscc.EvaluateTransaction("GetChainInfo", "poschannel")
	if err != nil {
		return 0, err
	}

	info := &common.BlockchainInfo{}
	if err := proto.Unmarshal(res, info); err != nil {
		return 0, err
	}
	return info.Height, nil
}

func sendWS(conn *websocket.Conn, msg wsMessage) {
	payload, err := json.Marshal(msg)
	if err != nil {
		return
	}
	_ = websocket.Message.Send(conn, string(payload))
}
