export const postRequest = async <Req, Resp>(
  params: Req,
  endpoint: string
): Promise<Resp> => {
  const body = JSON.stringify(params);
  const response: Response = await fetch(
    //TODO use env var
    // `${process.env.NEXT_PUBLIC_BACKEND_HOST}${endpoint}`,
    `http://localhost:8080${endpoint}`,
    { method: "POST", body }
  );

  const result: Resp = await response.json();
  return result;
};
