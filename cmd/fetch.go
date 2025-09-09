package cmd

import (
	"context"
	"github.com/ethereum/go-ethereum/ethclient"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"math/big"
	"time"
)

const (
	FetchUrlFlag   = "fetch-url"
	TargetUrlFlag  = "target-url"
	BeginBlockFlag = "begin-block"
)

// fetchCmd represents the fetch command
var fetchCmd = &cobra.Command{
	Use:   "fetch",
	Short: "Fetch blocks and transactions from source and send to target",
	Run: func(cmd *cobra.Command, args []string) {
		fetchUrl, _ := cmd.Flags().GetString(FetchUrlFlag)
		if fetchUrl == "" {
			log.Errorf("Fetch URL is required")
			return
		}
		targetUrl, _ := cmd.Flags().GetString(TargetUrlFlag)
		if targetUrl == "" {
			log.Errorf("Target URL is required")
			return
		}
		beginBlock, _ := cmd.Flags().GetUint64(BeginBlockFlag)
		if beginBlock == 0 {
			log.Errorf("Begin block is required")
			return
		}
		doFetch(fetchUrl, targetUrl, beginBlock)
	},
}

func doFetch(fetchUrl, targetUrl string, beginBlock uint64) {
	// Connect to source chain
	sourceClient, err := ethclient.Dial(fetchUrl)
	if err != nil {
		log.Errorf("Failed to connect to source chain: %s", err)
		return
	}
	defer sourceClient.Close()

	endBlock, err := sourceClient.BlockNumber(context.Background())
	if err != nil {
		log.Errorf("Failed to get latest block number from source chain: %s", err)
		return
	}

	if endBlock < beginBlock {
		log.Errorf("end block %d < begin block %d", endBlock, beginBlock)
		return
	}

	// Connect to target chain
	targetClient, err := ethclient.Dial(targetUrl)
	if err != nil {
		log.Errorf("Failed to connect to target chain: %s", err)
		return
	}
	defer targetClient.Close()

	ctx := context.TODO()
	currentBlock := beginBlock

	for currentBlock = beginBlock; currentBlock < endBlock; {
		// Fetch block from source
		block, err := sourceClient.BlockByNumber(ctx, big.NewInt(int64(currentBlock)))
		if err != nil {
			log.Errorf("Failed to fetch block %d: %s", currentBlock, err)
			return
		}
		txs := block.Transactions()
		if len(txs) == 0 {
			currentBlock++
			continue
		}

		log.Infof("Processing block %d with %d transactions, remain block %d.", currentBlock, len(block.Transactions()), endBlock-currentBlock)

		// Process each transaction in the block
		for _, tx := range block.Transactions() {
			// Send transaction to target chain
			err := targetClient.SendTransaction(ctx, tx)
			if err != nil {
				log.Errorf("Failed to send transaction %s: %s", tx.Hash().Hex(), err)
				continue
			}
			// wait tx receipt.
			for {
				receipt, err := targetClient.TransactionReceipt(ctx, tx.Hash())
				if err == nil {
					log.Infof("transaction %s result : %d", tx.Hash().Hex(), receipt.Status)
					break
				}
				time.Sleep(1 * time.Second)
			}
			log.Infof("Successfully sent transaction %s", tx.Hash().Hex())
		}

		// Add a small delay to avoid overwhelming the networks
		time.Sleep(100 * time.Millisecond)
		currentBlock++
	}
}

func init() {
	fetchCmd.Flags().String(FetchUrlFlag, "", "URL of the source chain to fetch from")
	fetchCmd.Flags().String(TargetUrlFlag, "", "URL of the target chain to send to")
	fetchCmd.Flags().Uint64(BeginBlockFlag, 0, "Block number to start fetching from")
	fetchCmd.MarkFlagRequired(FetchUrlFlag)
	fetchCmd.MarkFlagRequired(TargetUrlFlag)
	fetchCmd.MarkFlagRequired(BeginBlockFlag)

	rootCmd.AddCommand(fetchCmd)
}
