package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"os"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

const (
	USDT_ADDRESS = "0xdAC17F958D2ee523a2206206994597C13D831ec7"
	RPC_URL      = "https://rpc.ankr.com/eth"
)

func main() {
	client, err := ethclient.Dial(RPC_URL)
	if err != nil {
		log.Fatal(err)
	}

	// get latest block
	header, err := client.HeaderByNumber(context.Background(), nil)
	if err != nil {
		log.Fatal(err)
	}
	latestBlock := header.Number.Uint64()

	// calculate issue event signature
	issueEventSig := []byte("Issue(uint256)")
	issueEventHash := crypto.Keccak256Hash(issueEventSig)
	fmt.Printf("Issue Event Hash: %s\n", issueEventHash.Hex())

	// increase search range
	batchSize := uint64(10000)
	totalBlocks := uint64(500000)

	startBlock := latestBlock
	if totalBlocks > latestBlock {
		startBlock = 0
	} else {
		startBlock = latestBlock - totalBlocks
	}

	// query by batch
	for currentBlock := latestBlock; currentBlock > startBlock; {
		batchStart := currentBlock - batchSize
		if batchStart < startBlock {
			batchStart = startBlock
		}

		query := ethereum.FilterQuery{
			FromBlock: big.NewInt(int64(batchStart)),
			ToBlock:   big.NewInt(int64(currentBlock)),
			Addresses: []common.Address{common.HexToAddress(USDT_ADDRESS)},
			Topics: [][]common.Hash{
				{issueEventHash}, // Issue event signature
			},
		}

		logs, err := client.FilterLogs(context.Background(), query)
		if err != nil {
			log.Printf("Error querying blocks %d to %d: %v\n", batchStart, currentBlock, err)
			currentBlock = batchStart
			continue
		}

		fmt.Printf("Searching blocks %d to %d...\n", batchStart, currentBlock)

		for _, vLog := range logs {
			amount := new(big.Int).SetBytes(vLog.Data)
			actualAmount := new(big.Float).Quo(
				new(big.Float).SetInt(amount),
				new(big.Float).SetInt64(1000000),
			)

			// get transaction by hash
			tx, _, err := client.TransactionByHash(context.Background(), vLog.TxHash)
			if err != nil {
				continue
			}

			from, err := client.TransactionSender(context.Background(), tx, vLog.BlockHash, vLog.TxIndex)
			if err != nil {
				continue
			}

			fmt.Printf("\nFound USDT Issue event:\n")
			fmt.Printf("Block Number: %d\n", vLog.BlockNumber)
			fmt.Printf("Transaction: https://etherscan.io/tx/%s\n", vLog.TxHash.Hex())
			fmt.Printf("From: %s\n", from.Hex())
			fmt.Printf("USDT Issue Amount: %f\n", actualAmount)

			// we only need one event
			os.Exit(0)
		}

		currentBlock = batchStart
	}
}
