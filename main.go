package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
)

func (cli *CLI) createBC(a string) {
	if !CheckAddress(a) {
		log.Panic("ERROR: Address is not valid")
	}
	bc := CreateBlockchain(a)
	bc.db.Close()
	fmt.Println("Done!")
}

func (cli *CLI) createWallet() {
	wallets, _ := NewWallets()
	address := wallets.CreateWallet()
	wallets.SaveToFile()

	fmt.Printf("Your new address: %s\n", address)
}

func (cli *CLI) getBalance(a string) {
	if !CheckAddress(a) {
		log.Panic("ERROR: Address is not valid")
	}
	bc := NewBlockchain(a)
	defer bc.db.Close()

	balance := 0
	pubKeyHash := Base58Decode([]byte(a))
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-4]
	UTXOs := bc.FindUTXO(pubKeyHash)

	for _, out := range UTXOs {
		balance += out.V
	}

	fmt.Printf("Balance of '%s': %d\n", a, balance)
}


func (cli *CLI) listAddresses() {
	wallets, err := NewWallets()
	if err != nil {
		log.Panic(err)
	}
	addresses := wallets.GetAddresses()

	for _, address := range addresses {
		fmt.Println(address)
	}
}

func (cli *CLI) printChain() {
	bc := NewBlockchain("")
	defer bc.db.Close()

	bci := bc.Iterator()

	for {
		block := bci.Next()

		fmt.Printf("============ Block %x ============\n", block.Hash)
		fmt.Printf("Previous block: %x\n", block.PreHash)
		pow := NewPOW(block)
		fmt.Printf("PoW: %s\n\n", strconv.FormatBool(pow.Validate()))
		for _, tx := range block.Transactions {
			fmt.Println(tx)
		}
		fmt.Printf("\n\n")

		if len(block.PreHash) == 0 {
			break
		}
	}
}

func (cli *CLI) send(f, t string, m int) {
	if !CheckAddress(f) {
		log.Panic("ERROR: Sender address is not valid")
	}
	if !CheckAddress(t) {
		log.Panic("ERROR: Recipient address is not valid")
	}

	bc := NewBlockchain(f)
	defer bc.db.Close()

	tx := NewTransaction(f, t, m, bc)
	bc.Mine([]*Transaction{tx})
	fmt.Println("Success!")
}

// CLI responsible for processing command line arguments
type CLI struct{}

func (cli *CLI) printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  createblockchain/cb -a ADDRESS - Create a rootchain and send block reward to ADDRESS")
	fmt.Println("  createwallet/cw - Generates a new key-pair and saves it into the wallet file")
	fmt.Println("  getbalance/g -a ADDRESS - Get balance of ADDRESS")
	fmt.Println("  listaddresses/l - Lists all addresses from the wallet file")
	fmt.Println("  printchain/p - Print all the blocks of the blockchain")
	fmt.Println("  send/s -f FROM -t TO -m AMOUNT - Send AMOUNT of coins from FROM address to TO")
}

func (cli *CLI) checkArgs() {
	if len(os.Args) < 2 {
		cli.printUsage()
		os.Exit(1)
	}
}

// Run parses command line arguments and processes commands
func (cli *CLI) Run() {
	cli.checkArgs()

	createBCCli := flag.NewFlagSet("createblockchain", flag.ExitOnError)
	getBalanceCli := flag.NewFlagSet("getbalance", flag.ExitOnError)
	createWalletCli := flag.NewFlagSet("createwallet", flag.ExitOnError)
	listAddressesCli := flag.NewFlagSet("listaddresses", flag.ExitOnError)
	sendCli := flag.NewFlagSet("send", flag.ExitOnError)
	printChainCli := flag.NewFlagSet("printchain", flag.ExitOnError)

	getBalanceAddress := getBalanceCli.String("a", "", "The address to get balance for")
	createBCAddress := createBCCli.String("a", "", "The address to send genesis block reward to")
	from := sendCli.String("f", "", "Source wallet address")
	to := sendCli.String("t", "", "Destination wallet address")
	amount := sendCli.Int("m", 0, "Amount to send")

	switch os.Args[1] {
	case "getbalance":
	case "g":
		err := getBalanceCli.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "createblockchain":
	case "cb":
		err := createBCCli.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "createwallet":
	case "cw":
		err := createWalletCli.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "listaddresses":
	case "l":
		err := listAddressesCli.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "printchain":
	case "p":
		err := printChainCli.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "send":
	case "s":
		err := sendCli.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	default:
		cli.printUsage()
		os.Exit(1)
	}

	if getBalanceCli.Parsed() {
		if *getBalanceAddress == "" {
			getBalanceCli.Usage()
			os.Exit(1)
		}
		cli.getBalance(*getBalanceAddress)
	}

	if createBCCli.Parsed() {
		if *createBCAddress == "" {
			createBCCli.Usage()
			os.Exit(1)
		}
		cli.createBC(*createBCAddress)
	}

	if createWalletCli.Parsed() {
		cli.createWallet()
	}

	if listAddressesCli.Parsed() {
		cli.listAddresses()
	}

	if printChainCli.Parsed() {
		cli.printChain()
	}

	if sendCli.Parsed() {
		if *from == "" || *to == "" || *amount <= 0 {
			sendCli.Usage()
			os.Exit(1)
		}

		cli.send(*from, *to, *amount)
	}
}

func main() {
	cli := CLI{}
	cli.Run()
}
