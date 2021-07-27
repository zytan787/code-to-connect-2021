package internal

import (
	"encoding/base64"
	"fmt"
	"github.com/araddon/dateparse"
	"github.com/gocarina/gocsv"
	"github.com/zytan787/code-to-connect-2021/api"
	"sort"
	"strconv"
	"strings"
	"time"
)

type PortfolioLoader struct {
	CcpTradeIDToCompressibleTrades map[string][]*Trade
	ExcludedTrades                 []*ExcludedTrade
}

func (handler *MainHandler) DecodeInputFiles(inputFiles []api.File) ([]*RawTrade, error) {
	result := make([]*RawTrade, 0)

	var fileBytes []byte
	var err error
	for _, inputFile := range inputFiles {
		fileBytes, err = base64.StdEncoding.DecodeString(inputFile.FileContent)
		if err != nil {
			return nil, fmt.Errorf("unable to decode the base64 string of file %s due to: %s", inputFile.FileName, err.Error())
		}
		trades := make([]*RawTrade, 0)
		err = gocsv.UnmarshalBytes(fileBytes, &trades)
		if err != nil {
			return nil, fmt.Errorf("unable to unmarshal bytes into trades for file %s due to: %s, "+
				"make sure your CSV file has the correct format", inputFile.FileName, err.Error())
		}
		result = append(result, trades...)
	}

	return result, nil
}

func (handler *MainHandler) LoadPortfolio(rawTrades []*RawTrade) {
	ccpTradeIDToCompressibleTrades, excludedTrades := categorizeRawTrades(rawTrades)

	handler.PortfolioLoader.CcpTradeIDToCompressibleTrades = ccpTradeIDToCompressibleTrades
	handler.PortfolioLoader.ExcludedTrades = excludedTrades
}

func categorizeRawTrades(rawTrades []*RawTrade) (ccpTradeIDToCompressibleTrades map[string][]*Trade, excludedTrades []*ExcludedTrade) {
	tmpCleanTrades := make(map[string][]*Trade)

	var cleanTrade *Trade
	var err error
	for _, rawTrade := range rawTrades {
		cleanTrade, err = cleanRawTrade(rawTrade)
		if err == nil {
			tmpCleanTrades[cleanTrade.CCPTradeID] = append(tmpCleanTrades[cleanTrade.CCPTradeID], cleanTrade)
		} else {
			excludedTrades = append(excludedTrades, createExcludedTradeFromRawTrade(rawTrade, err))
		}
	}

	ccpTradeIDToCompressibleTrades = make(map[string][]*Trade)
	var compressible bool
	for CCPTradeID, seenCleanTrades := range tmpCleanTrades {
		compressible = false

		if len(seenCleanTrades) == 2 {
			err = verifyPairedTrades(seenCleanTrades[0], seenCleanTrades[1])
			if err == nil {
				compressible = true
			}
		}

		if compressible {
			ccpTradeIDToCompressibleTrades[CCPTradeID] = seenCleanTrades
		} else {
			if len(seenCleanTrades) == 1 {
				excludedTrades = append(excludedTrades, createExcludedTradeFromCleanTrades(seenCleanTrades, "trade not submitted on both sides")...)
			} else if len(seenCleanTrades) == 2 {
				excludedTrades = append(excludedTrades, createExcludedTradeFromCleanTrades(seenCleanTrades, err.Error())...)
			} else {
				excludedTrades = append(excludedTrades, createExcludedTradeFromCleanTrades(seenCleanTrades, fmt.Sprintf("more than 2 trades have the same CCPTradeID: %s", CCPTradeID))...)
			}
		}
	}
	return
}

func verifyPairedTrades(trade1 *Trade, trade2 *Trade) error {
	if trade1.PayOrReceive == trade2.PayOrReceive {
		return fmt.Errorf("both trades with CCPTradeID=%s have PayOrReceive=%s",
			trade1.CCPTradeID,
			trade1.PayOrReceive)
	}

	if trade1.Notional != trade2.Notional {
		return fmt.Errorf("trades with CCPTradeID=%s have different notionals: %d and %d",
			trade1.CCPTradeID,
			trade1.Notional,
			trade2.Notional)
	}

	if trade1.Currency != trade2.Currency {
		return fmt.Errorf("trades with CCPTradeID=%s have different currency: %s and %s",
			trade1.CCPTradeID,
			trade1.Currency,
			trade2.Currency)
	}

	if !trade1.MaturityDate.Equal(trade2.MaturityDate) {
		return fmt.Errorf("trades with CCPTradeID=%s have different MaturityDate: %s and %s",
			trade1.CCPTradeID,
			trade1.MaturityDate.Format(DATE_FORMAT),
			trade2.MaturityDate.Format(DATE_FORMAT))
	}

	if trade1.Cpty != trade2.Party || trade2.Cpty != trade1.Party {
		return fmt.Errorf("trades with CCPTradeID=%s counterparties do not match, trade1: Party=%s, Cpty=%s, trade2: Party=%s, Cpty=%s",
			trade1.CCPTradeID,
			trade1.Party, trade1.Cpty,
			trade2.Party, trade2.Cpty)
	}

	return nil
}

func cleanRawTrade(rawTrade *RawTrade) (*Trade, error) {
	errors := make([]string, 0)
	emptyColumns := make([]string, 0)

	party := strings.TrimSpace(rawTrade.Party)
	if len(party) == 0 {
		emptyColumns = append(emptyColumns, "Party")
	}

	book := strings.TrimSpace(rawTrade.Book)
	if len(book) == 0 {
		emptyColumns = append(emptyColumns, "Book")
	}

	tradeID := strings.TrimSpace(rawTrade.TradeID)
	if len(tradeID) == 0 {
		emptyColumns = append(emptyColumns, "TradeID")
	}

	currency := strings.TrimSpace(rawTrade.Currency)
	if len(currency) == 0 {
		emptyColumns = append(emptyColumns, "Currency")
	}

	cPty := strings.TrimSpace(rawTrade.Cpty)
	if len(cPty) == 0 {
		emptyColumns = append(emptyColumns, "Cpty")
	}

	ccpTradeID := strings.TrimSpace(rawTrade.CCPTradeID)
	if len(ccpTradeID) == 0 {
		emptyColumns = append(emptyColumns, "CCPTradeID")
	}

	if len(emptyColumns) == 1 {
		errors = append(errors, fmt.Sprintf("%s is empty", emptyColumns[0]))
	} else if len(emptyColumns) > 1 {
		errors = append(errors, fmt.Sprintf("%s are empty", strings.Join(emptyColumns, ", ")))
	}

	notional, err := strconv.Atoi(rawTrade.Notional)
	if err != nil {
		errors = append(errors, fmt.Sprintf("Notional %s is not a valid integer", rawTrade.Notional))
	} else if notional < 0 {
		errors = append(errors, fmt.Sprintf("Notional %s is a negative value", rawTrade.Notional))
	}

	if rawTrade.PayOrReceive != "P" && rawTrade.PayOrReceive != "R" {
		errors = append(errors, fmt.Sprintf("PayOrReceive %s is neither 'P' or 'R'", rawTrade.PayOrReceive))
	}

	maturityDate, err := dateparse.ParseAny(rawTrade.MaturityDate)
	if err != nil {
		errors = append(errors, fmt.Sprintf("fail to parse MaturityDate %s, date is invalid", rawTrade.MaturityDate))
	}

	if len(errors) > 0 {
		return nil, fmt.Errorf("%s", strings.Join(errors, "\n"))
	}

	return &Trade{
		Party:        party,
		Book:         book,
		TradeID:      tradeID,
		PayOrReceive: rawTrade.PayOrReceive,
		Currency:     rawTrade.Currency,
		MaturityDate: maturityDate,
		Cpty:         cPty,
		CCPTradeID:   ccpTradeID,
		Notional:     uint64(notional),
	}, nil
}

func createExcludedTradeFromRawTrade(rawTrade *RawTrade, err error) *ExcludedTrade {
	return &ExcludedTrade{
		Party:        rawTrade.Party,
		Book:         rawTrade.Book,
		TradeID:      rawTrade.TradeID,
		PayOrReceive: rawTrade.PayOrReceive,
		Currency:     rawTrade.Currency,
		MaturityDate: rawTrade.MaturityDate,
		Cpty:         rawTrade.Cpty,
		CCPTradeID:   rawTrade.CCPTradeID,
		Notional:     rawTrade.Notional,
		Error:        err.Error(),
	}
}

func createExcludedTradeFromCleanTrades(cleanTrades []*Trade, errorMessage string) []*ExcludedTrade {
	excludedTrades := make([]*ExcludedTrade, len(cleanTrades))

	for i := 0; i < len(cleanTrades); i++ {
		excludedTrades[i] = &ExcludedTrade{
			Party:        cleanTrades[i].Party,
			Book:         cleanTrades[i].Book,
			TradeID:      cleanTrades[i].TradeID,
			PayOrReceive: cleanTrades[i].PayOrReceive,
			Currency:     cleanTrades[i].Currency,
			MaturityDate: cleanTrades[i].MaturityDate.Format(DATE_FORMAT),
			Cpty:         cleanTrades[i].Cpty,
			CCPTradeID:   cleanTrades[i].CCPTradeID,
			Notional:     fmt.Sprintf("%d", cleanTrades[i].Notional),
			Error:        errorMessage,
		}
	}
	return excludedTrades
}

func (handler *MainHandler) GetExcludedTradesAsCSV() (string, error) {
	excludedTrades := handler.PortfolioLoader.ExcludedTrades

	sort.Slice(excludedTrades, func(i, j int) bool {
		if excludedTrades[i].Party != excludedTrades[j].Party {
			return excludedTrades[i].Party < excludedTrades[j].Party
		}
		if excludedTrades[i].Book != excludedTrades[j].Book {
			return excludedTrades[i].Book < excludedTrades[j].Book
		}
		if excludedTrades[i].Currency != excludedTrades[j].Currency {
			return excludedTrades[i].Currency < excludedTrades[j].Currency
		}
		if excludedTrades[i].MaturityDate != excludedTrades[j].MaturityDate {
			timeI, _ := time.Parse(DATE_FORMAT, excludedTrades[i].MaturityDate)
			timeJ, _ := time.Parse(DATE_FORMAT, excludedTrades[j].MaturityDate)
			return timeI.Before(timeJ)
		}
		if excludedTrades[i].PayOrReceive != excludedTrades[j].PayOrReceive {
			return excludedTrades[i].PayOrReceive < excludedTrades[j].PayOrReceive
		}
		return excludedTrades[i].TradeID < excludedTrades[j].TradeID
	})

	exclusion, err := gocsv.MarshalBytes(excludedTrades)
	if err != nil {
		return "", err
	}

	result := base64.StdEncoding.EncodeToString(exclusion)
	return result, nil
}
