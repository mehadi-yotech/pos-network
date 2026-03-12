package main

import (
	"fmt"
	"os"

	"rest-api-go/web"
)

func main() {
	cryptoPath := "/home/yotech-65/go/src/github.com/mehadi-yotech/pos-network/organizations/peerOrganizations/pos.com"
	peerEndpoint := os.Getenv("PEER_ENDPOINT")
	if peerEndpoint == "" {
		peerEndpoint = "localhost:7051"
	}
	gatewayPeer := os.Getenv("GATEWAY_PEER")
	if gatewayPeer == "" {
		gatewayPeer = "peer0.pos.com"
	}
	orgConfig := web.OrgSetup{
		OrgName:      "pos.com",
		MSPID:        "POSBusinessMSP",
		CertPath:     cryptoPath + "/users/User1@pos.com/msp/signcerts/User1@pos.com-cert.pem",
		KeyPath:      cryptoPath + "/users/User1@pos.com/msp/keystore/",
		TLSCertPath:  cryptoPath + "/peers/peer0.pos.com/tls/ca.crt",
		PeerEndpoint: peerEndpoint,
		GatewayPeer:  gatewayPeer,
	}

	orgSetup, err := web.Initialize(orgConfig)
	if err != nil {
		fmt.Println("Error initializing setup for pos.com: ", err)
	}
	web.Serve(web.OrgSetup(*orgSetup))
}
