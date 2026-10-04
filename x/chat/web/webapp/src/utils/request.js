import axios from "axios";
import { message } from "antd";
import { stringify } from "qs";
import Cookie from "./cookie";
import NProgress from "nprogress";

//message 全局配置
message.config({
  top: 50
});

axios.defaults.baseURL = "";
axios.defaults.headers.post["Content-Type"] =
  "application/x-www-form-urlencoded; charset=UTF-8";
axios.defaults.headers.post["Access-Control-Allow-Headers"] = "*";

const fetch = (url, options) => {
  let defaultContent = "application/x-www-form-urlencoded; charset=UTF-8";
  const { method = "get", data } = options;
  if (
    data !== undefined &&
    data.ContentType !== undefined &&
    data.ContentType !== ""
  ) {
    defaultContent = data.ContentType;
  }
  let x = axios.create({
    baseURL: "",
    headers: {
      "Content-Type": defaultContent,
      "Access-Control-Allow-Headers": "*"
    }
  });

  if (defaultContent === "multipart/form-data") {
    return x.post(url, data.data);
  }

  switch (method.toLowerCase()) {
    case "get":
      return x.get(url, { params: data });
    case "delete":
      return x.delete(url, { data });
    case "head":
      return x.head(url, data);
    case "post":
      return x.post(url, stringify(data, { indices: false }));
    case "put":
      return x.put(url, stringify(data));
    case "patch":
      return x.patch(url, data);
    default:
      return axios(options);
  }
};

function checkStatus(res) {
  if (res.status >= 200 && res.status < 300) {
    return res;
  }
}

function handelData(res) {
  NProgress.done();
  const data = res.data;
  if (data && data.message && !data.success) {
    message.error(data.message);
  }
  // else if(data && data.msg && data.success) {
  //   message.success(data.msg)
  // }
  console.log(data)
  return { ...data.data, success: data.success || data.message === "Success", message: data.message };
}

function handleError(error) {
  NProgress.done();
  const data = error.response.data;
  if (data !== "") {
    message.error(data);
  } else {
    message.error("未知错误");
  }
  if (error.response.status === 400) {
    Cookie.remove("login_token");
    location.href = "/#/login";
  }
  return { success: false };
}

export default function request(url, options) {
  NProgress.start();
  return fetch(url, options)
    .then(checkStatus)
    .then(handelData)
    .catch(handleError);
}

export function get(url, options) {
  return request(url, { ...options, method: "get" });
}

export function post(url, options) {
  return request(url, { ...options, method: "post" });
}

export function put(url, options) {
  return request(url, { ...options, method: "put" });
}

export function deleted(url, options) {
  return request(url, { ...options, method: "deleted" });
}
