/**
 * 所有接口地址定义
 */
const apiConfig = {
    // 管理员
    accountUpdateUser: "/api/user/update",
    accountQueryUser: "/api/user/get",
    accountDelUser: "/api/user/del",
    accountGetOneUser: "/api/user/getOne",
    // 角色
    accountGetRole: "/api/role/get",
    accountSetRole: "/api/role/set",
    accountDelRole: "/api/role/del",
    // 用户
    accountUser: "/api/user",
    accountUserItem: "/api/userItem",
    // 操作记录
    queryRecord: "/api/record/query",
    // 获取token
    appAuthToken: "/api/login",
    appAdminCheck: "/admin/check",
    appLogout: "/api/logout",
    appUserInfo: "/api/userInfo",
    // 聊天系统
    uploadSensitiveWordsFile: "/api/chatSystem/uploadSensitiveWordsFile",
    downloadSensitiveWordsFile: "/api/chatSystem/downloadSensitiveWordsFile",
    searchChatInfo: "/api/chatSystem/searchChatInfo",
    queryChannelEtcd: "/api/chatSystem/queryChannelEtcd",
    editChannelEtcd: "/api/chatSystem/editChannelEtcd",
    editChatStoreTime: "/api/chatSystem/editChatStoreTime",
    queryChatStoreTime: "/api/chatSystem/queryChatStoreTime",
    // 获取大区ID
    getSid: "/api/get/shardIds",

    // more apiConfig
};

export {apiConfig};
