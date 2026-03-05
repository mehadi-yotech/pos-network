#!/bin/bash
set -e
# --- 1. Aggressive Cleanup ---
echo "Cleaning up environment..."
docker rm -f $(docker ps -aq) 2>/dev/null || true
docker volume prune -f
docker network prune -f
docker system prune -a --volumes -f

# Consolidate directory removal
sudo rm -rf organizations/peerOrganizations organizations/ordererOrganizations channel-artifacts/ poscontract.tar.gz
sudo rm -rf chaincode/poscontract/vendor chaincode/poscontract/go.mod chaincode/poscontract/go.sum
mkdir -p channel-artifacts

# --- 2. Crypto & Artifact Generation ---
./bin/cryptogen generate --config=./crypto-config.yaml --output="organizations"
export FABRIC_CFG_PATH=$PWD/config
./bin/configtxgen -profile POSChannelProfile -outputBlock ./channel-artifacts/poschannel.block -channelID poschannel
chmod +x ./bin/*

# --- 3. Start Network ---
cd docker
docker-compose up -d
cd ..
sleep 10

echo "--- Step 4: Joining Orderers to Channel ---"
# Define common paths to reduce clutter
ORDERER_CA=$PWD/organizations/ordererOrganizations/pos.com/orderers/orderer0.pos.com/tls/ca.crt
ORDERER_CERT=$PWD/organizations/ordererOrganizations/pos.com/orderers/orderer0.pos.com/tls/server.crt
ORDERER_KEY=$PWD/organizations/ordererOrganizations/pos.com/orderers/orderer0.pos.com/tls/server.key

# Joining the three orderers
./bin/osnadmin channel join --channelID poschannel --config-block ./channel-artifacts/poschannel.block -o orderer0.pos.com:7053 --ca-file $ORDERER_CA --client-cert $ORDERER_CERT --client-key $ORDERER_KEY
sleep 2
./bin/osnadmin channel join --channelID poschannel --config-block ./channel-artifacts/poschannel.block -o orderer1.pos.com:8053 --ca-file $ORDERER_CA --client-cert $PWD/organizations/ordererOrganizations/pos.com/orderers/orderer1.pos.com/tls/server.crt --client-key $PWD/organizations/ordererOrganizations/pos.com/orderers/orderer1.pos.com/tls/server.key
sleep 2
./bin/osnadmin channel join --channelID poschannel --config-block ./channel-artifacts/poschannel.block -o orderer2.pos.com:9053 --ca-file $ORDERER_CA --client-cert $PWD/organizations/ordererOrganizations/pos.com/orderers/orderer2.pos.com/tls/server.crt --client-key $PWD/organizations/ordererOrganizations/pos.com/orderers/orderer2.pos.com/tls/server.key

sleep 25

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

docker logs orderer0.pos.com 2>&1 | grep "Raft leader changed" || true

echo "--- Step 6: Preparing Chaincode & Installation ---"
cd chaincode/poscontract
go mod init poscontract 2>/dev/null || true
go mod tidy
cd ../..

sudo chmod 666 /var/run/docker.sock
docker pull hyperledger/fabric-ccenv:3.1

echo "packaging chaincode"
./bin/peer lifecycle chaincode package poscontract.tar.gz --path ./chaincode/poscontract/ --lang golang --label poscontract_1.0

echo "--- Step 7: Installing on Peers ---"
# Install on Peer 0
export CORE_PEER_ADDRESS=peer0.pos.com:7051
export CORE_PEER_TLS_ROOTCERT_FILE=$PWD/organizations/peerOrganizations/pos.com/peers/peer0.pos.com/tls/ca.crt
./bin/peer lifecycle chaincode install poscontract.tar.gz

# Install on Peer 1
export CORE_PEER_ADDRESS=peer1.pos.com:9051
export CORE_PEER_TLS_ROOTCERT_FILE=$PWD/organizations/peerOrganizations/pos.com/peers/peer1.pos.com/tls/ca.crt
./bin/peer lifecycle chaincode install poscontract.tar.gz

echo "--- Step 8: Automated Approval ---"
echo "Waiting for Raft leader election..."
sleep 10
# Capturing the Package ID automatically
PACKAGE_ID=$(./bin/peer lifecycle chaincode queryinstalled | grep "Label: poscontract" | tail -n 1 | awk -F 'Package ID: |, Label' '{print $2}')

if [ -z "$PACKAGE_ID" ]; then
    echo "Error: Package ID not found!"
    exit 1
fi

./bin/peer lifecycle chaincode approveformyorg \
  -o orderer0.pos.com:7050 \
  --ordererTLSHostnameOverride orderer0.pos.com \
  --channelID poschannel \
  --name poscontract \
  --version 1.0 \
  --package-id "$PACKAGE_ID" \
  --sequence 1 \
  --tls \
  --cafile "$ORDERER_CA"

echo "--- Step 9: Committing and Testing ---"
sleep 5

./bin/peer lifecycle chaincode commit \
  -o orderer0.pos.com:7050 \
  --ordererTLSHostnameOverride orderer0.pos.com \
  --channelID poschannel \
  --name poscontract \
  --version 1.0 \
  --sequence 1 \
  --tls \
  --cafile "$ORDERER_CA" \
  --peerAddresses peer0.pos.com:7051 --tlsRootCertFiles $PWD/organizations/peerOrganizations/pos.com/peers/peer0.pos.com/tls/ca.crt \
  --peerAddresses peer1.pos.com:9051 --tlsRootCertFiles $PWD/organizations/peerOrganizations/pos.com/peers/peer1.pos.com/tls/ca.crt

sleep 5

# Final Invoke & Query test
./bin/peer chaincode invoke -o orderer0.pos.com:7050 --ordererTLSHostnameOverride orderer0.pos.com --tls --cafile "$ORDERER_CA" --channelID poschannel --name poscontract --peerAddresses peer0.pos.com:7051 --tlsRootCertFiles $PWD/organizations/peerOrganizations/pos.com/peers/peer0.pos.com/tls/ca.crt --peerAddresses peer1.pos.com:9051 --tlsRootCertFiles $PWD/organizations/peerOrganizations/pos.com/peers/peer1.pos.com/tls/ca.crt -c '{"Args":["RecordTransaction","STRIPE_100","SushiGarden","55.00","ch_3Oljlk23"]}'

sleep 2

./bin/peer chaincode query -C poschannel -n poscontract -c '{"Args":["GetRecord","STRIPE_100"]}'

echo "--- Step 10: Launching REST API ---"
cd application/rest-api-go
go run .


