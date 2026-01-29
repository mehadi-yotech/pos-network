package web

import (
	"fmt"
	"net/http"

	"github.com/hyperledger/fabric-gateway/pkg/client"
)

type OrgSetup struct {
	OrgName      string
	MSPID        string
	CryptoPath   string
	CertPath     string
	KeyPath      string
	TLSCertPath  string
	PeerEndpoint string
	GatewayPeer  string
	Gateway      client.Gateway
}

func Serve(setups OrgSetup) {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	})

	http.HandleFunc("/query", setups.Query)
	http.HandleFunc("/invoke", setups.Invoke)

	http.HandleFunc("/chaininfo", setups.GetChainInfo)
	http.HandleFunc("/block", setups.GetBlockByNumber)
	http.HandleFunc("/history", setups.GetHistory)
	http.HandleFunc("/search", setups.UniversalSearch)

	fmt.Println("Listening (http://localhost:3000/)...")
	if err := http.ListenAndServe(":3000", nil); err != nil {
		fmt.Println(err)
	}
}
