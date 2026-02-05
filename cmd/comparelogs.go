package cmd

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"sort"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const (
	CompareChain1Flag = "chain-1"
	CompareChain2Flag = "chain-2"
	FromBlockFlag     = "from-block"
	ToBlockFlag       = "to-block"
	AddressFlag       = "address"
	TopicsFlag        = "topics"
	IgnoreOrderFlag   = "ignore-order"
	TimeoutFlag       = "timeout"
)

// MaxBlocksPerRequest is the maximum block span per FilterLogs call.
// A value of 300 means we query [from..to] where to-from+1 <= 300.
const MaxBlocksPerRequest uint64 = 300

var compareLogsCmd = &cobra.Command{
	Use:   "comparelogs",
	Short: "Fetch logs from two chains and compare if the results are the same",
	Run: func(cmd *cobra.Command, args []string) {
		chain1, _ := cmd.Flags().GetString(CompareChain1Flag)
		chain2, _ := cmd.Flags().GetString(CompareChain2Flag)
		fromBlock, _ := cmd.Flags().GetUint64(FromBlockFlag)
		toBlock, _ := cmd.Flags().GetUint64(ToBlockFlag)
		addrStr, _ := cmd.Flags().GetString(AddressFlag)
		topicsRaw, _ := cmd.Flags().GetStringSlice(TopicsFlag)
		ignoreOrder, _ := cmd.Flags().GetBool(IgnoreOrderFlag)
		timeout, _ := cmd.Flags().GetDuration(TimeoutFlag)

		if chain1 == "" || chain2 == "" {
			log.Error("Both --chain-1 and --chain-2 are required")
			return
		}
		if fromBlock == 0 {
			log.Error("--from-block is required (must be > 0)")
			return
		}
		if toBlock != 0 && toBlock < fromBlock {
			log.Errorf("--to-block (%d) < --from-block (%d)", toBlock, fromBlock)
			return
		}

		var addr *common.Address
		if strings.TrimSpace(addrStr) != "" {
			a := common.HexToAddress(addrStr)
			addr = &a
		}

		topics, err := parseTopics(topicsRaw)
		if err != nil {
			log.WithError(err).Error("Invalid --topics")
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		err = doCompareLogs(ctx, chain1, chain2, fromBlock, toBlock, addr, topics, ignoreOrder)
		if err != nil {
			log.WithError(err).Error("comparelogs failed")
			return
		}
	},
}

func init() {
	compareLogsCmd.Flags().String(CompareChain1Flag, "", "RPC endpoint for chain 1")
	compareLogsCmd.Flags().String(CompareChain2Flag, "", "RPC endpoint for chain 2")
	compareLogsCmd.Flags().Uint64(FromBlockFlag, 0, "Start block (inclusive)")
	compareLogsCmd.Flags().Uint64(ToBlockFlag, 0, "End block (inclusive). 0 means latest on each chain")
	compareLogsCmd.Flags().String(AddressFlag, "", "Contract address to filter (optional)")
	compareLogsCmd.Flags().StringSlice(TopicsFlag, nil, "Topics to filter. Each item is a comma-separated list of topic hashes for that position (OR). Example: --topics 0xddf...,0xabc... --topics 0x123...")
	compareLogsCmd.Flags().Bool(IgnoreOrderFlag, true, "Ignore log ordering differences")
	compareLogsCmd.Flags().Duration(TimeoutFlag, 30*time.Second, "Overall timeout")

	_ = compareLogsCmd.MarkFlagRequired(CompareChain1Flag)
	_ = compareLogsCmd.MarkFlagRequired(CompareChain2Flag)
	_ = compareLogsCmd.MarkFlagRequired(FromBlockFlag)

	rootCmd.AddCommand(compareLogsCmd)
}

func doCompareLogs(ctx context.Context, chain1, chain2 string, fromBlock, toBlock uint64, addr *common.Address, topics [][]common.Hash, ignoreOrder bool) error {
	c1, err := ethclient.DialContext(ctx, chain1)
	if err != nil {
		return fmt.Errorf("dial chain1: %w", err)
	}
	defer c1.Close()

	c2, err := ethclient.DialContext(ctx, chain2)
	if err != nil {
		return fmt.Errorf("dial chain2: %w", err)
	}
	defer c2.Close()

	to1 := toBlock
	to2 := toBlock
	if toBlock == 0 {
		b1, err := c1.BlockNumber(ctx)
		if err != nil {
			return fmt.Errorf("chain1 latest block: %w", err)
		}
		b2, err := c2.BlockNumber(ctx)
		if err != nil {
			return fmt.Errorf("chain2 latest block: %w", err)
		}
		to1, to2 = b1, b2
		if to1 != to2 {
			log.Warnf("Chains latest blocks differ: chain1=%d chain2=%d; comparing each chain against its own latest", to1, to2)
		}
	}

	if fromBlock > to1 || fromBlock > to2 {
		return fmt.Errorf("from-block (%d) is greater than chain latest (chain1=%d chain2=%d)", fromBlock, to1, to2)
	}

	end := to1
	if to2 < end {
		end = to2
	}

	var (
		totalRanges     uint64
		mismatchRanges  uint64
		okRanges        uint64
		totalLogsChain1 uint64
		totalLogsChain2 uint64
	)

	for start := fromBlock; start <= end; {
		windowEnd := start + MaxBlocksPerRequest - 1
		if windowEnd > end {
			windowEnd = end
		}
		totalRanges++

		q1 := ethereum.FilterQuery{FromBlock: uint64ToBig(start), ToBlock: uint64ToBig(windowEnd), Topics: topics}
		q2 := ethereum.FilterQuery{FromBlock: uint64ToBig(start), ToBlock: uint64ToBig(windowEnd), Topics: topics}
		if addr != nil {
			q1.Addresses = []common.Address{*addr}
			q2.Addresses = []common.Address{*addr}
		}

		logs1, err := c1.FilterLogs(ctx, q1)
		if err != nil {
			return fmt.Errorf("chain1 FilterLogs [%d..%d]: %w", start, windowEnd, err)
		}
		logs2, err := c2.FilterLogs(ctx, q2)
		if err != nil {
			return fmt.Errorf("chain2 FilterLogs [%d..%d]: %w", start, windowEnd, err)
		}

		totalLogsChain1 += uint64(len(logs1))
		totalLogsChain2 += uint64(len(logs2))

		s1 := canonicalizeLogs(logs1, ignoreOrder)
		s2 := canonicalizeLogs(logs2, ignoreOrder)

		h1 := sha256.Sum256([]byte(strings.Join(s1, "\n")))
		h2 := sha256.Sum256([]byte(strings.Join(s2, "\n")))

		if h1 != h2 {
			mismatchRanges++
			log.Errorf("[range %d..%d] Logs differ (sha256 chain1=%s chain2=%s) count(chain1)=%d count(chain2)=%d",
				start, windowEnd, hex.EncodeToString(h1[:]), hex.EncodeToString(h2[:]), len(s1), len(s2))

			// show small diff info (best-effort)
			min := len(s1)
			if len(s2) < min {
				min = len(s2)
			}
			idx := -1
			for i := 0; i < min; i++ {
				if s1[i] != s2[i] {
					idx = i
					break
				}
			}
			if idx != -1 {
				log.Errorf("[range %d..%d] First mismatch at index %d:\nchain1: %s\nchain2: %s", start, windowEnd, idx, s1[idx], s2[idx])
			} else if len(s1) != len(s2) {
				log.Errorf("[range %d..%d] Log count differs: chain1=%d chain2=%d", start, windowEnd, len(s1), len(s2))
			}
		} else {
			okRanges++
			log.Infof("[range %d..%d] Logs equal. count=%d sha256=%s", start, windowEnd, len(s1), hex.EncodeToString(h1[:]))
		}

		if windowEnd == end {
			break
		}
		start = windowEnd + 1
	}

	log.Infof("comparelogs summary: ranges=%d ok=%d mismatch=%d totalLogs(chain1)=%d totalLogs(chain2)=%d", totalRanges, okRanges, mismatchRanges, totalLogsChain1, totalLogsChain2)
	if mismatchRanges > 0 {
		return fmt.Errorf("found %d mismatching ranges", mismatchRanges)
	}
	return nil
}

func canonicalizeLogs(in []types.Log, ignoreOrder bool) []string {
	out := make([]string, 0, len(in))
	for _, l := range in {
		topics := make([]string, 0, len(l.Topics))
		for _, t := range l.Topics {
			topics = append(topics, t.Hex())
		}
		// Include Data as it's part of the event payload; without it we can miss real differences.
		out = append(out, fmt.Sprintf("%d|%s|%d|%s|%d|%s|%s|%s",
			l.BlockNumber,
			l.BlockHash.Hex(),
			l.TxIndex,
			l.TxHash.Hex(),
			l.Index,
			l.Address.Hex(),
			hex.EncodeToString(l.Data),
			strings.Join(topics, ","),
		))
	}
	if ignoreOrder {
		sort.Strings(out)
	}
	return out
}

func parseTopics(raw []string) ([][]common.Hash, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	res := make([][]common.Hash, 0, len(raw))
	for _, pos := range raw {
		pos = strings.TrimSpace(pos)
		if pos == "" {
			res = append(res, nil)
			continue
		}
		parts := strings.Split(pos, ",")
		vals := make([]common.Hash, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			if !isHexTopicHash(p) {
				return nil, fmt.Errorf("not a topic hash: %s", p)
			}
			vals = append(vals, common.HexToHash(p))
		}
		res = append(res, vals)
	}
	return res, nil
}

func isHexTopicHash(s string) bool {
	// Accept 0x + 64 hex chars
	if len(s) != 66 {
		return false
	}
	if !strings.HasPrefix(s, "0x") && !strings.HasPrefix(s, "0X") {
		return false
	}
	_, err := hex.DecodeString(s[2:])
	return err == nil
}

func uint64ToBig(v uint64) *big.Int {
	// local helper to avoid importing math/big all over call sites
	return new(big.Int).SetUint64(v)
}
