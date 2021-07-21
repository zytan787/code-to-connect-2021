package internal

import (
	"encoding/base64"
	"fmt"
	"github.com/gocarina/gocsv"
	"sort"
	"strconv"
	"strings"
)

type EventGenerator struct {
	NewCcpTradeIndex      uint32
	PartyToNewTradeIndex map[string]uint32
	KeyToProposals map[string][]*Proposal
	CcpTradeIDToProposals map[string][]*Proposal
	KeyToVisitState map[string]VisitState
	KeyToNotional map[string]int
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
						handler.EventGenerator.performSymmetricalAction(proposal.CCPTradeID, CANCEL)
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

	for key, _ = range keyToProposals {
		if !strings.HasPrefix(key, DEFAULT_CPTY) {
			handler.EventGenerator.tallyNotionalByKey(key)
		}
	}

	return nil
}

func (eventGenerator *EventGenerator) tallyNotionalByKey(key string) {
	if eventGenerator.KeyToVisitState[key] == VISITED {
		return
	}

	proposals := eventGenerator.KeyToProposals[key]
	pendingProposals := getPendingProposals(proposals)

	if len(pendingProposals) == 0 {
		eventGenerator.KeyToVisitState[key] = VISITED
		return
	}

	addedSum := sumAddedProposalsNotional(proposals)
	target := eventGenerator.KeyToNotional[key] - addedSum
	pendingProposals = sortProposalsByNotional(pendingProposals)
	proposalsToKeep := findNSum(pendingProposals, target)

	if proposalsToKeep != nil {
		for _, proposalToKeep := range proposalsToKeep {
			eventGenerator.performSymmetricalAction(proposalToKeep.CCPTradeID, KEEP)
		}
		proposalsToCancel := getPendingProposals(proposals)
		for _, proposalToCancel := range proposalsToCancel {
			eventGenerator.performSymmetricalAction(proposalToCancel.CCPTradeID, CANCEL)
		}
	} else {
		for _, pendingProposal := range pendingProposals {
			eventGenerator.performSymmetricalAction(pendingProposal.CCPTradeID, CANCEL)
		}

		eventGenerator.addPairedProposalsForNewTrade(
			pendingProposals[0].Party, pendingProposals[0].PayOrReceive, pendingProposals[0].Currency,
			pendingProposals[0].MaturityDate, DEFAULT_CPTY, uint64(abs(target)))
	}

	eventGenerator.KeyToVisitState[key] = VISITED
	return
}

func (eventGenerator *EventGenerator) performSymmetricalAction(ccpTradeID string, action ActionType) {
	for _, proposal := range eventGenerator.CcpTradeIDToProposals[ccpTradeID] {
		proposal.Action = action
	}
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

func createNewProposalFromTrade(trade *Trade) *Proposal{
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

	for i:=0; i<len(sortedProposals)-2; i++ {
		if i == 0 || (i > 0 && sortedProposals[i-1].Notional != sortedProposals[i].Notional) {
			res := findNSum(sortedProposals[i+1:], target - getProposalNotionalAsInt(sortedProposals[i]))
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

func getPendingProposals(proposals []*Proposal) []*Proposal {
	result := make([]*Proposal, 0, len(proposals))

	for _, proposal := range proposals {
		if proposal.Action == PENDING {
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

func sortProposalsByNotional(proposals []*Proposal) []*Proposal {
	sort.Slice(proposals, func(i, j int) bool {
		return proposals[i].Notional < proposals[j].Notional
	})

	return proposals
}

func (eventGenerator *EventGenerator) addPairedProposalsForNewTrade(
	party string, payOrReceive string, currency string, maturityDate string, cPty string, notional uint64) {

	ccpTradeID := fmt.Sprintf("%s%d", CCPTRADEID_PREFIX, eventGenerator.NewCcpTradeIndex)

	proposal1 := eventGenerator.generateProposalForNewTrade(
		party, payOrReceive, currency, maturityDate, cPty, ccpTradeID, notional)

	if payOrReceive == "P" {
		payOrReceive = "R"
	} else {
		payOrReceive = "P"
	}
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

func (handler *MainHandler) GetProposalsAsCSV() (map[string]string, error) {
	partyToProposals := make(map[string][]*Proposal)

	for _, proposals := range handler.EventGenerator.KeyToProposals {
		partyToProposals[proposals[0].Party] = append(partyToProposals[proposals[0].Party], proposals...)
	}

	result := make(map[string]string)

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

		result[party] = base64.StdEncoding.EncodeToString(proposalsBytes)
	}

	return result, nil
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}