const typeMap = new Map([["活动", "1"], ["更新", "2"], ["最新", "3"]]);

const kindMap = new Map([
    ["通常", "0"],
    ["维护", "1"],
    ["强制更新", "2"],
    ["开服小游戏", "3"]
]);

const locationMap = new Map([
    ["内外都显示", "0"],
    ["游戏内", "1"],
    ["游戏外", "2"]
]);

const cornerMarkMap = new Map([
    ["无", 0],
    ["版本", 1],
    ["公告", 2],
]);

const statusMap = {
    0: "正常",
    1: "已过期",
    2: "将来生效"
};

const statusColorMap = {
    0: "green",
    1: "red",
    2: "blue"
};

// 背包映射
const bagNameMap = {
    1: 'materialbag', // 养成材料
    2: 'equipbag', // 装备
    3: 'herobag', // 武将碎片
    4: 'consumablebag' // 消耗品
};

// 排行榜映射
const rankTypeMap = {
    rank_elo: '演武排行榜',
    rank_lvl_pass_time_1: '挑战-张角排行榜',
    rank_lvl_pass_time_2: '挑战-董卓排行榜',
};

const getTypeByValue = t => {
    for (let [k, v] of typeMap) {
        if (t == v) {
            return k;
        }
    }
};

const getKindByValue = t => {
    for (let [k, v] of kindMap) {
        if (t == v) {
            return k;
        }
    }
};

const getLocationByValue = t => {
    for (let [k, v] of locationMap) {
        if (t == v) {
            return k;
        }
    }
};

export {
    typeMap,
    kindMap,
    locationMap,
    cornerMarkMap,
    statusMap,
    statusColorMap,
    bagNameMap,
    rankTypeMap,
    getTypeByValue,
    getKindByValue,
    getLocationByValue
};
