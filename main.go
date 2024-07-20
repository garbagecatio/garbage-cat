package main

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	watcher "github.com/garbagecatio/rug-ninja-sniper/internal/algod"
	"github.com/garbagecatio/rug-ninja-sniper/internal/config"
	"github.com/garbagecatio/rug-ninja-sniper/store"

	"github.com/algorand/go-algorand-sdk/v2/abi"
	"github.com/algorand/go-algorand-sdk/v2/client/v2/algod"
	"github.com/algorand/go-algorand-sdk/v2/crypto"
	"github.com/algorand/go-algorand-sdk/v2/mnemonic"
	"github.com/algorand/go-algorand-sdk/v2/transaction"
	"github.com/algorand/go-algorand-sdk/v2/types"
	"github.com/fatih/color"
	"github.com/garbagecatio/rug-ninja-sniper/misc"
	"github.com/mdp/qrterminal"
)

var AlgorandNodeURL = "https://mainnet-api.algonode.cloud"
var TimeFormat = "2006-01-02 15:04:05"
var PrintAllTxns = false
var pk = ""
var RugNinjaMainNetAppID uint64 = 2020762574
var appAddress = "7TL5PKBGPH4W7LEZW5SW5BGC4TH32XVFV5NVTXE4HTTPVK2JUJODCVTHSU"

var purchaseAmount uint64 = 10_000_000

var Algod *algod.Client
var err error
var txParams types.SuggestedParams

var RugNinjaTokenMint = "XrzHXA=="
var RugNinjaBuy = "ul43GA=="
var RugNinjaSell = "ymo5EA=="

var contract *abi.Contract
var account crypto.Account
var signer transaction.BasicAccountTransactionSigner
var buyCoinMethod abi.Method

func main() {

	flag.Uint64Var(&purchaseAmount, "amt", 1_000_000, "set the amount of each new token to purchase in algo")

	flag.Parse()

	fmt.Println("")
	fmt.Printf("Purchasing: %v Algo worth of every new token", color.YellowString("%v", purchaseAmount/100_000))
	fmt.Println("")

	setup()

	var ctx context.Context
	ctx, cancelFunc := context.WithCancel(context.Background())

	go func() {
		watchingConfig := config.StreamerConfig{
			Algod: &watcher.AlgoConfig{
				FRound: -1,
				LRound: -1,
				Queue:  1,
				ANodes: []*watcher.AlgoNodeConfig{
					{
						Address: AlgorandNodeURL,
						Id:      "nodely-node",
					},
				},
			},
		}

		blocks, status, err := watcher.AlgoStreamer(ctx, watchingConfig.Algod)
		if err != nil {
			log.Fatalf("[!ERR][_MAIN] error getting algod stream: %s\n", err)
		}

		go func() {
			for {
				select {
				case <-status:
					//noop
				case b := <-blocks:
					ProcessBlock(b)
				case <-ctx.Done():
					fmt.Println("DONE APPARENTLY")
				}
			}
		}()

		<-ctx.Done()
		fmt.Println("BLOCK WATCHER GOROUTINE FINISHED")
	}()

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGTERM, syscall.SIGINT)

loop:
	for range sigc {
		cancelFunc()
		fmt.Print("Shutting Down\n", time.Now().Format(TimeFormat))
		break loop
	}
}

func ProcessBlock(b *watcher.BlockWrap) {
	fmt.Printf("[BLK]: %v\n", b.Block.Round)

	txParams, err = Algod.SuggestedParams().Do(context.Background())
	if err != nil {
		fmt.Println(err)
		return
	}

	for i := range b.Block.Payset {
		stxn := b.Block.Payset[i]
		txn := b.Block.Payset[i].SignedTxnWithAD.SignedTxn.Txn

		id, err := watcher.DecodeTxnId(b.Block.BlockHeader, &stxn)
		if err != nil {
			fmt.Println(err)
			continue
		}

		if PrintAllTxns {
			fmt.Printf("[TXN]%s[%s]: %s\n", strings.Repeat(" ", 6-len(string(txn.Type))), strings.ToUpper(string(txn.Type)), id)
			innerTxns := misc.ListInner(&stxn.SignedTxnWithAD)
			if len(innerTxns) > 0 {
				for i := range innerTxns {
					stxn := innerTxns[i]
					fmt.Printf("     %s[%s]: [%v]\n", strings.Repeat(" ", 6-len(string(stxn.Txn.Type))), strings.ToUpper(string(stxn.Txn.Type)), i)
				}
			}
		}

		txnsToProcess := append([]types.SignedTxnWithAD{stxn.SignedTxnWithAD}, misc.ListInner(&stxn.SignedTxnWithAD)...)

		for i := range txnsToProcess {
			stxn := txnsToProcess[i]
			txn := txnsToProcess[i].Txn
			txAppID := uint64(txn.ApplicationFields.ApplicationID)

			isRugNinjaAppCall := txn.Type == types.ApplicationCallTx && txAppID == RugNinjaMainNetAppID
			hasArg := len(txn.ApplicationFields.ApplicationArgs) > 0
			if isRugNinjaAppCall && hasArg {
				encodedArg := base64.StdEncoding.EncodeToString(txn.ApplicationArgs[0])
				switch encodedArg {
				case RugNinjaTokenMint:
					// get the assetID
					assetID := stxn.EvalDelta.GlobalDelta["LAST_COIN"].Uint
					name := stxn.EvalDelta.InnerTxns[0].Txn.AssetConfigTxnFields.AssetParams.AssetName
					// buy the token
					fmt.Printf("[MINT]      [%s]: %v\n", name, assetID)

					err = buyToken(name, assetID, purchaseAmount)
					if err != nil {
						fmt.Println(err)
						continue
					}

					fmt.Printf("[PURCHASED]      [%s]: %v\n", name, (purchaseAmount / 100_000))
				case RugNinjaBuy:
				case RugNinjaSell:
				}
			}
		}
	}
}

func setup() {

	if !store.StoreExists() {
		mn := newAccount()
		err = store.StoreMnemonicToFile(mn)
		if err != nil {
			log.Fatalf("failed to store mnemonic to file: %s", err)
		}

		pk = mn

		key, err := mnemonic.ToPrivateKey(pk)
		if err != nil {
			log.Fatalln(err, "failed to convert mnemonic to private key")
		}

		account, err = crypto.AccountFromPrivateKey(key)
		if err != nil {
			log.Fatalln(err, "failed to convert private key to account")
		}

		fmt.Println("")
		fmt.Println("New Account Generated: ", color.GreenString(account.Address.String()))
		fmt.Println("")
		color.Red("Mnemonics are stored in plain text, do not put this on a public server.")
		color.Red("Do not keep large amounts of Algo in it.")
		color.Red("Please backup the mnemonic in the secret.txt file.")
		color.Red("If you lose the mnemonic, you lose the account.")
		fmt.Println("")
		fmt.Println("Fund this account & it will automatically purchase new tokens.")
		fmt.Println("")
		fmt.Println("Scan the QR code below or copy the address and fund the account.")
		fmt.Println("")

		PrintQR(account.Address.String())

		fmt.Println("")
		greenEnter := color.CyanString("enter")

		fmt.Printf("Press %v to continue\n", greenEnter)

		_, err = fmt.Scanln()
		if err != nil {
			log.Fatalf("failed to read input: %s", err)
		}
	} else {
		pk, err = store.ReadMnemonicFromFile()
		if err != nil {
			log.Fatalf("failed to read mnemonic from file: %s", err)
		}

		key, err := mnemonic.ToPrivateKey(pk)
		if err != nil {
			log.Fatalln(err, "failed to convert mnemonic to private key")
		}

		account, err = crypto.AccountFromPrivateKey(key)
		if err != nil {
			log.Fatalln(err, "failed to convert private key to account")
		}

		fmt.Println("")
		fmt.Println("Account: ", color.GreenString(account.Address.String()))

		fmt.Println("")
		fmt.Println("Fund this account & it will automatically purchase new tokens.")
		fmt.Println("")
		fmt.Println("Scan the QR code below or copy the address and fund the account.")
		fmt.Println("")

		PrintQR(account.Address.String())
		fmt.Println("")
	}

	signer = transaction.BasicAccountTransactionSigner{Account: account}

	b, err := os.ReadFile("./RugNinja.arc4.json")
	if err != nil {
		log.Fatalf("failed to read contract file: %s", err)
	}

	contract = &abi.Contract{}
	if err := json.Unmarshal(b, contract); err != nil {
		log.Fatalf("failed to unmarshal contract: %s", err)
	}

	buyCoinMethod, err = contract.GetMethodByName("buyCoin")
	if err != nil {
		log.Fatalln(err)
	}

	Algod, err = algod.MakeClient(AlgorandNodeURL, "")
	if err != nil {
		log.Fatalln(err)
	}
}

func newAccount() string {
	account := crypto.GenerateAccount()
	mn, err := mnemonic.FromPrivateKey(account.PrivateKey)
	if err != nil {
		log.Fatalln(err, "failed to convert private key to mnemonic")
	}
	return mn
}

func createBoxName(address types.Address, value uint64) []byte {
	// Create a slice to hold the result (32 bytes for address + 8 bytes for uint64)
	result := make([]byte, 40)

	// Copy the 32-byte address into the first 32 bytes of the result
	copy(result[:32], address[:])

	// Convert the uint64 to 8 bytes and append it to the result
	binary.BigEndian.PutUint64(result[32:], value)

	return result
}

func buyToken(assetName string, assetID uint64, amount uint64) error {
	txParams, err = Algod.SuggestedParams().Do(context.Background())
	if err != nil {
		return err
	}

	atc := transaction.AtomicTransactionComposer{}

	pmt, err := transaction.MakePaymentTxn(
		account.Address.String(),
		appAddress,
		amount,
		nil,
		types.ZeroAddress.String(),
		txParams,
	)
	if err != nil {
		return err
	}

	atc.AddTransaction(transaction.TransactionWithSigner{Txn: pmt, Signer: signer})

	mcp := transaction.AddMethodCallParams{
		AppID:           RugNinjaMainNetAppID,
		Sender:          account.Address,
		SuggestedParams: txParams,
		OnComplete:      types.NoOpOC,
		Signer:          signer,
		Method:          buyCoinMethod,
		MethodArgs:      []interface{}{assetID, 0},
		ForeignAssets:   []uint64{assetID},
		BoxReferences: []types.AppBoxReference{
			{
				AppID: RugNinjaMainNetAppID,
				Name:  []byte(assetName),
			},
			{
				AppID: RugNinjaMainNetAppID,
				Name:  createBoxName(account.Address, assetID),
			},
		},
	}

	err = atc.AddMethodCall(mcp)
	if err != nil {
		return err
	}

	// result, err := atc.Simulate(context.Background(), Algod, models.SimulateRequest{})
	_, err = atc.Execute(Algod, context.Background(), 4)
	if err != nil {
		return err
	}

	return nil
}

func PrintQR(address string) {
	config := qrterminal.Config{
		Level:     qrterminal.M,
		Writer:    os.Stdout,
		BlackChar: qrterminal.BLACK,
		WhiteChar: qrterminal.WHITE,
		QuietZone: 3,
	}

	qrterminal.GenerateWithConfig(fmt.Sprintf("algorand://%v?amount=0", account.Address.String()), config)
}
