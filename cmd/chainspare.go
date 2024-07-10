package cmd

import (
	"context"
	"encoding/json"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
)

const (
	Chain1Flag      = "chain-1"
	Chain2Flag      = "chain-2"
	AccountFileFlag = "account-file"
)

// chainPareCmd represents the base command when called without any subcommands
var chainPareCmd = &cobra.Command{
	Use:   "cspare",
	Short: "Compare accounts in two chains",
	Run: func(cmd *cobra.Command, args []string) {
		// parse args to flag
		chain1, _ := cmd.Flags().GetString(Chain1Flag)
		if chain1 == "" {
			log.Errorf("Chain1 is required")
			return
		}
		chain2, _ := cmd.Flags().GetString(Chain2Flag)
		if chain2 == "" {
			log.Errorf("Chain2 is required")
			return
		}
		accountFile, _ := cmd.Flags().GetString(AccountFileFlag)
		if accountFile == "" {
			log.Errorf("Account file is required")
			return
		}
		doCompare(chain1, chain2, accountFile)
	},
}

func doCompare(chain1, chain2, accountFile string) {
	// do compare
	clientChain1, err := ethclient.Dial(chain1)
	if err != nil {
		log.Errorf("Failed to connect to the first chain: %s", err)
		return
	}
	clientChain2, err := ethclient.Dial(chain2)
	if err != nil {
		log.Errorf("Failed to connect to the second chain: %s", err)
		return
	}
	addresslist := make([]string, 0)
	if data, err := os.ReadFile(accountFile); err != nil {
		log.Errorf("Failed to read account file: %s", err)
		return
	} else {
		err = json.Unmarshal(data, &addresslist)
		if err != nil {
			log.Errorf("Failed to parse account file: %s", err)
			return
		}
	}
	//height := big.NewInt(610013)
	ctx := context.TODO()
	for _, address := range addresslist {
		addr := common.HexToAddress(address)
		balance1, err := clientChain1.BalanceAt(ctx, addr, nil)
		if err != nil {
			log.Errorf("Failed to get balance from the first chain: %s", err)
			return
		}
		balance2, err := clientChain2.BalanceAt(ctx, addr, nil)
		if err != nil {
			log.Errorf("Failed to get balance from the second chain: %s", err)
			return
		}
		if balance1.Cmp(balance2) != 0 {
			log.Errorf("Balance not equal for address: %s, Chain1: %s, Chain2: %s", address, balance1.Text(10), balance2.Text(10))
		} else {
			log.Info("Balance equal for address: ", address)
		}
	}
}
