package cmd

import (
	"context"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"math/big"
	"sync"
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

type TxInfo struct {
	originBlock uint64
	idx         int
	tx          *types.Transaction
}

func fetchTx(sourceClient *ethclient.Client, beginBlock uint64, endBlock uint64, txsCh chan TxInfo) error {
	ctx := context.Background()
	for currentBlock := beginBlock; currentBlock < endBlock; {
		// Fetch block from source
		block, err := sourceClient.BlockByNumber(ctx, big.NewInt(int64(currentBlock)))
		if err != nil {
			log.Errorf("Failed to fetch block %d: %s", currentBlock, err)
			time.Sleep(1 * time.Second)
			continue
		}
		txs := block.Transactions()
		if len(txs) == 0 {
			currentBlock++
			continue
		}

		log.Infof("Processing block %d with %d transactions, remain block %d.", currentBlock, len(block.Transactions()), endBlock-currentBlock)

		for i, tx := range txs {
			txsCh <- TxInfo{
				originBlock: block.Number().Uint64(),
				idx:         i,
				tx:          tx,
			}
		}
		currentBlock++
		time.Sleep(50 * time.Millisecond)
	}
	close(txsCh)
	return nil
}

func batchBroadCast(targetClient *ethclient.Client, batch []TxInfo) error {
	ctx := context.Background()
	wg := sync.WaitGroup{}
	wait := make([]TxInfo, 0)
	for _, txInfo := range batch {
		err := targetClient.SendTransaction(ctx, txInfo.tx)
		if err != nil {
			log.Errorf("Failed to send transaction %s: %s", txInfo.tx.Hash().Hex(), err)
			continue
		}
		wait = append(wait, txInfo)
	}
	for _, txInfo := range wait {
		wg.Add(1)
		go func(info TxInfo) {
			defer wg.Done()
			receipt, err := targetClient.TransactionReceipt(ctx, info.tx.Hash())
			for err != nil || receipt == nil {
				time.Sleep(1 * time.Second)
				receipt, err = targetClient.TransactionReceipt(ctx, info.tx.Hash())
			}
			if err != nil {
				log.Errorf("Failed to get receipt for transaction %d:%s: %s", info.originBlock, info.tx.Hash().Hex(), err)
				return
			}
			log.Infof("Transaction %d:%s mined in block %d", info.originBlock, info.tx.Hash().Hex(), receipt.BlockNumber.Uint64())
		}(txInfo)
	}
	wg.Wait()

	return nil
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

	txCh := make(chan TxInfo, 1000)
	go fetchTx(sourceClient, beginBlock, endBlock, txCh)
	maxBatchSize := 20

	batch := make([]TxInfo, 0, maxBatchSize)

	finish := false
	for finish {
		select {
		case tx, ok := <-txCh:
			if !ok {
				finish = true
				break
			}
			batch = append(batch, tx)
			if len(batch) >= maxBatchSize {
				err := batchBroadCast(targetClient, batch)
				if err != nil {
					log.Errorf("Failed to broadcast batch: %s", err)
				}
				batch = batch[:0]
			}
		}
	}
	if err := batchBroadCast(targetClient, batch); err != nil {
		log.Errorf("Failed to broadcast final batch: %s", err)
	} else {
		log.Infof("Successfully fetched all transactions.", len(batch))
	}
	return
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
