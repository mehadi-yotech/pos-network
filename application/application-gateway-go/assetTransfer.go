//package main
//
//import (
//	"crypto/x509"
//	"fmt"
//	"log"
//	"net/http"
//	"os"
//	"path"
//
//	"github.com/hyperledger/fabric-gateway/pkg/client"
//	"github.com/hyperledger/fabric-gateway/pkg/identity"
//	"github.com/joho/godotenv"
//	"google.golang.org/grpc"
//	"google.golang.org/grpc/credentials"
//)
//
//func main() {
//	godotenv.Load()
//	contract := initGateway()
//
//	http.HandleFunc("/query", func(w http.ResponseWriter, r *http.Request) {
//		fmt.Println("--> Querying Chaincode...")
//
//		// 1. IMPORTANT: Change "GetAllAssets" to the actual function name
//		// inside your 'poscontract' chaincode!
//		evaluateResponse, err := contract.EvaluateTransaction("GetAllAssets")
//
//		if err != nil {
//			fmt.Printf("Chaincode Error: %v\n", err)
//			http.Error(w, err.Error(), http.StatusInternalServerError)
//			return
//		}
//		w.Header().Set("Content-Type", "application/json")
//		w.Write(evaluateResponse)
//	})
//
//	fmt.Println("API Server started on http://localhost:8082")
//	log.Fatal(http.ListenAndServe(":8082", nil))
//}
//
//func initGateway() *client.Contract {
//	cert, _ := os.ReadFile(os.Getenv("TLS_CERT_PATH"))
//	cp := x509.NewCertPool()
//	cp.AppendCertsFromPEM(cert)
//	creds := credentials.NewClientTLSFromCert(cp, os.Getenv("GATEWAY_PEER"))
//
//	// Establish gRPC connection
//	grpcConn, err := grpc.Dial(os.Getenv("PEER_ENDPOINT"), grpc.WithTransportCredentials(creds))
//	if err != nil {
//		log.Fatalf("gRPC connection failed: %v", err)
//	}
//
//	certBytes, _ := os.ReadFile(os.Getenv("CERT_PATH"))
//	idCert, _ := identity.CertificateFromPEM(certBytes)
//	id, _ := identity.NewX509Identity(os.Getenv("MSPID"), idCert)
//
//	files, _ := os.ReadDir(os.Getenv("KEY_DIR_PATH"))
//	keyBytes, _ := os.ReadFile(path.Join(os.Getenv("KEY_DIR_PATH"), files[0].Name()))
//	privateKey, _ := identity.PrivateKeyFromPEM(keyBytes)
//	signer, _ := identity.NewPrivateKeySign(privateKey)
//
//	// Connect to Gateway
//	gw, err := client.Connect(id, client.WithSign(signer), client.WithClientConnection(grpcConn))
//	if err != nil {
//		log.Fatalf("Gateway connection failed: %v", err)
//	}
//
//	return gw.GetNetwork(os.Getenv("CHANNEL_ID")).GetContract(os.Getenv("CHAINCODE_ID"))
//}

/*
Copyright 2021 IBM All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

/*
Copyright 2021 IBM All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/hyperledger/fabric-gateway/pkg/client"
	"github.com/hyperledger/fabric-gateway/pkg/hash"
	"github.com/hyperledger/fabric-gateway/pkg/identity"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
)

const (
	mspID        = "POSBusinessMSP"
	cryptoPath   = "../../organizations/peerOrganizations/pos.com"
	certPath     = cryptoPath + "/users/User1@pos.com/msp/signcerts"
	keyPath      = cryptoPath + "/users/User1@pos.com/msp/keystore"
	tlsCertPath  = cryptoPath + "/peers/peer0.pos.com/tls/ca.crt"
	peerEndpoint = "dns:///localhost:7051"
	gatewayPeer  = "peer0.pos.com"
)

var now = time.Now()
var assetId = fmt.Sprintf("asset%d", now.Unix()*1e3+int64(now.Nanosecond())/1e6)

func main() {
	clientConnection := newGrpcConnection()
	defer clientConnection.Close()

	id := newIdentity()
	sign := newSign()

	gw, err := client.Connect(
		id,
		client.WithSign(sign),
		client.WithHash(hash.SHA256),
		client.WithClientConnection(clientConnection),
		client.WithEvaluateTimeout(5*time.Second),
		client.WithEndorseTimeout(15*time.Second),
		client.WithSubmitTimeout(5*time.Second),
	)
	if err != nil {
		panic(err)
	}
	defer gw.Close()

	// Configuration
	chaincodeName := "poscontract"
	channelName := "poschannel"

	network := gw.GetNetwork(channelName)
	contract := network.GetContract(chaincodeName)

	uniqueID := fmt.Sprintf("TX_POS_%d", time.Now().Unix())
	// Create a new transaction
	// Arguments: ID, RestaurantID, Amount, StripeID
	recordTransaction(contract, uniqueID, "YoTech_Cafe", "125.50", "ch_stripe_new_999")

	// Let's fetch an existing record
	// Note: Change "tx101" to an ID you know exists in your ledger
	// fetchRecordByID(contract, "STRIPE_100")
	getAllRecords(contract)
}

// recordTransaction adds a new POS transaction to the ledger
func recordTransaction(contract *client.Contract, id string, restaurantID string, amount string, stripeID string) {
	fmt.Printf("\n--> Submit Transaction: RecordTransaction, ID: %s\n", id)

	// Use .Submit instead of .SubmitTransaction to use ProposalOptions
	_, err := contract.Submit("RecordTransaction",
		client.WithArguments(id, restaurantID, amount, stripeID),
		client.WithEndorsingOrganizations("POSBusinessMSP"),
	)

	if err != nil {
		panic(fmt.Errorf("failed to submit transaction: %w", err))
	}

	fmt.Printf("*** Transaction committed successfully\n")
}

// fetchRecordByID calls the GetRecord function in your chaincode
func fetchRecordByID(contract *client.Contract, id string) {
	fmt.Printf("\n--> Evaluate Transaction: GetRecord, function returns record attributes for ID: %s\n", id)

	// We use EvaluateTransaction for queries (no ledger write, no endorsement needed)
	evaluateResult, err := contract.EvaluateTransaction("GetRecord", id)
	if err != nil {
		panic(fmt.Errorf("failed to evaluate transaction: %w", err))
	}

	// Since your chaincode returns a string(recordBytes), we just print it
	fmt.Printf("*** Result: %s\n", string(evaluateResult))
}

// fetch all records
func getAllRecords(contract *client.Contract) {
	fmt.Println("\n--> Evaluate Transaction: GetAllRecords")

	evaluateResult, err := contract.EvaluateTransaction("GetAllRecords")
	if err != nil {
		panic(fmt.Errorf("failed to evaluate transaction: %w", err))
	}

	// The result is a JSON array of strings
	var records []string
	err = json.Unmarshal(evaluateResult, &records)
	if err != nil {
		panic(fmt.Errorf("failed to unmarshal records: %w", err))
	}

	fmt.Printf("*** Found %d records:\n", len(records))
	for _, record := range records {
		fmt.Printf("- %s\n", record)
	}
}

func updateTransaction(contract *client.Contract, id string, restaurantID string, amount string, stripeID string) {
	fmt.Printf("\n--> Submit Transaction: UpdateTransaction, ID: %s\n", id)

	_, err := contract.Submit("UpdateTransaction",
		client.WithArguments(id, restaurantID, amount, stripeID),
		client.WithEndorsingOrganizations("POSBusinessMSP"),
	)

	if err != nil {
		panic(fmt.Errorf("failed to submit transaction: %w", err))
	}

	fmt.Printf("*** Transaction updated successfully\n")
}

func deleteTransaction(contract *client.Contract, id string) {
	fmt.Printf("\n--> Submit Transaction: DeleteTransaction, ID: %s\n", id)

	_, err := contract.Submit("DeleteTransaction",
		client.WithArguments(id),
		client.WithEndorsingOrganizations("POSBusinessMSP"),
	)

	if err != nil {
		panic(fmt.Errorf("failed to submit transaction: %w", err))
	}

	fmt.Printf("*** Transaction deleted successfully\n")
}

// newGrpcConnection creates a gRPC connection to the Gateway server.
func newGrpcConnection() *grpc.ClientConn {
	certificatePEM, err := os.ReadFile(tlsCertPath)
	if err != nil {
		panic(fmt.Errorf("failed to read TLS certificate file: %w", err))
	}

	certificate, err := identity.CertificateFromPEM(certificatePEM)
	if err != nil {
		panic(err)
	}

	certPool := x509.NewCertPool()
	certPool.AddCert(certificate)
	transportCredentials := credentials.NewClientTLSFromCert(certPool, gatewayPeer)

	connection, err := grpc.NewClient(peerEndpoint, grpc.WithTransportCredentials(transportCredentials))
	if err != nil {
		panic(fmt.Errorf("failed to create gRPC connection: %w", err))
	}

	return connection
}

// newIdentity creates a client identity for this Gateway connection using an X.509 certificate.
func newIdentity() *identity.X509Identity {
	certificatePEM, err := readFirstFile(certPath)
	if err != nil {
		panic(fmt.Errorf("failed to read certificate file: %w", err))
	}

	certificate, err := identity.CertificateFromPEM(certificatePEM)
	if err != nil {
		panic(err)
	}

	id, err := identity.NewX509Identity(mspID, certificate)
	if err != nil {
		panic(err)
	}

	return id
}

// newSign creates a function that generates a digital signature from a message digest using a private key.
func newSign() identity.Sign {
	privateKeyPEM, err := readFirstFile(keyPath)
	if err != nil {
		panic(fmt.Errorf("failed to read private key file: %w", err))
	}

	privateKey, err := identity.PrivateKeyFromPEM(privateKeyPEM)
	if err != nil {
		panic(err)
	}

	sign, err := identity.NewPrivateKeySign(privateKey)
	if err != nil {
		panic(err)
	}

	return sign
}

func readFirstFile(dirPath string) ([]byte, error) {
	dir, err := os.Open(dirPath)
	if err != nil {
		return nil, err
	}

	fileNames, err := dir.Readdirnames(1)
	if err != nil {
		return nil, err
	}

	return os.ReadFile(path.Join(dirPath, fileNames[0]))
}

// This type of transaction would typically only be run once by an application the first time it was started after its
// initial deployment. A new version of the chaincode deployed later would likely not need to run an "init" function.
func initLedger(contract *client.Contract) {
	fmt.Printf("\n--> Submit Transaction: InitLedger, function creates the initial set of assets on the ledger \n")

	_, err := contract.SubmitTransaction("InitLedger")
	if err != nil {
		panic(fmt.Errorf("failed to submit transaction: %w", err))
	}

	fmt.Printf("*** Transaction committed successfully\n")
}

// Evaluate a transaction to query ledger state.
func getAllAssets(contract *client.Contract) {
	fmt.Println("\n--> Evaluate Transaction: GetAllAssets, function returns all the current assets on the ledger")

	evaluateResult, err := contract.EvaluateTransaction("GetAllAssets")
	if err != nil {
		panic(fmt.Errorf("failed to evaluate transaction: %w", err))
	}
	result := formatJSON(evaluateResult)

	fmt.Printf("*** Result:%s\n", result)
}

// Submit a transaction synchronously, blocking until it has been committed to the ledger.
func createAsset(contract *client.Contract) {
	fmt.Printf("\n--> Submit Transaction: CreateAsset, creates new asset with ID, Color, Size, Owner and AppraisedValue arguments \n")

	_, err := contract.SubmitTransaction("CreateAsset", assetId, "yellow", "5", "Tom", "1300")
	if err != nil {
		panic(fmt.Errorf("failed to submit transaction: %w", err))
	}

	fmt.Printf("*** Transaction committed successfully\n")
}

// Evaluate a transaction by assetID to query ledger state.
func readAssetByID(contract *client.Contract) {
	fmt.Printf("\n--> Evaluate Transaction: ReadAsset, function returns asset attributes\n")

	evaluateResult, err := contract.EvaluateTransaction("ReadAsset", assetId)
	if err != nil {
		panic(fmt.Errorf("failed to evaluate transaction: %w", err))
	}
	result := formatJSON(evaluateResult)

	fmt.Printf("*** Result:%s\n", result)
}

// Submit transaction asynchronously, blocking until the transaction has been sent to the orderer, and allowing
// this thread to process the chaincode response (e.g. update a UI) without waiting for the commit notification
func transferAssetAsync(contract *client.Contract) {
	fmt.Printf("\n--> Async Submit Transaction: TransferAsset, updates existing asset owner")

	submitResult, commit, err := contract.SubmitAsync("TransferAsset", client.WithArguments(assetId, "Mark"))
	if err != nil {
		panic(fmt.Errorf("failed to submit transaction asynchronously: %w", err))
	}

	fmt.Printf("\n*** Successfully submitted transaction to transfer ownership from %s to Mark. \n", string(submitResult))
	fmt.Println("*** Waiting for transaction commit.")

	if commitStatus, err := commit.Status(); err != nil {
		panic(fmt.Errorf("failed to get commit status: %w", err))
	} else if !commitStatus.Successful {
		panic(fmt.Errorf("transaction %s failed to commit with status: %d", commitStatus.TransactionID, int32(commitStatus.Code)))
	}

	fmt.Printf("*** Transaction committed successfully\n")
}

// Submit transaction, passing in the wrong number of arguments ,expected to throw an error containing details of any error responses from the smart contract.
func exampleErrorHandling(contract *client.Contract) {
	fmt.Println("\n--> Submit Transaction: UpdateAsset asset70, asset70 does not exist and should return an error")

	_, err := contract.SubmitTransaction("UpdateAsset", "asset70", "blue", "5", "Tomoko", "300")
	if err == nil {
		panic("******** FAILED to return an error")
	}

	fmt.Println("*** Successfully caught the error:")

	var commitStatusErr *client.CommitStatusError
	var transactionErr *client.TransactionError

	if errors.As(err, &commitStatusErr) {
		if errors.Is(err, context.DeadlineExceeded) {
			fmt.Printf("Timeout waiting for transaction %s commit status: %s\n", commitStatusErr.TransactionID, commitStatusErr)
		} else {
			fmt.Printf("Error obtaining commit status for transaction %s with gRPC status %v: %s\n", commitStatusErr.TransactionID, status.Code(commitStatusErr), commitStatusErr)
		}
	} else if errors.As(err, &transactionErr) {
		// The error could be an EndorseError, SubmitError or CommitError.
		fmt.Println(err)
		fmt.Printf("TransactionID: %s\n", transactionErr.TransactionID)
	} else {
		panic(fmt.Errorf("unexpected error type %T: %w", err, err))
	}
}

// Format JSON data
func formatJSON(data []byte) string {
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, data, "", "  "); err != nil {
		panic(fmt.Errorf("failed to parse JSON: %w", err))
	}
	return prettyJSON.String()
}
