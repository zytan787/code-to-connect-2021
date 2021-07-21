package api

type CompressTradesReq struct {
	PartyATradesInput     string `json:"party_a_trades_input"`
	MultiPartyTradesInput string `json:"multi_party_trades_input"`
}

type CompressTradesResp struct {
	Exclusion                  string            `json:"exclusion"`
	CompressionReport          string            `json:"compression_report"`
	CompressionReportBookLevel string            `json:"compression_report_book_level"`
	Proposals                  map[string]string `json:"proposals"`
	DataCheck                  string            `json:"data_check"`
	Error                      string            `json:"error"`
}
