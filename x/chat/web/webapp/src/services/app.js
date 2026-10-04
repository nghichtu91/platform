import {request, apiConfig} from "../utils";
import axios from "axios";
import NProgress from "nprogress";

export async function getToken(params) {
    const data = {
        client_id: "gmtool.taiyouxi.cn",
        client_secret: "st4mtjULIuh2ks6r",
        grant_type: "client_credentials"
    };
    return request(apiConfig.appAuthToken, {
        method: "post",
        data: data
    });
}

export async function getGid(params) {
    return request(apiConfig.getGid, {
        method: "post",
        data: params
    });
}

export async function login(params) {
    return request(apiConfig.appAdminCheck, {
        method: "post",
        data: params
    });
}

export async function logout(params) {
    return request(apiConfig.appLogout, {
        method: "post",
        data: params
    });
}

export async function userInfo(params) {
    return request(apiConfig.appUserInfo, {
        method: "get",
        data: params
    });
}

export async function getItemData(params) {
    NProgress.start();
    return axios.get('../asset/item2name.json').then(response => {
        // 请求成功
        NProgress.done();
        const data = response.data;
        return {data, success: true};
    }).catch(error => {
        // 请求失败，
        NProgress.done();
        return {success: false};
    });
}

export async function getHeroData(params) {
    NProgress.start();
    return axios.get('../asset/hero2name.json').then(response => {
        // 请求成功
        NProgress.done();
        const data = response.data;
        return {data, success: true};
    }).catch(error => {
        // 请求失败，
        NProgress.done();
        return {success: false};
    });
}
