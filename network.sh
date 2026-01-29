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


#
#docker rm -f $(docker ps -aq)
#docker volume prune -f
#sudo rm -rf organizations/peerOrganizations
#sudo rm -rf organizations/ordererOrganizations
#sudo rm -rf channel-artifacts/
#docker network prune -f
#docker rm -f $(docker ps -aq)
#docker volume rm compose_orderer.example.com compose_peer0.org1.example.com compose_peer0.org2.example.com
#docker volume ls
#
#
#
#cd docker
#docker-compose down --volumes --remove-orphans
#
#cd ..
#rm -rf organizations/ordererOrganizations
#rm -rf organizations/peerOrganizations
#rm -f channel-artifacts/poschannel.block
#
#docker rm -f $(docker ps -aq)
#docker system prune -a --volumes
#docker ps -a
#
#
#
#
#sudo rm -rf poscontract.tar.gz
#cd organizations
#sudo rm -rf ordererOrganizations
#sudo rm -rf peerOrganizations
#cd ..
#cd channel-artifacts
#sudo rm -rf poschannel.block
#cd ..
#cd chaincode
#cd poscontract
#sudo rm -rf vendor
#sudo rm -rf go.mod
#sudo rm -rf go.sum
#
#
#./bin/cryptogen generate --config=./crypto-config.yaml --output="organizations"
#sleep 2
#
#export FABRIC_CFG_PATH=$PWD/config
#./bin/configtxgen -profile POSChannelProfile -outputBlock ./channel-artifacts/poschannel.block -channelID poschannel
#chmod +x ./bin/*
#sleep 2
#
#cd docker
#docker-compose up -d
#sleep 10
#
#cd ..

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

sleep 15

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

#./bin/osnadmin channel join \
#  --channelID poschannel \
#  --config-block ./channel-artifacts/poschannel.block \
#  -o orderer0.pos.com:7053 \
#  --ca-file ./organizations/ordererOrganizations/pos.com/orderers/orderer0.pos.com/tls/ca.crt \
#  --client-cert ./organizations/ordererOrganizations/pos.com/orderers/orderer0.pos.com/tls/server.crt \
#  --client-key ./organizations/ordererOrganizations/pos.com/orderers/orderer0.pos.com/tls/server.key
#
#sleep 2
#
#./bin/osnadmin channel join \
#  --channelID poschannel \
#  --config-block ./channel-artifacts/poschannel.block \
#  -o orderer1.pos.com:8053 \
#  --ca-file ./organizations/ordererOrganizations/pos.com/orderers/orderer1.pos.com/tls/ca.crt \
#  --client-cert ./organizations/ordererOrganizations/pos.com/orderers/orderer1.pos.com/tls/server.crt \
#  --client-key ./organizations/ordererOrganizations/pos.com/orderers/orderer1.pos.com/tls/server.key
#
#sleep 2
#
#./bin/osnadmin channel join \
#  --channelID poschannel \
#  --config-block ./channel-artifacts/poschannel.block \
#  -o orderer2.pos.com:9053 \
#  --ca-file ./organizations/ordererOrganizations/pos.com/orderers/orderer2.pos.com/tls/ca.crt \
#  --client-cert ./organizations/ordererOrganizations/pos.com/orderers/orderer2.pos.com/tls/server.crt \
#  --client-key ./organizations/ordererOrganizations/pos.com/orderers/orderer2.pos.com/tls/server.key
#
#sleep 20
#
#./bin/osnadmin channel list -o orderer0.pos.com:7053 --ca-file ./organizations/ordererOrganizations/pos.com/orderers/orderer0.pos.com/tls/ca.crt --client-cert ./organizations/ordererOrganizations/pos.com/orderers/orderer0.pos.com/tls/server.crt --client-key ./organizations/ordererOrganizations/pos.com/orderers/orderer0.pos.com/tls/server.key
#
#sleep 20

#docker logs orderer0.pos.com 2>&1 | grep "Raft leader changed"
#
#export FABRIC_CFG_PATH=$PWD/config
#export CORE_PEER_TLS_ENABLED=true
#export CORE_PEER_LOCALMSPID="POSBusinessMSP"
#export CORE_PEER_TLS_ROOTCERT_FILE=$PWD/organizations/peerOrganizations/pos.com/peers/peer0.pos.com/tls/ca.crt
#export CORE_PEER_MSPCONFIGPATH=$PWD/organizations/peerOrganizations/pos.com/users/Admin@pos.com/msp
#export CORE_PEER_ADDRESS=peer0.pos.com:7051
#sleep 2
#./bin/peer channel join -b ./channel-artifacts/poschannel.block
#sleep 2
#
#export CORE_PEER_ADDRESS=peer1.pos.com:9051
#export CORE_PEER_TLS_ROOTCERT_FILE=$PWD/organizations/peerOrganizations/pos.com/peers/peer1.pos.com/tls/ca.crt
#sleep 2
#./bin/peer channel join -b ./channel-artifacts/poschannel.block
#./bin/peer channel list
#
#cd chaincode/poscontract
#go mod init poscontract
#go mod tidy
#
#cd ../..
#sudo chmod 666 /var/run/docker.sock
#
#./bin/peer lifecycle chaincode package poscontract.tar.gz \
#  --path ./chaincode/poscontract/ \
#  --lang golang \
#  --label poscontract_1.0
#
#sleep 2
## Ensure Peer 0 Env is set:
#export FABRIC_CFG_PATH=$PWD/config
#export CORE_PEER_TLS_ENABLED=true
#export CORE_PEER_LOCALMSPID="POSBusinessMSP"
#export CORE_PEER_TLS_ROOTCERT_FILE=$PWD/organizations/peerOrganizations/pos.com/peers/peer0.pos.com/tls/ca.crt
#export CORE_PEER_MSPCONFIGPATH=$PWD/organizations/peerOrganizations/pos.com/users/Admin@pos.com/msp
#export CORE_PEER_ADDRESS=peer0.pos.com:7051
#sleep 2
#./bin/peer lifecycle chaincode install poscontract.tar.gz
#
#sleep 2
## Change env variables to Peer 1
#export CORE_PEER_ADDRESS=peer1.pos.com:9051
#export CORE_PEER_TLS_ROOTCERT_FILE=$PWD/organizations/peerOrganizations/pos.com/peers/peer1.pos.com/tls/ca.crt
#sleep 2
#./bin/peer lifecycle chaincode install poscontract.tar.gz
#sleep 2
#
#echo "Finding Package ID..."
#
## 1. Capture the ID into a variable called PACKAGE_ID
## We use 'awk' to find the line with "Package ID:" and print the 3rd column
## PACKAGE_ID=$(./bin/peer lifecycle chaincode queryinstalled | awk -F 'Package ID: |, Label' '/Package ID:/{print $2}')
## Finds the Package ID specifically for 'poscontract_1.0'
#PACKAGE_ID=$(./bin/peer lifecycle chaincode queryinstalled | grep "Label: poscontract" | tail -n 1 | awk -F 'Package ID: |, Label' '{print $2}')
## 2. Check if the variable is empty (safety check)
#if [ -z "$PACKAGE_ID" ]; then
#    echo "Error: Could not find a Package ID. Is the chaincode installed?"
#    exit 1
#fi
#
#echo "Found Package ID: $PACKAGE_ID"
#
## 3. Use the variable in the next command
#./bin/peer lifecycle chaincode approveformyorg \
#  -o orderer0.pos.com:7050 \
#  --ordererTLSHostnameOverride orderer0.pos.com \
#  --channelID poschannel \
#  --name poscontract \
#  --version 1.0 \
#  --package-id "$PACKAGE_ID" \
#  --sequence 1 \
#  --tls \
#  --cafile "$PWD/organizations/ordererOrganizations/pos.com/orderers/orderer0.pos.com/tls/ca.crt"
#
#echo "Approval submitted successfully!"
#sleep 10
#
#
#./bin/peer lifecycle chaincode commit \
#  -o orderer0.pos.com:7050 \
#  --ordererTLSHostnameOverride orderer0.pos.com \
#  --channelID poschannel \
#  --name poscontract \
#  --version 1.0 \
#  --sequence 1 \
#  --tls \
#  --cafile $PWD/organizations/ordererOrganizations/pos.com/orderers/orderer0.pos.com/tls/ca.crt \
#  --peerAddresses peer0.pos.com:7051 \
#  --tlsRootCertFiles $PWD/organizations/peerOrganizations/pos.com/peers/peer0.pos.com/tls/ca.crt \
#  --peerAddresses peer1.pos.com:9051 \
#  --tlsRootCertFiles $PWD/organizations/peerOrganizations/pos.com/peers/peer1.pos.com/tls/ca.crt
#
#sleep 10
#
#
#./bin/peer chaincode invoke \
#  -o orderer0.pos.com:7050 \
#  --ordererTLSHostnameOverride orderer0.pos.com \
#  --tls \
#  --cafile $PWD/organizations/ordererOrganizations/pos.com/orderers/orderer0.pos.com/tls/ca.crt \
#  --channelID poschannel \
#  --name poscontract \
#  --peerAddresses peer0.pos.com:7051 --tlsRootCertFiles $PWD/organizations/peerOrganizations/pos.com/peers/peer0.pos.com/tls/ca.crt \
#  --peerAddresses peer1.pos.com:9051 --tlsRootCertFiles $PWD/organizations/peerOrganizations/pos.com/peers/peer1.pos.com/tls/ca.crt \
#  -c '{"Args":["RecordTransaction","STRIPE_100","SushiGarden","55.00","ch_3Oljlk23"]}'
#
#sleep 2
#
#./bin/peer chaincode query -C poschannel -n poscontract -c '{"Args":["GetRecord","STRIPE_100"]}'
#
#
#
#cd application/rest-api-go
#go run .

docker logs orderer0.pos.com 2>&1 | grep "Raft leader changed"

echo "--- Step 6: Preparing Chaincode & Installation ---"
cd chaincode/poscontract
go mod init poscontract 2>/dev/null || true
go mod tidy
cd ../..

sudo chmod 666 /var/run/docker.sock

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


