package main

import (
	"fmt"
	"rest-api-go/web"
)

func main() {
	cryptoPath := "/home/yotech-65/go/src/github.com/mehadi-yotech/pos-network/organizations/peerOrganizations/pos.com"
	orgConfig := web.OrgSetup{
		OrgName:      "pos.com",
		MSPID:        "POSBusinessMSP",
		CertPath:     cryptoPath + "/users/User1@pos.com/msp/signcerts/User1@pos.com-cert.pem",
		KeyPath:      cryptoPath + "/users/User1@pos.com/msp/keystore/",
		TLSCertPath:  cryptoPath + "/peers/peer0.pos.com/tls/ca.crt",
		PeerEndpoint: "dns:///localhost:7051",
		GatewayPeer:  "peer0.pos.com",
	}

	orgSetup, err := web.Initialize(orgConfig)
	if err != nil {
		fmt.Println("Error initializing setup for pos.com: ", err)
	}
	web.Serve(web.OrgSetup(*orgSetup))
}
