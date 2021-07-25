package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"
)

const DATA_FORMAT = "%s,%s,%s,%s,%s,%s,%s,%s,%d,\n"
const CCPTRADEID_FORMAT = "CCP%d"
const BOOK_FORMAT= "%dBK%s"
const TRADEID_FORMAT= "%s%d"
const CAPITAL_LETTERS = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
const DATE_FORMAT = "2006/01/02"
const MAX_NOTIONAL = 999000

type TradeDetails struct {
	NextTradeIndex int
	Books []string
}

var baseDate time.Time

func init() {
	baseDate, _ = time.Parse(DATE_FORMAT, "2022/12/31")
}

func main() {
	totalParties := flag.Int("totalParties", 20, "total number of counterparties")
	totalTrades := flag.Int("totalTrades", 100000, "total number of trades")
	totalMaturityDates := flag.Int("totalMaturityDates", 3, "total number of different maturity dates")
	maxNoOfBook := flag.Int("maxNoOfBook", 1, "maximum number of books for one party")
	currenciesStr := flag.String("currencies", "AUD", "list of currencies to be included, separated by whitespace")
	flag.Parse()

	currencies := strings.Split(*currenciesStr, " ")
	log.Printf("Get %d currencies: %s\n", len(currencies), strings.Join(currencies, ", "))

	maturityDates := make([]string, *totalMaturityDates)
	for i:=0; i<len(maturityDates); i++ {
		maturityDates[i] = baseDate.AddDate(i, 0, 0).Format(DATE_FORMAT)
	}
	log.Printf("Populate %d maturity dates: %s\n", len(maturityDates), strings.Join(maturityDates, ", "))

	parties := make([]string, *totalParties)
	for i:=0; i<len(parties); i++ {
		parties[i] = string(CAPITAL_LETTERS[i])
	}
	log.Printf("Have %d parties: %s\n", len(parties), strings.Join(parties, ", "))

	partyToTradeDetails := make(map[string]*TradeDetails)
	for _, party := range parties {
		books := make([]string, rand.Intn(*maxNoOfBook)+1)
		for i:=0; i<len(books); i++ {
			books[i] = fmt.Sprintf(BOOK_FORMAT, i+1, party)
		}
		partyToTradeDetails[party] = &TradeDetails{
			NextTradeIndex: 0,
			Books:          books,
		}
	}

	trades := make([]string, *totalTrades*2)
	ccpTradeIndex := 0

	var party, cpty, book, tradeID, ccpTradeID, maturityDate, currency string
	var notional uint64
	for i:=0; i<*totalTrades; i++ {
		party = parties[rand.Intn(*totalParties)]
		cpty = parties[rand.Intn(*totalParties)]

		for party == cpty {
			party = parties[rand.Intn(*totalParties)]
			cpty = parties[rand.Intn(*totalParties)]
		}

		book = partyToTradeDetails[party].Books[rand.Intn(len(partyToTradeDetails[party].Books))]

		tradeID = fmt.Sprintf(TRADEID_FORMAT, party, partyToTradeDetails[party].NextTradeIndex)
		partyToTradeDetails[party].NextTradeIndex++

		currency = currencies[rand.Intn(len(currencies))]

		maturityDate = maturityDates[rand.Intn(*totalMaturityDates)]

		ccpTradeID = fmt.Sprintf(CCPTRADEID_FORMAT, ccpTradeIndex)
		ccpTradeIndex++

		notional = uint64(rand.Intn(MAX_NOTIONAL) + 1000)

		trades[i*2] = fmt.Sprintf(DATA_FORMAT, party, book, tradeID, "P", currency, maturityDate, cpty, ccpTradeID, notional)
		log.Printf("Added trade %d\n", i*2)

		book = partyToTradeDetails[cpty].Books[rand.Intn(len(partyToTradeDetails[cpty].Books))]

		tradeID = fmt.Sprintf(TRADEID_FORMAT, cpty, partyToTradeDetails[cpty].NextTradeIndex)
		partyToTradeDetails[cpty].NextTradeIndex++

		trades[i*2+1] = fmt.Sprintf(DATA_FORMAT, cpty, book, tradeID, "R", currency, maturityDate, party, ccpTradeID, notional)
		log.Printf("Added trade %d\n", i*2+1)
	}

	output := "Party,Book,TradeID,PAY/RECEIVE,Currency,MaturityDate,Cpty,CCPTradeID,Notional,\n" + strings.Join(trades, "")

	f, err := os.Create("mock_input.csv")
	if err != nil {
		log.Fatalf(err.Error())
	}

	f.WriteString(output)
	f.Close()
}
