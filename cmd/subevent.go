package cmd

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/xueqianLu/ethtools/erc20"
)

const (
	ChainEndpointFlag   = "chain-endpoint"
	ContractAddressFlag = "token-address"
)

// newBlockCmd represents the new block command
var newBlockCmd = &cobra.Command{
	Use:   "newblock",
	Short: "Listen for new blocks on a chain",
	Run: func(cmd *cobra.Command, args []string) {
		chainEndpoint, _ := cmd.Flags().GetString(ChainEndpointFlag)
		if chainEndpoint == "" {
			log.Errorf("Chain endpoint is required")
			return
		}
		listenForNewBlocks(chainEndpoint)
	},
}

// transferEventCmd represents the transfer event command
var transferEventCmd = &cobra.Command{
	Use:   "transfer",
	Short: "Listen for Transfer events from an ERC-20 contract",
	Run: func(cmd *cobra.Command, args []string) {
		chainEndpoint, _ := cmd.Flags().GetString(ChainEndpointFlag)
		if chainEndpoint == "" {
			log.Errorf("Chain endpoint is required")
			return
		}
		contractAddressStr, _ := cmd.Flags().GetString(ContractAddressFlag)
		if contractAddressStr == "" {
			log.Errorf("Contract address is required")
			return
		}

		contractAddress := common.HexToAddress(contractAddressStr)
		listenForTransferEvents(chainEndpoint, contractAddress)
	},
}

func init() {
	rootCmd.AddCommand(newBlockCmd)
	rootCmd.AddCommand(transferEventCmd)

	newBlockCmd.Flags().String(ChainEndpointFlag, "", "Chain endpoint URL")
	transferEventCmd.Flags().String(ChainEndpointFlag, "", "Chain endpoint URL")
	transferEventCmd.Flags().String(ContractAddressFlag, "", "ERC-20 contract address")
}

func listenForNewBlocks(chainEndpoint string) {
	client, err := ethclient.Dial(chainEndpoint)
	if err != nil {
		log.Fatalf("Failed to connect to the Ethereum client: %v", err)
	}

	headers := make(chan *types.Header)
	sub, err := client.SubscribeNewHead(context.Background(), headers)
	if err != nil {
		log.Fatalf("Failed to subscribe to new head events: %v", err)
	}

	for {
		select {
		case err := <-sub.Err():
			log.Fatal(err)
		case header := <-headers:
			fmt.Println("New block mined:", header.Number.Uint64())
			block, err := client.BlockByNumber(context.Background(), header.Number)
			if err != nil {
				log.Errorf("Failed to get block: %v", err)
				continue
			}
			fmt.Println("Block Number:", block.Number().String())
			fmt.Println("Block Hash:", block.Hash().Hex())
			fmt.Println("Block Timestamp:", block.Time())
			fmt.Println("Number of Transactions:", len(block.Transactions()))
		}
	}
}

func listenForTransferEvents(chainEndpoint string, contractAddress common.Address) {
	client, err := ethclient.Dial(chainEndpoint)
	if err != nil {
		log.Fatalf("Failed to connect to the Ethereum client: %v", err)
	}
	contract, _ := erc20.NewErc20(contractAddress, client)
	// Create a channel to receive events
	events := make(chan *erc20.Erc20Transfer)
	//contract.WatchTransfer()
	// Subscribe to Transfer events
	opts := &bind.WatchOpts{Context: context.Background()}
	subscription, err := contract.WatchTransfer(opts, events, nil, nil)
	if err != nil {
		log.Fatalf("Failed to subscribe to Transfer events: %v", err)
		return
	}
	defer subscription.Unsubscribe()

	// Listen for events
	for {
		select {
		case err := <-subscription.Err():
			log.Fatal(err)
		case event := <-events:
			fmt.Println("Transfer Event:")
			fmt.Println("Block Number:", event.Raw.BlockNumber)
			fmt.Println("Transaction Hash:", event.Raw.TxHash.Hex())
			fmt.Println("Contract Address:", event.Raw.Address.Hex())
			fmt.Println("From:", event.From.Hex())
			fmt.Println("To:", event.To.Hex())
			fmt.Println("Value:", event.Value.String())
		}
	}

	//query := ethereum.FilterQuery{
	//	Addresses: []common.Address{contractAddress},
	//	Topics: [][]common.Hash{
	//		{common.HexToHash("0xddf252ad1be2b89b69c2b068fc378daa952ba778895dca41f1506ef3ca7a739b")}, // Transfer event signature
	//	},
	//
	//	//Topics:    [][]common.Hash{{common.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba778895dca41f1506ef3ca7a739b")}}, // Transfer event signature
	//}
	//
	//logs := make(chan types.Log)
	//sub, err := client.SubscribeFilterLogs(context.Background(), query, logs)
	//if err != nil {
	//	log.Fatalf("Failed to subscribe to Transfer events: %v", err)
	//}
	//
	//for {
	//	select {
	//	case err := <-sub.Err():
	//		log.Fatal(err)
	//	case eventLog := <-logs:
	//		fmt.Println("Contract Event:", eventLog.Topics[0].Hex())
	//		fmt.Println("Block Number:", eventLog.BlockNumber)
	//		fmt.Println("Transaction Hash:", eventLog.TxHash.Hex())
	//		fmt.Println("Contract Address:", eventLog.Address.Hex())
	//
	//		// Parse event data
	//		from := common.HexToAddress(eventLog.Topics[1].Hex())
	//		to := common.HexToAddress(eventLog.Topics[2].Hex())
	//		value := new(big.Int).SetBytes(eventLog.Data)
	//
	//		fmt.Println("From:", from.Hex())
	//		fmt.Println("To:", to.Hex())
	//		fmt.Println("Value:", value.String())
	//	}
	//}
}
