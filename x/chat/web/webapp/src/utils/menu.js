//power = { 1: "查看菜单", 2: "查看页面", 3: "新增", 4: "修改", 5: "删除", 6："查看详情" 7: "审核", 8: "上传"，9:"状态" }
const menu = [
    //dashboard
    {
        id: 1,
        key: "dashboard",
        name: "管理平台",
        icon: "laptop",
        power: [1, 2]
    },
    //account
    {
        id: 100,
        key: "account",
        name: "用户管理",
        icon: "user",
        clickable: false,
        power: [1],
        children: [
            {
                id: 101,
                key: "admin",
                name: "用户列表",
                power: [1, 2, 3, 4, 5, 6, 7, 8]
            },
            {
                id: 102,
                key: "role",
                name: "权限分组",
                power: [1, 2, 3, 4, 5, 6]
            },
            {
                id: 103,
                key: "record",
                name: "操作记录",
                power: [1, 2, 3, 4, 5, 6]
            }
        ]
    },
    {
        id: 200,
        key: "chatSystem",
        name: "聊天系统",
        icon: "exception",
        clickable: false,
        power: [1],
        children: [
            {
                id: 201,
                key: "sensitiveWords",
                name: "敏感词文件",
                power: [1, 2, 3, 4, 5, 6]
            }, {
                id: 202,
                key: "searchChatInfo",
                name: "查询聊天记录",
                power: [1, 2, 3, 4, 5, 6]
            },{
                id: 203,
                key: "editConfig",
                name: "修改聊天记录保存数目",
                power: [1, 2, 3, 4, 5, 6]
            },
            {
                id: 204,
                key: "editChatStoreTime",
                name: "修改聊天记录保存时间",
                power: [1, 2, 3, 4, 5, 6]
            },
        ]
    },
];

export default menu;
