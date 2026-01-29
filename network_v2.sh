#!/bin/bash
set -e

# --- 1. Aggressive Cleanup ---
echo "--- Step 1: Cleaning environment ---"
docker rm -f $(docker ps -aq) 2>/dev/null || true
docker volume prune -f
docker network prune -f
# Removed 'docker system prune -a' to prevent deleting your fabric images every time, saving time.

sudo rm -rf organizations/peerOrganizations organizations/ordererOrganizations channel-artifacts/ poscontract.tar.gz
sudo rm -rf chaincode/poscontract/vendor chaincode/poscontract/go.mod chaincode/poscontract/go.sum
mkdir -p channel-artifacts

# --- 2. Crypto & Artifact Generation ---
echo "--- Step 2: Generating Artifacts ---"
./bin/cryptogen generate --config=./crypto-config.yaml --output="organizations"
export FABRIC_CFG_PATH=$PWD/config
./bin/configtxgen -profile POSChannelProfile -outputBlock ./channel-artifacts/poschannel.block -channelID poschannel
chmod +x ./bin/*

# --- 3. Start Network ---
echo "--- Step 3: Starting Docker Containers ---"
cd docker
docker-compose up -d
cd ..
echo "Waiting for containers to stabilize..."
sleep 15

# --- 4. Orderers Join Channel ---
echo "--- Step 4: Joining Orderers to Channel ---"
ORDERER_CA=$PWD/organizations/ordererOrganizations/pos.com/orderers/orderer0.pos.com/tls/ca.crt
ORDERER_CERT=$PWD/organizations/ordererOrganizations/pos.com/orderers/orderer0.pos.com/tls/server.crt
ORDERER_KEY=$PWD/organizations/ordererOrganizations/pos.com/orderers/orderer0.pos.com/tls/server.key

./bin/osnadmin channel join --channelID poschannel --config-block ./channel-artifacts/poschannel.block -o orderer0.pos.com:7053 --ca-file $ORDERER_CA --client-cert $ORDERER_CERT --client-key $ORDERER_KEY
./bin/osnadmin channel join --channelID poschannel --config-block ./channel-artifacts/poschannel.block -o orderer1.pos.com:8053 --ca-file $ORDERER_CA --client-cert $PWD/organizations/ordererOrganizations/pos.com/orderers/orderer1.pos.com/tls/server.crt --client-key $PWD/organizations/ordererOrganizations/pos.com/orderers/orderer1.pos.com/tls/server.key
./bin/osnadmin channel join --channelID poschannel --config-block ./channel-artifacts/poschannel.block -o orderer2.pos.com:9053 --ca-file $ORDERER_CA --client-cert $PWD/organizations/ordererOrganizations/pos.com/orderers/orderer2.pos.com/tls/server.crt --client-key $PWD/organizations/ordererOrganizations/pos.com/orderers/orderer2.pos.com/tls/server.key

sleep 5

# --- 5. Peers Join Channel ---
echo "--- Step 5: Joining Peers to Channel ---"
export FABRIC_CFG_PATH=$PWD/config
export CORE_PEER_TLS_ENABLED=true
export CORE_PEER_LOCALMSPID="POSBusinessMSP"
export CORE_PEER_MSPCONFIGPATH=$PWD/organizations/peerOrganizations/pos.com/users/Admin@pos.com/msp

# Join Peer 0
export CORE_PEER_ADDRESS=peer0.pos.com:7051
export CORE_PEER_TLS_ROOTCERT_FILE=$PWD/organizations/peerOrganizations/pos.com/peers/peer0.pos.com/tls/ca.crt
./bin/peer channel join -b ./channel-artifacts/poschannel.block
sleep 2

# Join Peer 1
export CORE_PEER_ADDRESS=peer1.pos.com:9051
export CORE_PEER_TLS_ROOTCERT_FILE=$PWD/organizations/peerOrganizations/pos.com/peers/peer1.pos.com/tls/ca.crt
./bin/peer channel join -b ./channel-artifacts/poschannel.block

# --- 6. Chaincode Preparation ---
echo "--- Step 6: Preparing Chaincode ---"
cd chaincode/poscontract
go mod init poscontract 2>/dev/null || true
go mod tidy
cd ../..
sudo chmod 666 /var/run/docker.sock
./bin/peer lifecycle chaincode package poscontract.tar.gz --path ./chaincode/poscontract/ --lang golang --label poscontract_1.0

# --- 7. Chaincode Installation ---
echo "--- Step 7: Installing Chaincode ---"
# Peer 0
export CORE_PEER_ADDRESS=peer0.pos.com:7051
export CORE_PEER_TLS_ROOTCERT_FILE=$PWD/organizations/peerOrganizations/pos.com/peers/peer0.pos.com/tls/ca.crt
./bin/peer lifecycle chaincode install poscontract.tar.gz

# Peer 1
export CORE_PEER_ADDRESS=peer1.pos.com:9051
export CORE_PEER_TLS_ROOTCERT_FILE=$PWD/organizations/peerOrganizations/pos.com/peers/peer1.pos.com/tls/ca.crt
./bin/peer lifecycle chaincode install poscontract.tar.gz

# --- 8. Approval & Commit ---
# --- 8. Approval & Commit ---
echo "--- Step 8: Approving and Committing ---"
echo "Giving Raft cluster extra time to elect a leader..."
sleep 30  # Increase this from 5 to 30 temporarily
PACKAGE_ID=$(./bin/peer lifecycle chaincode queryinstalled | grep "Label: poscontract" | tail -n 1 | awk -F 'Package ID: |, Label' '{print $2}')

./bin/peer lifecycle chaincode approveformyorg -o orderer0.pos.com:7050 --ordererTLSHostnameOverride orderer0.pos.com --channelID poschannel --name poscontract --version 1.0 --package-id "$PACKAGE_ID" --sequence 1 --tls --cafile "$ORDERER_CA"

sleep 5

./bin/peer lifecycle chaincode commit -o orderer0.pos.com:7050 --ordererTLSHostnameOverride orderer0.pos.com --channelID poschannel --name poscontract --version 1.0 --sequence 1 --tls --cafile "$ORDERER_CA" --peerAddresses peer0.pos.com:7051 --tlsRootCertFiles $PWD/organizations/peerOrganizations/pos.com/peers/peer0.pos.com/tls/ca.crt --peerAddresses peer1.pos.com:9051 --tlsRootCertFiles $PWD/organizations/peerOrganizations/pos.com/peers/peer1.pos.com/tls/ca.crt

# --- 9. Test Invoke ---
echo "--- Step 9: Testing Chaincode ---"
sleep 5
./bin/peer chaincode invoke -o orderer0.pos.com:7050 --ordererTLSHostnameOverride orderer0.pos.com --tls --cafile "$ORDERER_CA" --channelID poschannel --name poscontract --peerAddresses peer0.pos.com:7051 --tlsRootCertFiles $PWD/organizations/peerOrganizations/pos.com/peers/peer0.pos.com/tls/ca.crt --peerAddresses peer1.pos.com:9051 --tlsRootCertFiles $PWD/organizations/peerOrganizations/pos.com/peers/peer1.pos.com/tls/ca.crt -c '{"Args":["RecordTransaction","STRIPE_100","SushiGarden","55.00","ch_3Oljlk23"]}'

# --- 10. REST API ---
echo "--- Step 10: Starting REST API ---"
cd application/rest-api-go
# If this fails, make sure you have a go.mod in the application folder
go mod tidy
go run .