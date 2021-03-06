package internal

import (
	"encoding/base64"
	"github.com/gocarina/gocsv"
	"github.com/zytan787/code-to-connect-2021/api"
	"sort"
)

type DataChecker struct {
	PartyToDataCheckResult map[string]*DataCheckResult
}

func (handler *MainHandler) CheckData() error {
	partyToProposals := make(map[string][]*Proposal)

	for _, proposals := range handler.EventGenerator.KeyToProposals {
		partyToProposals[proposals[0].Party] = append(partyToProposals[proposals[0].Party], proposals...)
	}

	partyToDataCheckResult := make(map[string]*DataCheckResult)
	var totalIn, totalOut, originalNotional, notional uint64
	for party, proposals := range partyToProposals {
		totalIn, totalOut, originalNotional, notional = 0, 0, 0, 0
		for _, proposal := range proposals {
			if proposal.Action != ADD {
				if proposal.PayOrReceive == "P" {
					totalOut += proposal.Notional
				} else {
					totalIn += proposal.Notional
				}
				originalNotional += proposal.Notional
			}
			if proposal.Action != CANCEL {
				notional += proposal.Notional
			}
		}
		partyToDataCheckResult[party] = &DataCheckResult{
			Party:            party,
			TotalIn:          totalIn,
			TotalOut:         totalOut,
			NetOut:           int(totalOut) - int(totalIn),
			OriginalNotional: originalNotional,
			Notional:         notional,
			Reduced:          notional < originalNotional,
		}
	}

	handler.DataChecker.PartyToDataCheckResult = partyToDataCheckResult
	return nil
}

func (handler *MainHandler) GetStatistics() []api.Statistic {
	partyToOriginalTradeCount := make(map[string]uint64)
	partyToNewTradeCount := make(map[string]uint64)
	for _, proposals := range handler.EventGenerator.KeyToProposals {
		for _, proposal := range proposals {
			if proposal.Action != ADD {
				partyToOriginalTradeCount[proposal.Party]++
			} else {
				partyToNewTradeCount[proposal.Party]++
			}
		}
	}

	result := make([]api.Statistic, len(handler.DataChecker.PartyToDataCheckResult))

	i := 0
	for party, dataCheckResult := range handler.DataChecker.PartyToDataCheckResult {
		result[i] = api.Statistic{
			Party:              party,
			OriginalNotional:   dataCheckResult.OriginalNotional,
			NewNotional:        dataCheckResult.Notional,
			OriginalNoOfTrades: partyToOriginalTradeCount[party],
			NewNoOfTrades:      partyToNewTradeCount[party],
		}
		i++
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Party < result[j].Party
	})

	return result
}

func (handler *MainHandler) GetDataCheckResultsAsCSV() (string, error) {
	dataCheckResults := make([]*DataCheckResult, len(handler.DataChecker.PartyToDataCheckResult))

	i := 0
	for _, dataCheckResult := range handler.DataChecker.PartyToDataCheckResult {
		dataCheckResults[i] = dataCheckResult
		i++
	}

	sort.Slice(dataCheckResults, func(i, j int) bool {
		return dataCheckResults[i].Party < dataCheckResults[j].Party
	})

	var totalIn, totalOut, originalNotional, notional uint64

	for _, dataCheckResult := range dataCheckResults {
		totalIn += dataCheckResult.TotalIn
		totalOut += dataCheckResult.TotalOut
		originalNotional += dataCheckResult.OriginalNotional
		notional += dataCheckResult.Notional
	}

	totalCheckResult := &DataCheckResult{
		Party:            "Total",
		TotalIn:          totalIn,
		TotalOut:         totalOut,
		NetOut:           int(totalOut) - int(totalIn),
		OriginalNotional: originalNotional,
		Notional:         notional,
		Reduced:          notional < originalNotional,
	}

	dataCheckResults = append(dataCheckResults, totalCheckResult)

	dataCheckResultsBytes, err := gocsv.MarshalBytes(dataCheckResults)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(dataCheckResultsBytes), nil
}
