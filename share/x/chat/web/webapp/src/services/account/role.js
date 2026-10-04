import { request, apiConfig } from "../../utils";

export async function query(params) {
  return request(apiConfig.accountGetRole, {
    method: "post",
    data: params
  });
}

export async function create(params) {
  return request(apiConfig.accountSetRole, {
    method: "post",
    data: params
  });
}

export async function update(params) {
  return request(apiConfig.accountSetRole, {
    method: "post",
    data: params
  });
}

export async function remove(params) {
  return request(apiConfig.accountDelRole, {
    method: "post",
    data: params
  });
}
