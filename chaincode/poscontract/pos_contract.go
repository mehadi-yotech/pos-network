package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

type SmartContract struct {
	contractapi.Contract
}

type Transaction struct {
	ID              string  `json:"id"`
	RestaurantID    string  `json:"restaurant_id"`
	Amount          float64 `json:"amount"`
	StripePaymentID string  `json:"stripe_payment_id"`
	Timestamp       string  `json:"timestamp"`
	Status          string  `json:"status"`
}

type Payout struct {
	ID           string   `json:"id"`
	RestaurantID string   `json:"restaurant_id"`
	TotalAmount  float64  `json:"total_amount"`
	TxIDs        []string `json:"tx_ids"`
	Status       string   `json:"status"`
	PayoutDate   string   `json:"payout_date"`
}

func (s *SmartContract) RecordTransaction(ctx contractapi.TransactionContextInterface, id string, restaurantID string, amount float64, stripeID string) error {
	tx := Transaction{
		ID:              id,
		RestaurantID:    restaurantID,
		Amount:          amount,
		StripePaymentID: stripeID,
		Timestamp:       time.Now().Format(time.RFC3339),
		Status:          "Settled",
	}
	txBytes, _ := json.Marshal(tx)
	return ctx.GetStub().PutState(id, txBytes)
}

func (s *SmartContract) CreatePayout(ctx contractapi.TransactionContextInterface, id string, restaurantID string, amount float64, txIDs []string) error {
	payout := Payout{
		ID:           id,
		RestaurantID: restaurantID,
		TotalAmount:  amount,
		TxIDs:        txIDs,
		Status:       "Pending",
		PayoutDate:   time.Now().Format(time.RFC3339),
	}
	payoutBytes, _ := json.Marshal(payout)
	return ctx.GetStub().PutState(id, payoutBytes)
}

func (s *SmartContract) UpdatePayoutStatus(ctx contractapi.TransactionContextInterface, id string, newStatus string) error {
	payoutBytes, err := ctx.GetStub().GetState(id)
	if err != nil || payoutBytes == nil {
		return fmt.Errorf("payout %s not found", id)
	}

	var payout Payout
	json.Unmarshal(payoutBytes, &payout)
	payout.Status = newStatus

	payoutBytes, _ = json.Marshal(payout)
	return ctx.GetStub().PutState(id, payoutBytes)
}

func (s *SmartContract) GetRecord(ctx contractapi.TransactionContextInterface, id string) (string, error) {
	recordBytes, err := ctx.GetStub().GetState(id)
	if err != nil || recordBytes == nil {
		return "", fmt.Errorf("record %s not found", id)
	}
	return string(recordBytes), nil
}

func (s *SmartContract) GetAllRecords(ctx contractapi.TransactionContextInterface) ([]string, error) {
	resultsIterator, err := ctx.GetStub().GetStateByRange("", "")
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	var records []string
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}
		records = append(records, string(queryResponse.Value))
	}

	return records, nil
}

func (s *SmartContract) UpdateTransaction(ctx contractapi.TransactionContextInterface, id string, restaurantId string, amountStr string, stripePaymentId string) error {
	recordBytes, err := ctx.GetStub().GetState(id)
	if err != nil {
		return fmt.Errorf("failed to read from world state: %v", err)
	}
	if recordBytes == nil {
		return fmt.Errorf("the record %s does not exist", id)
	}

	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		return fmt.Errorf("amount must be a valid number: %v", err)
	}

	record := map[string]interface{}{
		"id":                id,
		"restaurant_id":     restaurantId,
		"amount":            amount,
		"stripe_payment_id": stripePaymentId,
		"status":            "Updated",
		"timestamp":         time.Now().Format(time.RFC3339),
	}

	newRecordBytes, _ := json.Marshal(record)
	return ctx.GetStub().PutState(id, newRecordBytes)
}

func (s *SmartContract) DeleteTransaction(ctx contractapi.TransactionContextInterface, id string) error {
	recordBytes, err := ctx.GetStub().GetState(id)
	if err != nil {
		return fmt.Errorf("failed to read from world state: %v", err)
	}
	if recordBytes == nil {
		return fmt.Errorf("the record %s does not exist", id)
	}

	return ctx.GetStub().DelState(id)
}

func (s *SmartContract) GetHistory(ctx contractapi.TransactionContextInterface, id string) ([]map[string]interface{}, error) {
	resultsIterator, err := ctx.GetStub().GetHistoryForKey(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get history for record %s: %v", id, err)
	}
	defer resultsIterator.Close()

	var history []map[string]interface{}
	for resultsIterator.HasNext() {
		response, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}

		var record interface{}
		if !response.IsDelete {
			err = json.Unmarshal(response.Value, &record)
			if err != nil {
				return nil, err
			}
		} else {
			record = "DELETED"
		}

		historyEntry := map[string]interface{}{
			"txId":      response.TxId,
			"timestamp": time.Unix(response.Timestamp.Seconds, int64(response.Timestamp.Nanos)).Format(time.RFC3339),
			"isDelete":  response.IsDelete,
			"value":     record,
		}
		history = append(history, historyEntry)
	}

	return history, nil
}

func (s *SmartContract) GetRecordsWithMetadata(ctx contractapi.TransactionContextInterface) ([]map[string]interface{}, error) {
	resultsIterator, err := ctx.GetStub().GetStateByRange("", "")
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	var records []map[string]interface{}
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}

		var entity interface{}
		json.Unmarshal(queryResponse.Value, &entity)

		record := map[string]interface{}{
			"key":  queryResponse.Key,
			"data": entity,
			"txId": "Click for History",
		}
		records = append(records, record)
	}
	return records, nil
}

func main() {
	chaincode, err := contractapi.NewChaincode(&SmartContract{})
	if err != nil {
		fmt.Printf("Error creating POS chaincode: %s", err)
		return
	}

	if err := chaincode.Start(); err != nil {
		fmt.Printf("Error starting POS chaincode: %s", err)
	}
}
