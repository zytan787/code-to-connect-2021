package internal

import (
	"encoding/base64"
	"fmt"
	"github.com/gocarina/gocsv"
	"github.com/zytan787/code-to-connect-2021/api"
	"github.com/zytan787/code-to-connect-2021/internal/toolkit"
	"sort"
	"strconv"
	"strings"
)

type EventGenerator struct {
	KeyToProposals        map[string][]*Proposal
	CcpTradeIDToProposals map[string][]*Proposal
	KeyToDefaultBook      map[string]string
}

func (handler *MainHandler) GenerateProposals() error {
	// create proposal for each compressible trade
	keyToProposals := make(map[string][]*Proposal)
	ccpTradeIDToProposals := make(map[string][]*Proposal)
	keyToDefaultBook := make(map[string]string)
	keyWithoutPartyToDefaultCPty := make(map[string]string)

	var key, keyWithoutParty string
	var proposal *Proposal
	for ccpTradeID, trades := range handler.PortfolioLoader.CcpTradeIDToCompressibleTrades {
		proposal = createNewProposalFromTrade(trades[0])
		key = generateKeyFromTrade(trades[0], false)
		keyToProposals[key] = append(keyToProposals[key], proposal)
		ccpTradeIDToProposals[ccpTradeID] = append(ccpTradeIDToProposals[ccpTradeID], proposal)
		if _, ok := keyToDefaultBook[key]; !ok {
			keyToDefaultBook[key] = proposal.Book
		}
		keyWithoutParty = fmt.Sprintf(KEY_WITHOUT_PARTY_FORMAT, proposal.Currency, proposal.MaturityDate)
		if _, ok := keyWithoutPartyToDefaultCPty[keyWithoutParty]; !ok {
			keyWithoutPartyToDefaultCPty[keyWithoutParty] = proposal.Party
		}

		proposal = createNewProposalFromTrade(trades[1])
		key = generateKeyFromTrade(trades[1], false)
		keyToProposals[key] = append(keyToProposals[key], proposal)
		ccpTradeIDToProposals[ccpTradeID] = append(ccpTradeIDToProposals[ccpTradeID], proposal)
		if _, ok := keyToDefaultBook[key]; !ok {
			keyToDefaultBook[key] = proposal.Book
		}
		keyWithoutParty = fmt.Sprintf(KEY_WITHOUT_PARTY_FORMAT, proposal.Currency, proposal.MaturityDate)
		if _, ok := keyWithoutPartyToDefaultCPty[keyWithoutParty]; !ok {
			keyWithoutPartyToDefaultCPty[keyWithoutParty] = proposal.Party
		}
	}
	handler.EventGenerator.KeyToProposals = keyToProposals
	handler.EventGenerator.CcpTradeIDToProposals = ccpTradeIDToProposals
	handler.EventGenerator.KeyToDefaultBook = keyToDefaultBook

	// retrieve required notional for each key
	keyToNotional := make(map[string]int)
	var notional int
	for _, compressionResult := range handler.CompressionEngine.CompressionResults {
		key = fmt.Sprintf(KEY_FORMAT, compressionResult.Party, compressionResult.Currency, compressionResult.MaturityDate)
		if compressionResult.CompressionType != TERMINATION {
			notional, _ = strconv.Atoi(compressionResult.Notional)
			if compressionResult.PayOrReceive == "P" {
				keyToNotional[key] = -notional
			} else {
				keyToNotional[key] = notional
			}
		}
	}

	// add minimum number of trades
	var splitKey []string
	var party, payOrReceive, defaultCPty, currency, maturityDate string
	for key, _ = range keyToProposals {
		splitKey = strings.Split(key, "_")
		keyWithoutParty = strings.Join(splitKey[len(splitKey)-2:], "_")
		party = strings.Join(splitKey[:len(splitKey)-2], "_") // for cases where party name contains "_"
		if keyWithoutPartyToDefaultCPty[keyWithoutParty] != party {
			if keyToNotional[key] > 0 {
				payOrReceive = "R"
			} else {
				payOrReceive = "P"
			}

			defaultCPty = keyWithoutPartyToDefaultCPty[keyWithoutParty]
			currency = splitKey[len(splitKey)-2]
			maturityDate = splitKey[len(splitKey)-1]
			handler.EventGenerator.addPairedProposalsForNewTrade(
				party, payOrReceive, currency,
				maturityDate, defaultCPty, uint64(abs(keyToNotional[key])))
		}
	}

	// minimize total notional
	var target int
	var proposals, payingProposals, receivingProposals []*Proposal
	for keyWithoutParty, defaultCPty = range keyWithoutPartyToDefaultCPty {
		key = fmt.Sprintf("%s_%s", defaultCPty, keyWithoutParty)
		target = keyToNotional[key]

		proposals = handler.EventGenerator.KeyToProposals[key]
		proposals = filterProposalsByActionType(proposals, ADD)

		payingProposals = make([]*Proposal, 0, len(proposals))
		receivingProposals = make([]*Proposal, 0, len(proposals))

		for _, proposal = range proposals {
			if proposal.PayOrReceive == "P" {
				payingProposals = append(payingProposals, proposal)
			} else if proposal.PayOrReceive == "R" {
				receivingProposals = append(receivingProposals, proposal)
			}
		}

		payingProposals = sortProposalsByNotional(payingProposals, true)
		receivingProposals = sortProposalsByNotional(receivingProposals, true)

		if target < 0 {
			handler.EventGenerator.minimizeNotionalRecursively(payingProposals, receivingProposals, uint64(abs(target)), "P", "")
		} else if target > 0 {
			handler.EventGenerator.minimizeNotionalRecursively(payingProposals, receivingProposals, uint64(target), "R", "")
		}
	}

	return nil
}

func (eventGenerator *EventGenerator) minimizeNotionalRecursively(payingProposals []*Proposal, receivingProposals []*Proposal, amount uint64, payOrReceive string, originalCPty string) {
	if amount == 0 {
		return
	}

	var eligibleProposals []*Proposal
	if payOrReceive == "P" {
		eligibleProposals = payingProposals
	} else if payOrReceive == "R" {
		eligibleProposals = receivingProposals
	}

	for i := 0; i < len(eligibleProposals); i++ {
		if eligibleProposals[i].Notional >= amount {

			// if the notional is equal to the required amount, but there are still extra trades left
			if eligibleProposals[i].Notional == amount {
				extra := sumProposalsNotional(eligibleProposals[i+1:])
				if extra != 0 {
					amount -= extra
				}
			}

			remaining := eligibleProposals[i].Notional - amount
			if len(originalCPty) > 0 {
				eventGenerator.changeCPty(eligibleProposals[i].CCPTradeID, eligibleProposals[i].Cpty, originalCPty)
			}
			eventGenerator.changeNotional(eligibleProposals[i].CCPTradeID, amount)
			originalCPty = eligibleProposals[i].Cpty
			if payOrReceive == "P" {
				eventGenerator.minimizeNotionalRecursively(eligibleProposals[i+1:], receivingProposals, remaining, "R", originalCPty)
			} else if payOrReceive == "R" {
				eventGenerator.minimizeNotionalRecursively(payingProposals, eligibleProposals[i+1:], remaining, "P", originalCPty)
			}
			return
		} else {
			amount -= eligibleProposals[i].Notional
			if len(originalCPty) > 0 {
				eventGenerator.changeCPty(eligibleProposals[i].CCPTradeID, eligibleProposals[i].Cpty, originalCPty)
			}
		}
	}
}

func (eventGenerator *EventGenerator) changeNotional(ccpTradeID string, newNotional uint64) {
	for _, proposal := range eventGenerator.CcpTradeIDToProposals[ccpTradeID] {
		proposal.Notional = newNotional
	}
}

func (eventGenerator *EventGenerator) changeCPty(ccpTradeID string, party string, newCpty string) {
	newProposals := make([]*Proposal, 2)

	for _, proposal := range eventGenerator.CcpTradeIDToProposals[ccpTradeID] {
		if proposal.Party == party {
			proposal.Cpty = newCpty
			newProposals[0] = proposal
		} else {
			key := fmt.Sprintf(KEY_FORMAT, proposal.Party, proposal.Currency, proposal.MaturityDate)
			initialPartyProposals := eventGenerator.KeyToProposals[key]
			for i := 0; i < len(initialPartyProposals); i++ {
				if initialPartyProposals[i] == proposal {
					eventGenerator.KeyToProposals[key] = append(initialPartyProposals[:i], initialPartyProposals[i+1:]...)
					break
				}
			}
			newProposal := eventGenerator.generateProposalForNewTrade(
				newCpty, proposal.PayOrReceive, proposal.Currency,
				proposal.MaturityDate, party, ccpTradeID, proposal.Notional)
			newKey := fmt.Sprintf(KEY_FORMAT, newProposal.Party, newProposal.Currency, newProposal.MaturityDate)
			eventGenerator.KeyToProposals[newKey] = append(eventGenerator.KeyToProposals[newKey], newProposal)
			newProposals[1] = newProposal
		}
	}

	eventGenerator.CcpTradeIDToProposals[ccpTradeID] = newProposals
}

func createNewProposalFromTrade(trade *Trade) *Proposal {
	return &Proposal{
		Party:        trade.Party,
		Book:         trade.Book,
		TradeID:      trade.TradeID,
		PayOrReceive: trade.PayOrReceive,
		Currency:     trade.Currency,
		MaturityDate: trade.MaturityDate.Format(DATE_FORMAT),
		Cpty:         trade.Cpty,
		CCPTradeID:   trade.CCPTradeID,
		Notional:     trade.Notional,
		Action:       CANCEL,
	}
}

func filterProposalsByActionType(proposals []*Proposal, actionType ActionType) []*Proposal {
	result := make([]*Proposal, 0, len(proposals))

	for _, proposal := range proposals {
		if proposal.Action == actionType {
			result = append(result, proposal)
		}
	}

	return result
}

func sumProposalsNotional(proposals []*Proposal) uint64 {
	var sum uint64 = 0

	for _, proposal := range proposals {
		sum += proposal.Notional
	}

	return sum
}

func sortProposalsByNotional(proposals []*Proposal, descending bool) []*Proposal {
	if descending {
		sort.Slice(proposals, func(i, j int) bool {
			return proposals[i].Notional > proposals[j].Notional
		})
	} else {
		sort.Slice(proposals, func(i, j int) bool {
			return proposals[i].Notional < proposals[j].Notional
		})
	}

	return proposals
}

func (eventGenerator *EventGenerator) addPairedProposalsForNewTrade(
	party string, payOrReceive string, currency string, maturityDate string, cPty string, notional uint64) {

	ccpTradeID := fmt.Sprintf("%s%s", CCPTRADEID_PREFIX, toolkit.UniqueID())

	proposal1 := eventGenerator.generateProposalForNewTrade(
		party, payOrReceive, currency, maturityDate, cPty, ccpTradeID, notional)

	payOrReceive = getOppositePayOrReceive(payOrReceive)
	proposal2 := eventGenerator.generateProposalForNewTrade(
		cPty, payOrReceive, currency, maturityDate, party, ccpTradeID, notional)

	key1 := fmt.Sprintf(KEY_FORMAT, proposal1.Party, proposal1.Currency, proposal1.MaturityDate)
	key2 := fmt.Sprintf(KEY_FORMAT, proposal2.Party, proposal2.Currency, proposal2.MaturityDate)

	eventGenerator.KeyToProposals[key1] = append(eventGenerator.KeyToProposals[key1], proposal1)
	eventGenerator.KeyToProposals[key2] = append(eventGenerator.KeyToProposals[key2], proposal2)
	eventGenerator.CcpTradeIDToProposals[ccpTradeID] = []*Proposal{proposal1, proposal2}
}

func (eventGenerator *EventGenerator) generateProposalForNewTrade(
	party string, payOrReceive string, currency string,
	maturityDate string, cPty string, ccpTradeID string, notional uint64) *Proposal {

	newTradeProposal := &Proposal{
		Party:        party,
		Book:         fmt.Sprintf("%s", eventGenerator.KeyToDefaultBook[fmt.Sprintf(KEY_FORMAT, party, currency, maturityDate)]),
		TradeID:      fmt.Sprintf("%s%s", party, toolkit.UniqueID()),
		PayOrReceive: payOrReceive,
		Currency:     currency,
		MaturityDate: maturityDate,
		Cpty:         cPty,
		CCPTradeID:   ccpTradeID,
		Notional:     notional,
		Action:       ADD,
	}

	return newTradeProposal
}

func (handler *MainHandler) GetProposalsAsCSV() ([]api.Proposal, error) {
	partyToProposals := make(map[string][]*Proposal)

	for _, proposals := range handler.EventGenerator.KeyToProposals {
		partyToProposals[proposals[0].Party] = append(partyToProposals[proposals[0].Party], proposals...)
	}

	result := make([]api.Proposal, len(partyToProposals))

	i := 0
	for party, proposals := range partyToProposals {
		sort.Slice(proposals, func(i, j int) bool {
			indexI, _ := strconv.Atoi(proposals[i].TradeID[len(party):])
			indexJ, _ := strconv.Atoi(proposals[j].TradeID[len(party):])
			return indexI < indexJ
		})

		proposalsBytes, err := gocsv.MarshalBytes(proposals)
		if err != nil {
			return nil, err
		}

		proposalsString := base64.StdEncoding.EncodeToString(proposalsBytes)
		result[i] = api.Proposal{
			Party:    party,
			Proposal: proposalsString,
		}
		i += 1
	}

	return result, nil
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func getOppositePayOrReceive(payOrReceive string) string {
	if payOrReceive == "P" {
		return "R"
	} else if payOrReceive == "R" {
		return "P"
	}

	return ""
}
