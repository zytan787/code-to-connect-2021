package internal

import (
	"encoding/base64"
	"fmt"
	"github.com/gocarina/gocsv"
	"sort"
	"time"
)

type CompressionEngine struct {
	CompressionResults         []*CompressionResult
	BookLevelCompressionResults []*CompressionResultBookLevel
}

func (handler *MainHandler) GenerateCompressionResults() error {
	keyToPayOrReceiveToTrades := handler.getKeyToPayOrReceiveToTrades(false)

	// TODO don't repeat yourself
	compressionResults := make([]*CompressionResult, 0)
	var originalPayNotional, originalReceiveNotional, newPayNotional, newReceiveNotional uint64
	var trade *Trade
	for _, payOrReceiveToTrades := range keyToPayOrReceiveToTrades {
		originalPayNotional = sumNotional(payOrReceiveToTrades["P"])
		originalReceiveNotional = sumNotional(payOrReceiveToTrades["R"])

		if originalPayNotional > originalReceiveNotional {
			newPayNotional = originalPayNotional-originalReceiveNotional
			newReceiveNotional = 0
		} else {
			newReceiveNotional = originalReceiveNotional-originalPayNotional
			newPayNotional = 0
		}

		if len(payOrReceiveToTrades["P"]) > 0 {
			trade = payOrReceiveToTrades["P"][0]
		} else {
			trade = payOrReceiveToTrades["R"][0]
		}

		payCompressionResult := &CompressionResult{
			Party:            trade.Party,
			Currency:         trade.Currency,
			MaturityDate:     trade.MaturityDate.Format(DATE_FORMAT),
			PayOrReceive:     "P",
			CompressionType:  generateCompressionType(newPayNotional),
			OriginalNotional: fmt.Sprintf("%d", originalPayNotional),
			Notional:         fmt.Sprintf("%d", newPayNotional),
			CompressionRate:  generateCompressionRate(originalPayNotional, newPayNotional),
		}

		receiveCompressionResult := &CompressionResult{
			Party:            trade.Party,
			Currency:         trade.Currency,
			MaturityDate:     trade.MaturityDate.Format(DATE_FORMAT),
			PayOrReceive:     "R",
			CompressionType:  generateCompressionType(newReceiveNotional),
			OriginalNotional: fmt.Sprintf("%d", originalReceiveNotional),
			Notional:         fmt.Sprintf("%d", newReceiveNotional),
			CompressionRate:  generateCompressionRate(originalReceiveNotional, newReceiveNotional),
		}

		compressionResults = append(compressionResults, payCompressionResult)
		compressionResults = append(compressionResults, receiveCompressionResult)
	}

	handler.CompressionEngine.CompressionResults = compressionResults
	return nil
}

func (handler *MainHandler) GenerateBookLevelCompressionResults() error {
	bookLevelKeyToPayOrReceiveToTrades := handler.getKeyToPayOrReceiveToTrades(true)

	bookLevelCompressionResults := make([]*CompressionResultBookLevel, 0)
	var originalPayNotional, originalReceiveNotional, newPayNotional, newReceiveNotional uint64
	var trade *Trade
	for _, payOrReceiveToTrades := range bookLevelKeyToPayOrReceiveToTrades {
		originalPayNotional = sumNotional(payOrReceiveToTrades["P"])
		originalReceiveNotional = sumNotional(payOrReceiveToTrades["R"])

		if originalPayNotional > originalReceiveNotional {
			newPayNotional = originalPayNotional-originalReceiveNotional
			newReceiveNotional = 0
		} else {
			newReceiveNotional = originalReceiveNotional-originalPayNotional
			newPayNotional = 0
		}

		if len(payOrReceiveToTrades["P"]) > 0 {
			trade = payOrReceiveToTrades["P"][0]
		} else {
			trade = payOrReceiveToTrades["R"][0]
		}

		payCompressionResult := &CompressionResultBookLevel{
			Party:            trade.Party,
			Book:             trade.Book,
			Currency:         trade.Currency,
			MaturityDate:     trade.MaturityDate.Format(DATE_FORMAT),
			PayOrReceive:     "P",
			CompressionType:  generateCompressionType(newPayNotional),
			OriginalNotional: fmt.Sprintf("%d", originalPayNotional),
			Notional:         fmt.Sprintf("%d", newPayNotional),
			CompressionRate:  generateCompressionRate(originalPayNotional, newPayNotional),
		}

		receiveCompressionResult := &CompressionResultBookLevel{
			Party:            trade.Party,
			Book:             trade.Book,
			Currency:         trade.Currency,
			MaturityDate:     trade.MaturityDate.Format(DATE_FORMAT),
			PayOrReceive:     "R",
			CompressionType:  generateCompressionType(newReceiveNotional),
			OriginalNotional: fmt.Sprintf("%d", originalReceiveNotional),
			Notional:         fmt.Sprintf("%d", newReceiveNotional),
			CompressionRate:  generateCompressionRate(originalReceiveNotional, newReceiveNotional),
		}

		bookLevelCompressionResults = append(bookLevelCompressionResults, payCompressionResult)
		bookLevelCompressionResults = append(bookLevelCompressionResults, receiveCompressionResult)
	}

	handler.CompressionEngine.BookLevelCompressionResults = bookLevelCompressionResults
	return nil
}

func (handler *MainHandler) getKeyToPayOrReceiveToTrades(bookLevel bool) map[string]map[string][]*Trade {
	keyToPayOrReceiveToTrades := make(map[string]map[string][]*Trade)

	var key string
	var trades []*Trade
	for _, pairedTrades := range handler.PortfolioLoader.CcpTradeIDToCompressibleTrades {
		key = generateKeyFromTrade(pairedTrades[0], bookLevel)
		if keyToPayOrReceiveToTrades[key] == nil {
			keyToPayOrReceiveToTrades[key] = make(map[string][]*Trade)
		}
		trades = keyToPayOrReceiveToTrades[key][pairedTrades[0].PayOrReceive]
		keyToPayOrReceiveToTrades[key][pairedTrades[0].PayOrReceive] = append(trades, pairedTrades[0])

		key = generateKeyFromTrade(pairedTrades[1], bookLevel)
		if keyToPayOrReceiveToTrades[key] == nil {
			keyToPayOrReceiveToTrades[key] = make(map[string][]*Trade)
		}
		trades = keyToPayOrReceiveToTrades[key][pairedTrades[1].PayOrReceive]
		keyToPayOrReceiveToTrades[key][pairedTrades[1].PayOrReceive] = append(trades, pairedTrades[1])
	}

	return keyToPayOrReceiveToTrades
}

func generateKeyFromTrade(trade *Trade, bookLevel bool) string {
	key := fmt.Sprintf(KEY_FORMAT, trade.Party, trade.Currency, trade.MaturityDate.Format(DATE_FORMAT))
	if bookLevel {
		return fmt.Sprintf("%s_%s", key, trade.Book)
	}
	return key
}

func generateCompressionRate(originalNotional, newNotional uint64) string {
	if originalNotional == 0 {
		return "100%"
	}

	compressionRate := (float64(originalNotional) - float64(newNotional)) / float64(originalNotional) * 100

	if compressionRate == 100 {
		return "100%"
	}
	return fmt.Sprintf("%.2f%%", compressionRate)
}

func generateCompressionType(newNotional uint64) CompressionType {
	if newNotional == 0 {
		return TERMINATION
	}
	return PARTIAL
}

func sumNotional(trades []*Trade) uint64 {
	var sum uint64 = 0
	for _, trade := range trades {
		sum += trade.Notional
	}
	return sum
}

func (handler *MainHandler) GetCompressionReportAsCSV() (string, error) {
	compressionResults := handler.CompressionEngine.CompressionResults

	sort.Slice(compressionResults, func(i, j int) bool {
		if compressionResults[i].Party != compressionResults[j].Party {
			return compressionResults[i].Party < compressionResults[j].Party
		}
		if compressionResults[i].Currency != compressionResults[j].Currency {
			return compressionResults[i].Currency < compressionResults[j].Currency
		}
		if compressionResults[i].MaturityDate != compressionResults[j].MaturityDate {
			timeI, _ := time.Parse(DATE_FORMAT, compressionResults[i].MaturityDate)
			timeJ, _ := time.Parse(DATE_FORMAT, compressionResults[j].MaturityDate)
			return timeI.Before(timeJ)
		}
		return compressionResults[i].PayOrReceive < compressionResults[j].PayOrReceive
	})

	compressionResultsBytes, err := gocsv.MarshalBytes(compressionResults)
	if err != nil {
		return "", err
	}

	result := base64.StdEncoding.EncodeToString(compressionResultsBytes)
	return result, nil
}

func (handler *MainHandler) GetCompressionReportBookLevelAsCSV() (string, error) {
	bookLevelCompressionResults := handler.CompressionEngine.BookLevelCompressionResults

	sort.Slice(bookLevelCompressionResults, func(i, j int) bool {
		if bookLevelCompressionResults[i].Party != bookLevelCompressionResults[j].Party {
			return bookLevelCompressionResults[i].Party < bookLevelCompressionResults[j].Party
		}
		if bookLevelCompressionResults[i].Book != bookLevelCompressionResults[j].Book {
			return bookLevelCompressionResults[i].Book < bookLevelCompressionResults[j].Book
		}
		if bookLevelCompressionResults[i].Currency != bookLevelCompressionResults[j].Currency {
			return bookLevelCompressionResults[i].Currency < bookLevelCompressionResults[j].Currency
		}
		if bookLevelCompressionResults[i].MaturityDate != bookLevelCompressionResults[j].MaturityDate {
			timeI, _ := time.Parse(DATE_FORMAT, bookLevelCompressionResults[i].MaturityDate)
			timeJ, _ := time.Parse(DATE_FORMAT, bookLevelCompressionResults[j].MaturityDate)
			return timeI.Before(timeJ)
		}
		return bookLevelCompressionResults[i].PayOrReceive < bookLevelCompressionResults[j].PayOrReceive
	})

	bookLevelCompressionResultsBytes, err := gocsv.MarshalBytes(bookLevelCompressionResults)
	if err != nil {
		return "", err
	}

	result := base64.StdEncoding.EncodeToString(bookLevelCompressionResultsBytes)
	return result, nil
}
