package main

import (
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/xueqianLu/ethtools/erc20"
	"io/ioutil"
	"log"
	"math/big"
	"strings"
)

var (
	defaultgas, _      = new(big.Int).SetString("45000", 10)
	defaultgasprice, _ = new(big.Int).SetString("800000000000", 10)
)

func paddingHexString(str string, length int) string {
	var ret = str
	for len(ret) < length {
		ret = "0" + ret
	}
	return ret
}

func tokenInfo(coinAddr string, client *ethclient.Client) error {
	tokenAddress := common.HexToAddress(coinAddr)
	defaultOpt := &bind.CallOpts{}

	coin, err := erc20.NewErc20(tokenAddress, client)
	if err != nil {
		//log.Println("newErc20 failed")
		return err
	}

	name, err := coin.Name(defaultOpt)
	if err != nil {
		//log.Println("coin name failed")
		return err
	}

	symbol, err := coin.Symbol(defaultOpt)
	if err != nil {
		//log.Println("coin symbol failed")
		return err
	}

	decimal, err := coin.Decimals(defaultOpt)
	if err != nil {
		//log.Println("coin decimal failed")
		return err
	}
	log.Println("coin address        :", coinAddr)
	log.Println("coin name           :", name)
	log.Println("coin symbol         :", symbol)
	log.Println("coin decimal        :", decimal)
	return nil
}

type AccountData struct {
	Addr  string `json:"addr"`
	Value string `json:"value"`
}

type AccountInfo struct {
	Info []AccountData `json:"info"`
}

func main() {
	url := flag.String("u", "http://127.0.0.1:8545", "rpc url")
	privkey := flag.String("priv", "", "sender private key")
	coinAddr := flag.String("erc20", "", "coin contract address")
	datafile := flag.String("f", "", "the file name that contain all accounts info")

	flag.Parse()

	client := NewHttpClient(*url)
	chainId := client.ChainID()

	privateKey, err := crypto.HexToECDSA(*privkey)
	if err != nil {
		log.Fatal(err)
	}

	sender, err := getAddrFromPrivk(privateKey)
	if err != nil {
		log.Fatal(err)
	}
	nonce := client.GetNonce(sender.String())

	fmt.Println("from address", sender.String())
	fmt.Println("NonceAt", nonce)

	if *datafile == "" {
		log.Println("data file name is empty")
		return
	}

	infos := AccountInfo{}
	if content, err := ioutil.ReadFile(*datafile); err != nil {
		log.Println("read file failed, err:", err.Error())
		return
	} else {
		if e := json.Unmarshal(content, &infos); e != nil {
			log.Println("unmarshal data content failed, err:", e.Error())
			return
		}
	}

	if len(*coinAddr) > 1 { // erc20 token
		tokenAddress := common.HexToAddress(*coinAddr)
		for _, user := range infos.Info {
			to := common.HexToAddress(user.Addr)
			amount, ok := new(big.Int).SetString(user.Value, 10)
			parsed, err := abi.JSON(strings.NewReader(erc20.Erc20ABI))
			if err != nil {
				log.Println("abi json failed.")
				log.Fatal(err)
			}
			data, err := parsed.Pack("transfer", to, amount)
			if !ok {
				log.Fatal(errors.New("parse balance failed"))
			}

			tx := types.NewTransaction(nonce, tokenAddress, big.NewInt(0), defaultgas.Uint64(), defaultgasprice, data)
			signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainId), privateKey)
			if err != nil {
				log.Fatal(err)
			}

			txhash, err := client.SendSignedTx(signedTx)
			if err != nil {
				log.Fatal(err)
			}

			fmt.Printf("tx sent to %s, txhash: %s\n", user.Addr, txhash)
			nonce++
		}
	} else { // mainnet coin.
		for _, user := range infos.Info {
			to := common.HexToAddress(user.Addr)
			amount, _ := new(big.Int).SetString(user.Value, 10)
			tx := types.NewTransaction(nonce, to, amount, defaultgas.Uint64(), defaultgasprice, []byte{})
			signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainId), privateKey)
			if err != nil {
				log.Fatal(err)
			}
			txhash, err := client.SendSignedTx(signedTx)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Printf("tx sent to %s, txhash: %s\n", user.Addr, txhash)
			nonce++
		}
	}
}

func getAddrFromPrivk(priv *ecdsa.PrivateKey) (common.Address, error) {
	publicKey := priv.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return common.Address{}, errors.New("cannot assert type: publicKey is not of type *ecdsa.PublicKey")
	}

	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	return fromAddress, nil
}
