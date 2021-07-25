package api

type CompressTradesReq struct {
	RequestID string `json:"request_id,omitempty"`
	InputFiles []File `json:"input_files"`
}

type File struct {
	FileName    string `json:"file_name"`
	FileContent string `json:"file_content"`
}

type CompressTradesResp struct {
	RequestID string 	`json:"request_id"`
	Exclusion                  string     `json:"exclusion"`
	CompressionReport          string     `json:"compression_report"`
	CompressionReportBookLevel string     `json:"compression_report_book_level"`
	Proposals                  []Proposal `json:"proposals"`
	DataCheck                  string     `json:"data_check"`
	Error                      string     `json:"error,omitempty"`
}

type Proposal struct {
	Party    string `json:"party"`
	Proposal string `json:"proposal"`
}
