package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/dvcrn/pocketsmith-go"
)

type Config struct {
	PocketsmithToken              string
	PocketsmithTransactionAccount int
	TransactionMetaFile           string
}

func getConfig() *Config {
	config := &Config{}

	transactionAccountEnv := os.Getenv("POCKETSMITH_TRANSACTION_ACCOUNT")
	transactionAccountParsed := 0
	// try parse int
	if transactionAccountEnv != "" {
		transactionAccountParsed, _ = strconv.Atoi(transactionAccountEnv)
	}

	// Define command-line flags
	flag.StringVar(&config.PocketsmithToken, "pocketsmith-token", os.Getenv("POCKETSMITH_TOKEN"), "Pocketsmith API token")
	flag.IntVar(&config.PocketsmithTransactionAccount, "pocketsmith-transaction-account", transactionAccountParsed, "Bangkok Bank Pocketsmith Transaction account")
	flag.StringVar(&config.TransactionMetaFile, "transaction-meta-file", os.Getenv("POCKETSMITH_META_FILE"), "Path with meta information for transactions")

	flag.Parse()

	if config.PocketsmithToken == "" {
		fmt.Println("Error: Pocketsmith token is required. Set via -token flag or POCKETSMITH_TOKEN environment variable")
		os.Exit(1)
	}

	if config.PocketsmithTransactionAccount == 0 {
		fmt.Println("Error: Pocketsmith transaction account is required. Set via -pocketsmith-transaction-account flag or POCKETSMITH_TRANSACTION_ACCOUNT environment variable")
		os.Exit(1)
	}

	if config.TransactionMetaFile == "" {
		fmt.Println("Error: Transaction meta path is required. Set via -transaction-meta-file flag or TRANSACTION_META_FILE environment variable")
		os.Exit(1)
	}

	return config
}

func findField(fields []string, key string) string {
	for _, part := range fields {
		needle := fmt.Sprintf("%s=", key)
		if strings.HasPrefix(part, needle) {
			return strings.TrimPrefix(part, needle)
		}
	}

	return ""
}

func main() {
	config := getConfig()
	targetAccountID := config.PocketsmithTransactionAccount // bangkok bank transaction account
	ps := pocketsmith.NewClient(config.PocketsmithToken)

	var fileContent string
	if strings.HasPrefix(config.TransactionMetaFile, "http") {
		// download file
		response, err := http.Get(config.TransactionMetaFile)
		if err != nil {
			fmt.Println("Error downloading file:", err)
			return
		}
		defer response.Body.Close()

		content, err := io.ReadAll(response.Body)
		if err != nil {
			fmt.Println("Error reading file:", err)
			return
		}

		fileContent = string(content)
	} else {
		// read file
		file, err := os.Open(config.TransactionMetaFile)
		if err != nil {
			fmt.Println("Error opening file:", err)
			return
		}
		defer file.Close()

		fc, err := io.ReadAll(file)
		if err != nil {
			fmt.Println("Error reading file:", err)
			return
		}

		fileContent = string(fc)
	}

	// Parse the test content
	updatedTransactions := 0
	processedTxRefs := map[int64]struct{}{}

	lines := strings.Split(string(fileContent), "\n")
	// reverse array so newest are first
	slices.Reverse(lines)

	for i, content := range lines {
		if content == "" {
			continue
		}

		// Split the content into fields
		fields := strings.Split(content, ";")

		// Extract the fields
		filename := findField(fields, "filename")
		to := findField(fields, "to")
		from := findField(fields, "from")
		amount := strings.Replace(findField(fields, "amountTHB"), " THB", "", -1)
		// amountOther := findField(fields, "amountOther")
		// currencyOther := findField(fields, "currencyOther")
		dateField := findField(fields, "date")
		timeField := findField(fields, "time")
		// combine date and time into time.Time
		combined := fmt.Sprintf("%s %s", dateField, timeField)
		loc, _ := time.LoadLocation("Asia/Bangkok")
		date, err := time.ParseInLocation("2006-01-02 15:04", combined, loc)
		if err != nil {
			panic(err)
		}

		bankref := findField(fields, "bankref")
		txref := findField(fields, "txref")

		fmt.Printf("[%d/%d] Processing: %s from %s\n", i+1, len(lines), txref, to)

		memo := fmt.Sprintf("%s %s %s %s %s %s %s %s", filename, to, from, amount, date, date, bankref, txref)

		searchRes, err := ps.SearchTransactions(targetAccountID, dateField, dateField, amount)
		if err != nil {
			fmt.Println("Could not find transaction: ", err)
			continue
		}

		var tx *pocketsmith.Transaction
		if len(searchRes) > 1 {
			fmt.Printf("Multiple transactions found for %s: %d\n", memo, len(searchRes))
			// Multilple transactions with same amount and date, so can just take any
			for i, s := range searchRes {
				if strings.Contains(s.Memo, txref) {
					continue
				}

				// means this was already used, so we can skip it
				if _, ok := processedTxRefs[s.ID]; ok {
					continue
				}

				fmt.Printf("Using %d transaction: %s\n", i, s.Memo)
				tx = s

				processedTxRefs[tx.ID] = struct{}{}
				break
			}
		} else if len(searchRes) == 0 {
			fmt.Printf("No transactions found for %s\n", memo)
			continue
		} else {
			tx = searchRes[0]
		}

		if strings.Contains(tx.Memo, txref) {
			fmt.Printf("Transaction already enriched: %s\n", tx.Memo)
			continue
		}

		fmt.Printf("Enriching transaction: %d: %s -> %s\n", tx.ID, tx.Payee, to)
		txUpdate := &pocketsmith.CreateTransaction{
			Payee:       to,
			Memo:        content,
			Amount:      tx.Amount,
			Date:        tx.Date,
			IsTransfer:  tx.IsTransfer,
			NeedsReview: tx.NeedsReview,
			Note:        tx.Note,
		}

		err = ps.UpdateTransaction(tx.ID, txUpdate)
		if err != nil {
			fmt.Println("Could not update transaction: ", err)
			continue
		}

		updatedTransactions++
		fmt.Println("")
	}

	fmt.Printf("Done. Processed %d transactions, %d new", len(lines), updatedTransactions)
}
