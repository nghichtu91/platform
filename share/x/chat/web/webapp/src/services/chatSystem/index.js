import {apiConfig, request} from '../../utils'

export async function uploadSensitiveWordsFile(params) {
    return request(apiConfig.uploadSensitiveWordsFile, {
        method: "post",
        data: params
    });
}

export async function downloadSensitiveWordsFile(params) {
    let opt, temp, x;
    temp = document.createElement("form");
    temp.action = apiConfig.downloadSensitiveWordsFile;
    temp.method = "post";
    temp.style.display = "none";
    for (x in params) {
        opt = document.createElement("input");
        opt.name = x;
        opt.value = params[x];
        temp.appendChild(opt);
    }
    document.body.appendChild(temp);
    temp.submit();
    document.body.removeChild(temp);

}

export async function searchChatInfo(params) {
    return request(apiConfig.searchChatInfo, {
        method: "post",
        data: params
    });
}

export async function getSid(params) {
    return request(apiConfig.getSid, {
        method: "post",
        data: params
    });
}

export async function queryChannelEtcd(params) {
    return request(apiConfig.queryChannelEtcd, {
        method: "post",
        data: params
    });
}

export async function editChannelEtcd(params) {
    return request(apiConfig.editChannelEtcd, {
        method: "post",
        data: params
    });
}


export async function editChatStoreTime(params) {
    return request(apiConfig.editChatStoreTime, {
        method: "post",
        data: params
    });
}

export async function queryChatStoreTime(params) {
    return request(apiConfig.queryChatStoreTime, {
        method: "post",
        data: params
    });
}

