package web

import (
	"fmt"
	"net/http"

	"github.com/hyperledger/fabric-gateway/pkg/client"
)

func (setup *OrgSetup) Invoke(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Received Invoke request")
	if err := r.ParseForm(); err != nil {
		http.Error(w, fmt.Sprintf("ParseForm() err: %s", err), http.StatusBadRequest)
		return
	}
	chainCodeName := r.FormValue("chaincodeid")
	channelID := r.FormValue("channelid")
	function := r.FormValue("function")
	args := r.Form["args"]
	fmt.Printf("channel: %s, chaincode: %s, function: %s, args: %s\n", channelID, chainCodeName, function, args)

	network := setup.Gateway.GetNetwork(channelID)
	contract := network.GetContract(chainCodeName)

	txn_proposal, err := contract.NewProposal(function,
		client.WithArguments(args...),
		client.WithEndorsingOrganizations(setup.MSPID),
	)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error creating txn proposal: %s", err), http.StatusInternalServerError)
		return
	}
	txn_endorsed, err := txn_proposal.Endorse()
	if err != nil {
		http.Error(w, fmt.Sprintf("Error endorsing txn: %s", err), http.StatusInternalServerError)
		return
	}
	txn_committed, err := txn_endorsed.Submit()
	if err != nil {
		http.Error(w, fmt.Sprintf("Error submitting transaction: %s", err), http.StatusInternalServerError)
		return
	}
	w.Write([]byte(txn_committed.TransactionID()))
}
