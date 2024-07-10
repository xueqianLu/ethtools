package utils

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rlp"
	"io/ioutil"
	"math/big"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	defaultGas, _      = new(big.Int).SetString("100000", 10)
	defaultGasPrice, _ = new(big.Int).SetString("1000000000", 10)
)

type AccountNonce struct {
	Addr  string
	Nonce uint64
}

/*
[{"jsonrpc":"2.0","id":68,"result":{"raw":"0xf86780850430e2340083015f90949d882e29357b8fda9ad232760ab8ea763c7484908203e88026a008e1f63b9d2ceb607216d9b4f8ff7662c797ea5da84c59611519d7bf71fafc5ba07dd7f052492ed54194881accda1c22a104fd8191d156fda0cf3f9546277a2d98","tx":{"nonce":"0x0","gasPrice":"0x430e23400","gas":"0x15f90","to":"0x9d882e29357b8fda9ad232760ab8ea763c748490","value":"0x3e8","input":"0x","exdata":{"txversion":0,"txtype":0,"vmversion":0,"txflag":0,"reserve":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]},"v":"0x26","r":"0x8e1f63b9d2ceb607216d9b4f8ff7662c797ea5da84c59611519d7bf71fafc5b","s":"0x7dd7f052492ed54194881accda1c22a104fd8191d156fda0cf3f9546277a2d98","hash":"0x54e20f016901ca4e074e75ff17ad2d7795f08bc444c7af60062748a9056e9a2e"}}}]
*/

type TxInfo struct {
	Nonce string `json:"nonce"`
	To    string `json:"to"`
	Value string `json:"value"`
}

type SignTxResult struct {
	Raw    string `json:"raw"`
	Txinfo TxInfo `json:"tx"`
}

func (result SignTxResult) InValid() bool {
	return result.Raw == ""
}

func (result SignTxResult) String() string {
	var s = "raw:" + result.Raw + "\r\n"
	s += "nonce:" + result.Txinfo.Nonce + "\r\n"
	return s
}

type TxArgs struct {
	From     string `json:"from"`
	To       string `json:"to"`
	Value    string `json:"value"`
	Nonce    string `json:"nonce"`
	Gas      string `json:"gas"`
	GasPrice string `json:"gasPrice"`
	Data     string `json:"data"`
}

type SendSignData struct {
	Txs     []TxArgs `json:"params"`
	Id      int      `json:"id"`
	Jsonrpc string   `json:"jsonrpc"`
	Method  string   `json:"method"`
}

type RespSignData struct {
	Jsonrpc string       `json:"jsonrpc"`
	Id      int          `json:"id"`
	Signtx  SignTxResult `json:"result"`
}

type SendRawData struct {
	Txs     []string `json:"params"`
	Id      int      `json:"id"`
	Jsonrpc string   `json:"jsonrpc"`
	Method  string   `json:"method"`
}

type RespRawData struct {
	Jsonrpc string `json:"jsonrpc"`
	Id      int    `json:"id"`
	Txhash  string `json:"result"`
}

type GetAccounts struct {
	Params  []string `json:"params"`
	Id      int      `json:"id"`
	Jsonrpc string   `json:"jsonrpc"`
	Method  string   `json:"method"`
}

type RespAccounts struct {
	Jsonrpc  string   `json:"jsonrpc"`
	Id       int      `json:"id"`
	Accounts []string `json:"result"`
}

type GetAccountNonce struct {
	Params  []string `json:"params"`
	Id      int      `json:"id"`
	Jsonrpc string   `json:"jsonrpc"`
	Method  string   `json:"method"`
}

type RespAccountNonce struct {
	Jsonrpc string `json:"jsonrpc"`
	Id      int    `json:"id"`
	Nonce   string `json:"result"`
}

/*
[{"jsonrpc":"2.0","id":68,"error":{"code":-32000,"message":"authentication needed: password or unlock"}}]
*/
type ErrMsg struct {
	Msg string `json:"message"`
}
type RespErrData struct {
	Jsonrpc string `json:"jsonrpc"`
	Id      int    `json:"id"`
	Err     ErrMsg `json:"error"`
}

var (
	rpcid = 10
	mux   sync.Mutex
)

func getRPCId() int {
	mux.Lock()
	defer mux.Unlock()
	rpcid++
	return rpcid
}

func buildGetAccounts() string {
	id := getRPCId()
	datas := make([]interface{}, 0)

	ss := GetAccounts{Method: "eth_accounts", Jsonrpc: "2.0", Id: id}
	datas = append(datas, ss)

	body, _ := json.Marshal(datas)
	return string(body)
}

func doGetAccounts(url string, body string, client *http.Client) ([]string, error) {
	req, _ := http.NewRequest("POST", url, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	//req.Header.Set("Connection", "close")

	fmt.Println("doGetAccounts >>>>>>:", body)

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Http Send Error:", err)
		return nil, err
	}
	defer resp.Body.Close()
	content, err := ioutil.ReadAll(resp.Body)
	fmt.Println("doGetAccounts <<<<<<:", string(content))

	var results = make([]RespAccounts, 0)
	//err = json.NewDecoder(resp.Body).Decode(&results)
	err = json.Unmarshal(content, &results)
	if err != nil {
		return nil, err
	}
	accounts := make([]string, len(results[0].Accounts))
	copy(accounts, results[0].Accounts)
	return accounts, nil
}

func buildGetAccountNonce(addr string) string {
	id := getRPCId()
	datas := make([]interface{}, 0)

	ss := GetAccountNonce{Method: "eth_getTransactionCount", Jsonrpc: "2.0", Id: id}
	ss.Params = append(ss.Params, addr)
	ss.Params = append(ss.Params, "latest")

	datas = append(datas, ss)

	body, _ := json.Marshal(datas)
	return string(body)
}

func doGetAccountNonce(url string, body string, client *http.Client) (uint64, error) {
	req, _ := http.NewRequest("POST", url, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	//req.Header.Set("Connection", "close")

	//fmt.Println("doGetAccountNonce >>>>>>:", body)

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Http Send Error:", err)
		return 0, err
	}
	defer resp.Body.Close()
	content, err := ioutil.ReadAll(resp.Body)
	//fmt.Println("doGetAccountNonce <<<<<<:", string(content))

	var results = make([]RespAccountNonce, 0)
	//err = json.NewDecoder(resp.Body).Decode(&results)
	err = json.Unmarshal(content, &results)
	if err != nil {
		return 0, err
	}
	nonceStr := results[0].Nonce
	nonce, e := strconv.ParseUint(nonceStr[2:], 16, 64)
	if e != nil {
		fmt.Println("parse nonceStr error:", err)
		return 0, e
	}
	return nonce, nil
}

func buildSendTxString(from string, to string, value *big.Int, nonce uint64, num int) string {
	datas := make([]interface{}, 0)
	for i := 0; i < num; i++ {
		nid := getRPCId()
		ss := SendSignData{Method: "eth_sendTransaction", Jsonrpc: "2.0", Id: nid}
		hexval := value.Text(16)
		hexnonce := strconv.FormatUint(nonce+uint64(i), 16)
		_tx := TxArgs{From: from, To: to, Value: "0x" + hexval, Nonce: "0x" + hexnonce,
			Gas: "0x" + defaultGas.Text(16), GasPrice: "0x" + defaultGasPrice.Text(16)}

		ss.Txs = append(ss.Txs, _tx)
		datas = append(datas, ss)
	}

	body, _ := json.Marshal(datas)
	return string(body)
}

func doSendTx(url string, body string, client *http.Client) error {
	req, _ := http.NewRequest("POST", url, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	//req.Header.Set("Connection", "close")

	//	fmt.Println("doSendTx >>>>>>:",body)

	_, err := client.Do(req)
	if err != nil {
		fmt.Println("Http Send Error:", err)
		return err
	}
	//	defer resp.Body.Close()
	//	content, err := ioutil.ReadAll(resp.Body)
	//	fmt.Println("doSendTx <<<<<<:", string(content))
	return nil
}

func buildSignString(from string, to string, value *big.Int, nonce uint64, data []byte) string {
	id := getRPCId()
	datas := make([]interface{}, 0)
	ss := SendSignData{Method: "eth_signTransaction", Jsonrpc: "2.0", Id: id}
	_tx := TxArgs{From: from, To: to, Gas: "0x" + defaultGas.Text(16), GasPrice: "0x" + defaultGasPrice.Text(16)}
	if value.Uint64() > 0 {
		_tx.Value = "0x" + value.Text(16)
	}
	if nonce != 0 {
		_tx.Nonce = "0x" + strconv.FormatUint(nonce, 16)
	}
	if len(data) > 0 {
		_tx.Data = "0x" + hex.EncodeToString(data)
	}

	ss.Txs = append(ss.Txs, _tx)
	datas = append(datas, ss)

	body, _ := json.Marshal(datas)
	return string(body)
}

/*
[{"jsonrpc":"2.0","id":68,"result":{"raw":"0xf86780850430e2340083015f90949d882e29357b8fda9ad232760ab8ea763c7484908203e88026a008e1f63b9d2ceb607216d9b4f8ff7662c797ea5da84c59611519d7bf71fafc5ba07dd7f052492ed54194881accda1c22a104fd8191d156fda0cf3f9546277a2d98","tx":{"nonce":"0x0","gasPrice":"0x430e23400","gas":"0x15f90","to":"0x9d882e29357b8fda9ad232760ab8ea763c748490","value":"0x3e8","input":"0x","exdata":{"txversion":0,"txtype":0,"vmversion":0,"txflag":0,"reserve":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]},"v":"0x26","r":"0x8e1f63b9d2ceb607216d9b4f8ff7662c797ea5da84c59611519d7bf71fafc5b","s":"0x7dd7f052492ed54194881accda1c22a104fd8191d156fda0cf3f9546277a2d98","hash":"0x54e20f016901ca4e074e75ff17ad2d7795f08bc444c7af60062748a9056e9a2e"}}}]
*/
func doSignTx(url string, body string, client *http.Client) (*SignTxResult, error) {
	req, _ := http.NewRequest("POST", url, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	//req.Header.Set("Connection", "close")

	//fmt.Println("doSignTx >>>>>>:", body)

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Http Send Error:", err)
		return nil, err
	}
	defer resp.Body.Close()
	content, err := ioutil.ReadAll(resp.Body)
	//fmt.Println("doSignTx <<<<<<:", string(content))

	var results = make([]RespSignData, 0)
	//err = json.NewDecoder(resp.Body).Decode(&results)
	err = json.Unmarshal(content, &results)
	if err != nil {
		return nil, err
	}
	return &results[0].Signtx, nil
}

func buildSendRawString(raw string) string {
	id := getRPCId()
	datas := make([]interface{}, 0)
	ss := SendRawData{Method: "eth_sendRawTransaction", Jsonrpc: "2.0", Id: id, Txs: make([]string, 0)}
	ss.Txs = append(ss.Txs, raw)
	datas = append(datas, ss)

	body, _ := json.Marshal(datas)
	return string(body)
}

func doSendRaw(url string, body string, client *http.Client) ([]string, error) {
	req, err := http.NewRequest("POST", url, strings.NewReader(body))
	if err != nil {
		fmt.Println("new request failed, error: ", err)
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	//req.Header.Set("Connection", "close")

	//fmt.Println("doSendRaw >>>>>>:",string(body))
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Http Send Error:", err)
		return nil, err
	}
	defer resp.Body.Close()

	content, err := ioutil.ReadAll(resp.Body)
	//fmt.Println("doSendRaw <<<<<<:", string(content))

	var results = make([]RespRawData, 0)
	var txhashs = make([]string, 0)
	//err = json.NewDecoder(resp.Body).Decode(&results)
	err = json.Unmarshal(content, &results)
	if err != nil {
		return nil, err
	}
	for _, result := range results {
		txhashs = append(txhashs, result.Txhash)
	}

	return txhashs, nil
}

func parseUint(str string) uint64 {
	//println("str=",str)
	var nstr string
	if strings.Compare(str[0:2], "0x") == 0 {
		if len(str) == 2 {
			return 0
		} else {
			nstr = str[2:len(str)]
		}
	} else {
		nstr = str
	}
	s, _ := strconv.ParseUint(nstr, 16, 64)
	return s
}

const (
	MaxIdleConns        int = 100
	MaxIdleConnsPerHost int = 100
	IdleConnTimeout     int = 40
)

type HttpClient struct {
	c   *http.Client
	eth *ethclient.Client
	url string
}

func CreateHttpClient(url string) *HttpClient {
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			MaxIdleConns:        MaxIdleConns,
			MaxIdleConnsPerHost: MaxIdleConnsPerHost,
			IdleConnTimeout:     time.Duration(IdleConnTimeout) * time.Second,
		},

		Timeout: 20 * time.Second,
	}
	ethclient, _ := ethclient.Dial(url)
	return &HttpClient{client, ethclient, url}
}

func NewHttpClient(url string) *HttpClient {
	return CreateHttpClient(url)
}

func (c *HttpClient) GetNonce(addr string) uint64 {
	address := common.HexToAddress(addr)
	nonce, _ := c.eth.NonceAt(context.Background(), address, nil)
	return nonce
}

func (c *HttpClient) ChainID() *big.Int {
	chainId, err := c.eth.ChainID(context.Background())
	if err != nil {
		return nil
	}
	return chainId
}

func (c *HttpClient) GasPrice() *big.Int {
	price, err := c.eth.SuggestGasPrice(context.Background())
	if err != nil {
		return big.NewInt(10)
	}
	return price
}

func (c *HttpClient) SendTx(from, to common.Address, nonce uint64, value string, data []byte) (string, error) {
	val, _ := big.NewInt(0).SetString(value, 10)
	body := buildSignString(from.String(), to.String(), val, nonce, data)
	signedTx, err := doSignTx(c.url, body, c.c)

	if err != nil {
		return "", err
	}
	body = buildSendRawString(signedTx.Raw)
	hashs, err := doSendRaw(c.url, body, c.c)
	if err != nil {
		return "", err
	}
	return hashs[0], nil
}

func (c *HttpClient) SendSignedTx(tx *types.Transaction) (string, error) {
	data, err := rlp.EncodeToBytes(tx)
	if err != nil {
		return "", err
	}
	raw := hexutil.Encode(data)
	body := buildSendRawString(raw)
	hashs, err := doSendRaw(c.url, body, c.c)
	if err != nil {
		return "", err
	}
	return hashs[0], nil
}
