export type CompressTradesReq = {
  input_files: InputFile[];
};

export type InputFile = {
  file_name: string;
  file_content: string;
};

export type CompressTradesResp = {
  request_id: string;
  exclusion: string;
  compression_report: string;
  compression_report_book_level: string;
  proposals: Proposal[];
  data_check: string;
  statistics: Statistic[];
  error?: string;
};

export type Proposal = {
  party: string;
  proposal: string;
};

export type Statistic = {
  party: string;
  original_notional: number;
  new_notional: number;
  original_no_of_trades: number;
  new_no_of_trades: number;
};
