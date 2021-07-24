import { CompressTradesReq, CompressTradesResp } from "./types";
import { postRequest } from "../utils";

export const compressTradesRequest = async (
  compressTradesReq: CompressTradesReq
) => {
  const payload: CompressTradesReq = compressTradesReq;
  const resp = await postRequest<CompressTradesReq, CompressTradesResp>(
    payload,
    "/compress_trades"
  );

  const { error } = resp;

  if (error) {
    throw new Error(error);
  }

  return resp;
};
