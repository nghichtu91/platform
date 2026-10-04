import { request, apiConfig } from "../../utils";

export async function query(params) {
  return request(apiConfig.accountQueryUser, {
    method: "post",
    data: params
  });
}

export async function get(params) {
  return request(apiConfig.accountGetOneUser, {
    method: "post",
    data: params
  });
}

export async function create(params) {
  return request(apiConfig.accountUpdateUser, {
    method: "post",
    data: params
  });
}

export async function remove(params) {
  return request(apiConfig.accountDelUser, {
    method: "post",
    data: params
  });
}

export async function update(params) {
  return request(apiConfig.accountUpdateUser, {
    method: "post",
    data: params
  });
}

export async function queryRecord(params) {
  return request(apiConfig.queryRecord, {
    method: "post",
    data: params
  });
}
