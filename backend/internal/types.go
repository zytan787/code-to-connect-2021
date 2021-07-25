package internal

import "time"

const DATE_FORMAT = "2006/01/02"
const CCPTRADEID_PREFIX = "CCP"
const KEY_FORMAT = "%s_%s_%s"
const KEY_WITHOUT_PARTY_FORMAT = "%s_%s"

type RawTrade struct {
	Party        string `csv:"Party"`
	Book         string `csv:"Book"`
	TradeID      string `csv:"TradeID"`
	PayOrReceive string `csv:"PAY/RECEIVE"`
	Currency     string `csv:"Currency"`
	MaturityDate string `csv:"MaturityDate"`
	Cpty         string `csv:"Cpty"`
	CCPTradeID   string `csv:"CCPTradeID"`
	Notional     string `csv:"Notional"`
}

type ExcludedTrade struct {
	Party        string `csv:"Party"`
	Book         string `csv:"Book"`
	TradeID      string `csv:"TradeID"`
	PayOrReceive string `csv:"PAY/RECEIVE"`
	Currency     string `csv:"Currency"`
	MaturityDate string `csv:"MaturityDate"`
	Cpty         string `csv:"Cpty"`
	CCPTradeID   string `csv:"CCPTradeID"`
	Notional     string `csv:"Notional"`
	Error        string `csv:"Error"`
}

type Trade struct {
	Party        string
	Book         string
	TradeID      string
	PayOrReceive string
	Currency     string
	MaturityDate time.Time
	Cpty         string
	CCPTradeID   string
	Notional     uint64
}

type CompressionType string

const (
	TERMINATION CompressionType = "Termination"
	PARTIAL     CompressionType = "Partial"
)

type CompressionResult struct {
	Party            string          `csv:"Party"`
	Currency         string          `csv:"Currency"`
	MaturityDate     string          `csv:"MaturityDate"`
	PayOrReceive     string          `csv:"PAY/RECEIVE"`
	CompressionType  CompressionType `csv:"CompressionType"`
	OriginalNotional string          `csv:"Original_Notional"`
	Notional         string          `csv:"Notional"`
	CompressionRate  string          `csv:"CompressionRate"`
}

type CompressionResultBookLevel struct {
	Party            string          `csv:"Party"`
	Book             string          `csv:"Book"`
	Currency         string          `csv:"Currency"`
	MaturityDate     string          `csv:"MaturityDate"`
	PayOrReceive     string          `csv:"PAY/RECEIVE"`
	CompressionType  CompressionType `csv:"CompressionType"`
	OriginalNotional string          `csv:"Original_Notional"`
	Notional         string          `csv:"Notional"`
	CompressionRate  string          `csv:"CompressionRate"`
}

//TODO refactor trade into proposal?
type Proposal struct {
	Party        string     `csv:"Party"`
	Book         string     `csv:"Book"`
	TradeID      string     `csv:"TradeID"`
	PayOrReceive string     `csv:"PAY/RECEIVE"`
	Currency     string     `csv:"Currency"`
	MaturityDate string     `csv:"MaturityDate"`
	Cpty         string     `csv:"Cpty"`
	CCPTradeID   string     `csv:"CCPTradeID"`
	Notional     uint64     `csv:"Notional"`
	Action       ActionType `csv:"Action"`
}

type ActionType string

const (
	PENDING ActionType = "PENDING"
	KEEP    ActionType = ""
	CANCEL  ActionType = "CXL"
	ADD     ActionType = "ADD"
)

type DataCheckResult struct {
	Party            string `csv:"Party"`
	TotalIn          uint64 `csv:"TotalIn"`
	TotalOut         uint64 `csv:"TotalOut"`
	NetOut           int    `csv:"NetOut"`
	OriginalNotional uint64 `csv:"Original_Notional"`
	Notional         uint64 `csv:"Notional"`
	Reduced          bool   `csv:"Reduced"`
}

//type Comparable interface {
//	CompareTo(b Comparable) int
//}
//
//func (a *RawTrade) CompareTo(b *RawTrade) int {
//	if a.Party < b.Party {
//		return -1
//	}
//	if a.Party > b.Party {
//		return 1
//	}
//	return 0
//}
