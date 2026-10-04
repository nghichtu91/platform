import { request, apiConfig } from "../utils";

export async function query(params) {
  return request(apiConfig.dashboard, {
    method: "get",
    data: params
  });
}
