package internal

import (
	"encoding/base64"
	"fmt"
	"github.com/gocarina/gocsv"
	"github.com/zytan787/code-to-connect-2021/api"
	"sort"
	"strconv"
	"strings"
)

type EventGenerator struct {
	NewCcpTradeIndex      uint32
	PartyToNewTradeIndex  map[string]uint32
	KeyToProposals        map[string][]*Proposal
	CcpTradeIDToProposals map[string][]*Proposal
	KeyToVisitState       map[string]VisitState
	KeyToNotional         map[string]int
	//TODO party to proposals?

	//TODO create new struct to combine all key to ... maps
}

//TODO check if need visit state
type VisitState uint8

const (
	NOT_VISITED VisitState = iota
	VISITED
)

//TODO change to handle dynamic party names
const DEFAULT_CPTY = "A"

func (handler *MainHandler) GenerateProposals() error {
	handler.prepareDetailsForNewTrades()

	// create proposal for each compressible trade
	keyToProposals := make(map[string][]*Proposal)
	ccpTradeIDToProposals := make(map[string][]*Proposal)
	keyToVisitState := make(map[string]VisitState)

	var key string
	var proposal *Proposal
	for ccpTradeID, trades := range handler.PortfolioLoader.CcpTradeIDToCompressibleTrades {
		proposal = createNewProposalFromTrade(trades[0])
		key = generateKeyFromTrade(trades[0], false)
		keyToProposals[key] = append(keyToProposals[key], proposal)
		ccpTradeIDToProposals[ccpTradeID] = append(ccpTradeIDToProposals[ccpTradeID], proposal)
		keyToVisitState[key] = NOT_VISITED

		proposal = createNewProposalFromTrade(trades[1])
		key = generateKeyFromTrade(trades[1], false)
		keyToProposals[key] = append(keyToProposals[key], proposal)
		ccpTradeIDToProposals[ccpTradeID] = append(ccpTradeIDToProposals[ccpTradeID], proposal)
		keyToVisitState[key] = NOT_VISITED
	}
	handler.EventGenerator.KeyToProposals = keyToProposals
	handler.EventGenerator.CcpTradeIDToProposals = ccpTradeIDToProposals
	handler.EventGenerator.KeyToVisitState = keyToVisitState

	// cancel trades with compression type = Termination
	keyToNotional := make(map[string]int)
	var notional int
	for _, compressionResult := range handler.CompressionEngine.CompressionResults {
		key = generateKey(compressionResult.Party, compressionResult.Currency, compressionResult.MaturityDate)
		if compressionResult.CompressionType == TERMINATION {
			if proposals, ok := keyToProposals[key]; ok {
				for _, proposal = range proposals {
					if proposal.PayOrReceive == compressionResult.PayOrReceive {
						handler.EventGenerator.changeActionType(proposal.CCPTradeID, CANCEL)
					}
				}
			}
		} else {
			notional, _ = strconv.Atoi(compressionResult.Notional)
			if compressionResult.PayOrReceive == "P" {
				keyToNotional[key] = -notional
			} else {
				keyToNotional[key] = notional
			}
		}
	}
	handler.EventGenerator.KeyToNotional = keyToNotional

	//keys := make([]string, len(keyToProposals))
	//i := 0
	//for key, _ = range keyToProposals {
	//	keys[i] = key
	//	i++
	//}
	keysHasPrefixDefaultCPty := make([]string, 0, len(keyToProposals))
	for key, _ = range keyToProposals {
		if strings.HasPrefix(key, DEFAULT_CPTY) {
			keysHasPrefixDefaultCPty = append(keysHasPrefixDefaultCPty, key)
		} else {
			handler.EventGenerator.tallyNotionalByKey(key)
		}
	}

	var target int
	var proposals, payingProposals, receivingProposals []*Proposal
	for _, key = range keysHasPrefixDefaultCPty {
		target = handler.EventGenerator.KeyToNotional[key]

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
			handler.EventGenerator.fixDifferenceRecursive(payingProposals, receivingProposals, uint64(abs(target)), "P", "")
		} else if target > 0 {
			handler.EventGenerator.fixDifferenceRecursive(payingProposals, receivingProposals, uint64(target), "R", "")
		}
	}

	return nil
}

func (eventGenerator *EventGenerator) tallyNotionalByKey(key string) {
	if eventGenerator.KeyToVisitState[key] == VISITED {
		return
	}

	proposals := eventGenerator.KeyToProposals[key]
	pendingProposals := filterProposalsByActionType(proposals, PENDING)

	addedSum := sumAddedProposalsNotional(proposals)
	target := eventGenerator.KeyToNotional[key] - addedSum
	pendingProposals = sortProposalsByNotional(pendingProposals, false)
	proposalsToKeep := findNSum(pendingProposals, target)

	if target == 0 {
		return
	}

	var payOrReceive string
	if proposalsToKeep != nil {
		for _, proposalToKeep := range proposalsToKeep {
			eventGenerator.changeActionType(proposalToKeep.CCPTradeID, KEEP)
		}
		proposalsToCancel := filterProposalsByActionType(proposals, PENDING)
		for _, proposalToCancel := range proposalsToCancel {
			eventGenerator.changeActionType(proposalToCancel.CCPTradeID, CANCEL)
		}
	} else {
		for _, pendingProposal := range pendingProposals {
			eventGenerator.changeActionType(pendingProposal.CCPTradeID, CANCEL)
		}

		if target > 0 {
			payOrReceive = "R"
		} else {
			payOrReceive = "P"
		}

		eventGenerator.addPairedProposalsForNewTrade(
			proposals[0].Party, payOrReceive, proposals[0].Currency,
			proposals[0].MaturityDate, DEFAULT_CPTY, uint64(abs(target)))
	}

	eventGenerator.KeyToVisitState[key] = VISITED
	return
}

func (eventGenerator *EventGenerator) fixDifferenceRecursive(payingProposals []*Proposal, receivingProposals []*Proposal, amount uint64, payOrReceive string, originalCPty string) {
	if amount == 0 {
		return
	}

	var eligibleProposals []*Proposal
	if payOrReceive == "P" {
		eligibleProposals = payingProposals
	} else if payOrReceive == "R" {
		eligibleProposals = receivingProposals
	}

	for i:=0; i<len(eligibleProposals); i++ {
		if eligibleProposals[i].Notional >= amount {
			remaining := eligibleProposals[i].Notional - amount
			if len(originalCPty) > 0 {
				eventGenerator.changeCPty(eligibleProposals[i].CCPTradeID, eligibleProposals[i].Cpty, originalCPty)
			}
			eventGenerator.changeNotional(eligibleProposals[i].CCPTradeID, amount)
			originalCPty = eligibleProposals[i].Cpty
			if payOrReceive == "P" {
				eventGenerator.fixDifferenceRecursive(eligibleProposals[i+1:], receivingProposals, remaining, "R", originalCPty)
			} else if payOrReceive == "R" {
				eventGenerator.fixDifferenceRecursive(payingProposals, eligibleProposals[i+1:], remaining, "P", originalCPty)
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

func (eventGenerator *EventGenerator) changeActionType(ccpTradeID string, newActionType ActionType) {
	for _, proposal := range eventGenerator.CcpTradeIDToProposals[ccpTradeID] {
		proposal.Action = newActionType
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
			key := generateKey(proposal.Party, proposal.Currency, proposal.MaturityDate)
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
			newKey := generateKey(newProposal.Party, newProposal.Currency, newProposal.MaturityDate)
			eventGenerator.KeyToProposals[newKey] = append(eventGenerator.KeyToProposals[newKey], newProposal)
			newProposals[1] = newProposal
		}
	}

	eventGenerator.CcpTradeIDToProposals[ccpTradeID] = newProposals
}

func (handler *MainHandler) prepareDetailsForNewTrades() {
	partyToNewTradeIndex := make(map[string]uint32)
	var newCcpTradeIndex uint32 = 0

	var index, ccpIndex int
	var err error
	for party, rawTrades := range handler.PortfolioLoader.PartyToRawTrades {
		for _, rawTrade := range rawTrades {
			ccpIndex, err = strconv.Atoi(rawTrade.CCPTradeID[len(CCPTRADEID_PREFIX):])
			if err == nil {
				newCcpTradeIndex = max(newCcpTradeIndex, uint32(ccpIndex)+1)
			}

			index, err = strconv.Atoi(rawTrade.TradeID[len(party):])
			if err == nil {
				partyToNewTradeIndex[party] = max(partyToNewTradeIndex[party], uint32(index)+1)
			}
		}
	}

	handler.EventGenerator.NewCcpTradeIndex = newCcpTradeIndex
	handler.EventGenerator.PartyToNewTradeIndex = partyToNewTradeIndex
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
		Action:       PENDING,
	}
}

func max(x, y uint32) uint32 {
	if x > y {
		return x
	}
	return y
}

func findNSum(sortedProposals []*Proposal, target int) []*Proposal {
	if target == 0 {
		return nil
	}

	n := len(sortedProposals)
	if n == 0 {
		return nil
	}
	if n == 1 {
		if target == getProposalNotionalAsInt(sortedProposals[0]) {
			return []*Proposal{sortedProposals[0]}
		} else {
			return nil
		}
	}

	if n == 2 {
		l, r := 0, len(sortedProposals)-1
		var sum int

		for l < r {
			sum = getProposalNotionalAsInt(sortedProposals[l]) + getProposalNotionalAsInt(sortedProposals[r])
			if sum == target {
				return []*Proposal{sortedProposals[l], sortedProposals[r]}
			}
			if sum < target {
				l++
			} else {
				r--
			}
		}

		return nil
	}

	for i := 0; i < len(sortedProposals)-2; i++ {
		if i == 0 || (i > 0 && sortedProposals[i-1].Notional != sortedProposals[i].Notional) {
			res := findNSum(sortedProposals[i+1:], target-getProposalNotionalAsInt(sortedProposals[i]))
			if res != nil {
				return append(res, sortedProposals[i])
			}
		}
	}

	return nil
}

func getProposalNotionalAsInt(proposal *Proposal) int {
	if proposal.PayOrReceive == "P" {
		return -int(proposal.Notional)
	}
	return int(proposal.Notional)
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

func sumAddedProposalsNotional(proposals []*Proposal) int {
	sum := 0

	for _, proposal := range proposals {
		if proposal.Action == ADD {
			if proposal.PayOrReceive == "P" {
				sum -= int(proposal.Notional)
			} else {
				sum += int(proposal.Notional)
			}
		}
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

	ccpTradeID := fmt.Sprintf("%s%d", CCPTRADEID_PREFIX, eventGenerator.NewCcpTradeIndex)

	proposal1 := eventGenerator.generateProposalForNewTrade(
		party, payOrReceive, currency, maturityDate, cPty, ccpTradeID, notional)

	payOrReceive = getOppositePayOrReceive(payOrReceive)
	proposal2 := eventGenerator.generateProposalForNewTrade(
		cPty, payOrReceive, currency, maturityDate, party, ccpTradeID, notional)

	key1 := generateKey(proposal1.Party, proposal1.Currency, proposal1.MaturityDate)
	key2 := generateKey(proposal2.Party, proposal2.Currency, proposal2.MaturityDate)

	eventGenerator.KeyToProposals[key1] = append(eventGenerator.KeyToProposals[key1], proposal1)
	eventGenerator.KeyToProposals[key2] = append(eventGenerator.KeyToProposals[key2], proposal2)
	eventGenerator.CcpTradeIDToProposals[ccpTradeID] = []*Proposal{proposal1, proposal2}
	eventGenerator.NewCcpTradeIndex++
}

func (eventGenerator *EventGenerator) generateProposalForNewTrade(
	party string, payOrReceive string, currency string,
	maturityDate string, cPty string, ccpTradeID string, notional uint64) *Proposal {

	newTradeProposal := &Proposal{
		Party:        party,
		Book:         fmt.Sprintf("1BK%s", party),
		TradeID:      fmt.Sprintf("%s%d", party, eventGenerator.PartyToNewTradeIndex[party]),
		PayOrReceive: payOrReceive,
		Currency:     currency,
		MaturityDate: maturityDate,
		Cpty:         cPty,
		CCPTradeID:   ccpTradeID,
		Notional:     notional,
		Action:       ADD,
	}

	eventGenerator.PartyToNewTradeIndex[party]++

	return newTradeProposal
}

func generateKey(party string, currency string, maturityDate string) string {
	return fmt.Sprintf("%s_%s_%s", party, currency, maturityDate)
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
