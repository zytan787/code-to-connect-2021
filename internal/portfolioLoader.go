package internal

import (
	"encoding/base64"
	"fmt"
	"github.com/gocarina/gocsv"
	"sort"
	"strconv"
	"strings"
	"time"
)

type PortfolioLoader struct {
	PartyToRawTrades               map[string][]*RawTrade
	CcpTradeIDToCompressibleTrades map[string][]*Trade
	//TODO one more field to keep track party -> trades?
	ExcludedTrades                 []*RawTrade
}

func (handler *MainHandler) LoadPortfolio(partyATradesInput string, multiPartyTradesInput string) error {
	partyATradesBytes, err := base64.StdEncoding.DecodeString(partyATradesInput)
	if err != nil {
		return err
	}

	multiPartyTradesBytes, err := base64.StdEncoding.DecodeString(multiPartyTradesInput)
	if err != nil {
		return err
	}

	var partyATrades []*RawTrade
	err = gocsv.UnmarshalBytes(partyATradesBytes, &partyATrades)
	if err != nil {
		return err
	}

	var multiPartyTrades []*RawTrade
	err = gocsv.UnmarshalBytes(multiPartyTradesBytes, &multiPartyTrades)
	if err != nil {
		return err
	}

	allTrades := append(partyATrades, multiPartyTrades...)
	handler.PortfolioLoader.PartyToRawTrades = groupRawTradesByParty(allTrades)
	handler.PortfolioLoader.categorizeRawTrades()

	return nil
}

func groupRawTradesByParty(allRawTrades []*RawTrade) map[string][]*RawTrade {
	partyToRawTrades := make(map[string][]*RawTrade)

	for _, rawTrade := range allRawTrades {
		partyToRawTrades[rawTrade.Party] = append(partyToRawTrades[rawTrade.Party], rawTrade)
	}

	return partyToRawTrades
}

func (portfolioLoader *PortfolioLoader) categorizeRawTrades() {
	cleanTrades := make(map[string][]*Trade)
	excludedTrades := make([]*RawTrade, 0)

	var cleanTrade *Trade
	for _, rawTrades := range portfolioLoader.PartyToRawTrades {
		for _, rawTrade := range rawTrades {
			cleanTrade = portfolioLoader.cleanRawTrade(rawTrade)
			if cleanTrade == nil {
				excludedTrades = append(excludedTrades, rawTrade)
			} else {
				cleanTrades[rawTrade.CCPTradeID] = append(cleanTrades[rawTrade.CCPTradeID], cleanTrade)
			}
		}
	}

	compressibleTrades := make(map[string][]*Trade)
	var compressible bool
	for CCPTradeID, seenCleanTrades := range cleanTrades {
		compressible = false

		if len(seenCleanTrades) == 2 {
			err := verifyPairedTrades(seenCleanTrades[0], seenCleanTrades[1])
			if err == nil {
				compressible = true
			}
			//TODO add error to raw trade?
		}

		if compressible {
			compressibleTrades[CCPTradeID] = seenCleanTrades
		} else {
			rawTrades := convertToRawTrades(seenCleanTrades)
			excludedTrades = append(excludedTrades, rawTrades...)
		}
	}

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

	portfolioLoader.CcpTradeIDToCompressibleTrades = compressibleTrades
	portfolioLoader.ExcludedTrades = excludedTrades
}

func verifyPairedTrades(trade1 *Trade, trade2 *Trade) error {
	if trade1.PayOrReceive == trade2.PayOrReceive {
		return fmt.Errorf("trades are not symmetrical, trade1 (Party:%s, TradeID:%s), trade2 (Party:%s, TradeID:%s): both %s",
			trade1.Party, trade1.TradeID,
			trade2.Party, trade2.TradeID,
			trade1.PayOrReceive)
	}

	if trade1.Notional != trade2.Notional {
		return fmt.Errorf("notionals are different, trade1 (Party:%s, TradeID:%s): %d, trade2 (Party:%s, TradeID:%s): %d",
			trade1.Party, trade1.TradeID, trade1.Notional,
			trade2.Party, trade2.TradeID, trade2.Notional)
	}

	if trade1.Currency != trade2.Currency {
		return fmt.Errorf("currencies are different, trade1 (Party:%s, TradeID:%s): %s, trade2 (Party:%s, TradeID:%s): %s",
			trade1.Party, trade1.TradeID, trade1.Currency,
			trade2.Party, trade2.TradeID, trade2.Currency)
	}

	if !trade1.MaturityDate.Equal(trade2.MaturityDate) {
		return fmt.Errorf("maturity dates are different, trade1 (Party:%s, TradeID:%s): %s, trade2 (Party:%s, TradeID:%s): %s",
			trade1.Party, trade1.TradeID, trade1.MaturityDate.Format(DATE_FORMAT),
			trade2.Party, trade2.TradeID, trade2.MaturityDate.Format(DATE_FORMAT))
	}

	if trade1.Cpty != trade2.Party || trade2.Cpty != trade1.Party {
		return fmt.Errorf("counterparties do not match, trade1 (Party:%s, TradeID:%s): Cpty=%s, trade2 (Party:%s, TradeID:%s): Cpty=%s",
			trade1.Party, trade1.TradeID, trade1.Cpty,
			trade2.Party, trade2.TradeID, trade2.Cpty)
	}

	return nil
}

func (portfolioLoader *PortfolioLoader) cleanRawTrade(rawTrade *RawTrade) *Trade {
	party := strings.TrimSpace(rawTrade.Party)
	if len(party) == 0 {
		return nil
	}

	book := strings.TrimSpace(rawTrade.Book)
	if len(book) == 0 {
		return nil
	}

	tradeID := strings.TrimSpace(rawTrade.TradeID)
	if len(tradeID) == 0 {
		return nil
	}

	if rawTrade.PayOrReceive != "P" && rawTrade.PayOrReceive != "R" {
		return nil
	}

	//TODO check currency

	maturityDate, err := time.Parse(DATE_FORMAT, rawTrade.MaturityDate)
	if err != nil {
		return nil
	}

	cPty := strings.TrimSpace(rawTrade.Cpty)
	if len(cPty) == 0 {
		return nil
	}

	ccpTradeID := strings.TrimSpace(rawTrade.CCPTradeID)
	if len(ccpTradeID) == 0 {
		return nil
	}
	//TODO check CCPTradeID format

	notional, err := strconv.Atoi(rawTrade.Notional)
	if err != nil {
		return nil
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
	}
}

func convertToRawTrades(cleanTrades []*Trade) []*RawTrade {
	rawTrades := make([]*RawTrade, len(cleanTrades))

	for i := 0; i < len(cleanTrades); i++ {
		rawTrades[i] = &RawTrade{
			Party:        cleanTrades[i].Party,
			Book:         cleanTrades[i].Book,
			TradeID:      cleanTrades[i].TradeID,
			PayOrReceive: cleanTrades[i].PayOrReceive,
			Currency:     cleanTrades[i].Currency,
			MaturityDate: cleanTrades[i].MaturityDate.Format(DATE_FORMAT),
			Cpty:         cleanTrades[i].Cpty,
			CCPTradeID:   cleanTrades[i].CCPTradeID,
			Notional:     fmt.Sprintf("%d", cleanTrades[i].Notional),
		}
	}
	return rawTrades
}

func (handler *MainHandler) GetExcludedTradesAsCSV() (string, error) {
	exclusion, err := gocsv.MarshalBytes(handler.PortfolioLoader.ExcludedTrades)
	if err != nil {
		return "", err
	}

	result := base64.StdEncoding.EncodeToString(exclusion)
	return result, nil
}
