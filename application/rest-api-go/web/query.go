package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"crypto/sha256"
	"encoding/base64"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-protos-go/common"
)

func setupCORS(w http.ResponseWriter) {
	(w).Header().Set("Access-Control-Allow-Origin", "*")
	(w).Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	(w).Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
}

func (setup OrgSetup) Query(w http.ResponseWriter, r *http.Request) {
	setupCORS(w)
	if r.Method == "OPTIONS" {
		return
	}

	queryParams := r.URL.Query()
	chainCodeName := queryParams.Get("chaincodeid")
	channelID := queryParams.Get("channelid")
	function := queryParams.Get("function")
	args := queryParams["args"]

	network := setup.Gateway.GetNetwork(channelID)
	contract := network.GetContract(chainCodeName)

	evaluateResponse, err := contract.EvaluateTransaction(function, args...)
	if err != nil {
		http.Error(w, fmt.Sprintf("Blockchain Error: %s", err), http.StatusInternalServerError)
		return
	}
	w.Write(evaluateResponse)
}

func (setup OrgSetup) GetHistory(w http.ResponseWriter, r *http.Request) {
	setupCORS(w)
	if r.Method == "OPTIONS" {
		return
	}

	id := r.URL.Query().Get("id")
	channelID := r.URL.Query().Get("channelid")
	chaincodeID := r.URL.Query().Get("chaincodeid")

	network := setup.Gateway.GetNetwork(channelID)
	contract := network.GetContract(chaincodeID)

	res, err := contract.EvaluateTransaction("GetHistory", id)
	if err != nil {
		http.Error(w, fmt.Sprintf("History Error: %s", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(res)
}

func (setup OrgSetup) GetChainInfo(w http.ResponseWriter, r *http.Request) {
	setupCORS(w)
	channelID := r.URL.Query().Get("channelid")
	network := setup.Gateway.GetNetwork(channelID)
	contract := network.GetContract("qscc")

	res, err := contract.EvaluateTransaction("GetChainInfo", channelID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	blockchainInfo := &common.BlockchainInfo{}
	if err := proto.Unmarshal(res, blockchainInfo); err != nil {
		http.Error(w, "Failed to unmarshal blockchain info", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"height": blockchainInfo.Height,
	})
}

func (setup OrgSetup) UniversalSearch(w http.ResponseWriter, r *http.Request) {
	setupCORS(w)
	input := r.URL.Query().Get("input")
	channelID := "poschannel"

	network := setup.Gateway.GetNetwork(channelID)
	qscc := network.GetContract("qscc")
	poscc := network.GetContract("poscontract")

	if blockNum, err := strconv.Atoi(input); err == nil {
		res, err := qscc.EvaluateTransaction("GetBlockByNumber", channelID, strconv.Itoa(blockNum))
		if err == nil {
			block := &common.Block{}
			proto.Unmarshal(res, block)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(block)
			return
		}
	}

	if len(input) >= 64 {
		res, err := qscc.EvaluateTransaction("GetBlockByTxID", channelID, input)
		if err == nil {
			block := &common.Block{}
			if err := proto.Unmarshal(res, block); err == nil {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(block)
				return
			}
		}
	}

	res, err := poscc.EvaluateTransaction("GetRecord", input)
	if err != nil {
		http.Error(w, "Record not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(res)
}

func (setup OrgSetup) GetBlockByNumber(w http.ResponseWriter, r *http.Request) {
	channelID := r.URL.Query().Get("channelid")
	blockNum := r.URL.Query().Get("blocknum")
	network := setup.Gateway.GetNetwork(channelID)
	contract := network.GetContract("qscc")

	res, err := contract.EvaluateTransaction("GetBlockByNumber", channelID, blockNum)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	block := &common.Block{}
	if err := proto.Unmarshal(res, block); err != nil {
		http.Error(w, "Failed to unmarshal block", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(block)
}

func (setup OrgSetup) GetDashboardStats(w http.ResponseWriter, r *http.Request) {
	setupCORS(w)
	channelID := "poschannel"
	network := setup.Gateway.GetNetwork(channelID)
	qscc := network.GetContract("qscc")

	infoRes, err := qscc.EvaluateTransaction("GetChainInfo", channelID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Failed to get chain info"}`))
		return
	}
	blockchainInfo := &common.BlockchainInfo{}
	proto.Unmarshal(infoRes, blockchainInfo)

	height := int(blockchainInfo.Height)
	totalTransactions := 0
	var recentBlocks []map[string]interface{}

	for i := height - 1; i >= 0; i-- {
		res, err := qscc.EvaluateTransaction("GetBlockByNumber", channelID, strconv.Itoa(i))
		if err != nil {
			continue
		}

		block := &common.Block{}
		if err := proto.Unmarshal(res, block); err != nil {
			continue
		}

		txCount := len(block.Data.Data)
		totalTransactions += txCount

		if len(recentBlocks) < 10 {
			timestamp := "N/A"
			if txCount > 0 {
				env := &common.Envelope{}
				if err := proto.Unmarshal(block.Data.Data[0], env); err == nil {
					payload := &common.Payload{}
					if err := proto.Unmarshal(env.Payload, payload); err == nil {
						chHeader := &common.ChannelHeader{}
						if err := proto.Unmarshal(payload.Header.ChannelHeader, chHeader); err == nil {
							if chHeader.Timestamp != nil {
								// timestamp = time.Unix(chHeader.Timestamp.Seconds, int64(chHeader.Timestamp.Nanos)).Format(time.RFC3339)
							}
						}
					}
				}
			}

			recentBlocks = append(recentBlocks, map[string]interface{}{
				"number":     i,
				"tx_count":   txCount,
				"data_hash":  fmt.Sprintf("%x", block.Header.DataHash),
				"pre_hash":   fmt.Sprintf("%x", block.Header.PreviousHash),
				"created_at": timestamp,
				"channel":    channelID,
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"height":             height,
		"total_transactions": totalTransactions,
		"peers":              []string{"peer0.pos.com", "peer1.pos.com"},
		"recent_blocks":      recentBlocks,
	})
}

//	func computeBlockHash(block *common.Block) string {
//	    headerBytes := ...
//	    hash := sha256.Sum256(headerBytes)
//	    return base64.StdEncoding.EncodeToString(hash[:])
//	}
func computeBlockHash(block *common.Block) string {
	headerBytes := []byte(fmt.Sprintf("%d%x%x", block.Header.Number, block.Header.PreviousHash, block.Header.DataHash))
	hash := sha256.Sum256(headerBytes)
	return base64.StdEncoding.EncodeToString(hash[:])
}

func decodeBlockData(rawBytes []byte) {
	envelope := &common.Envelope{}
	proto.Unmarshal(rawBytes, envelope)

	payload := &common.Payload{}
	proto.Unmarshal(envelope.Payload, payload)
}
