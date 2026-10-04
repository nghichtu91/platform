import menu from "./menu";
import Cookie from "./cookie";

export config from "./config";
export { apiConfig } from "./apiconfig";

let request = require("./request").default;
export { request };

export { color } from "./theme";

let allPathPowers; //缓存 localStorage.getItem('allPathPowers') 数据

// 连字符转驼峰
String.prototype.hyphenToHump = function() {
  return this.replace(/-(\w)/g, function() {
    return arguments[1].toUpperCase();
  });
};

// 驼峰转连字符
String.prototype.humpToHyphen = function() {
  return this.replace(/([A-Z])/g, "-$1").toLowerCase();
};
// 驼峰转下划线
String.prototype.toLine = function() {
  return this.replace(/([A-Z])/g, "_$1").toLowerCase();
};

// 日期格式化
Date.prototype.format = function(format) {
  let o = {
    "M+": this.getMonth() + 1,
    "d+": this.getDate(),
    "h+": this.getHours(),
    "H+": this.getHours(),
    "m+": this.getMinutes(),
    "s+": this.getSeconds(),
    "q+": Math.floor((this.getMonth() + 3) / 3),
    S: this.getMilliseconds()
  };
  if (/(y+)/.test(format)) {
    format = format.replace(
      RegExp.$1,
      (this.getFullYear() + "").substr(4 - RegExp.$1.length)
    );
  }
  for (let k in o) {
    if (new RegExp("(" + k + ")").test(format)) {
      format = format.replace(
        RegExp.$1,
        RegExp.$1.length === 1 ? o[k] : ("00" + o[k]).substr(("" + o[k]).length)
      );
    }
  }
  return format;
};
// 秒 转换为 时分秒
function formateSeconds(endTime){
  let secondTime = parseInt(endTime)//将传入的秒的值转化为Number
  let min = 0// 初始化分
  let h =0// 初始化小时
  let result=''
  if(secondTime>60){//如果秒数大于60，将秒数转换成整数
    min=parseInt(secondTime/60)//获取分钟，除以60取整数，得到整数分钟
    secondTime=parseInt(secondTime%60)//获取秒数，秒数取佘，得到整数秒数
    if(min>60){//如果分钟大于60，将分钟转换成小时
      h=parseInt(min/60)//获取小时，获取分钟除以60，得到整数小时
      min=parseInt(min%60) //获取小时后取佘的分，获取分钟除以60取佘的分
    }
  }
  result=`${h.toString().padStart(2,'0')}:${min.toString().padStart(2,'0')}:${secondTime.toString().padStart(2,'0')}`
  return result
}

function equalSet(a, b) {
  const as = new Set(a);
  const bs = new Set(b);
  if (as.size !== bs.size) return false;
  for (let a of as) if (!bs.has(a)) return false;
  return true;
}

const isLogin = () => {
  return Cookie.get("login_token");
};

const userName = Cookie.get("user_name");

const setLoginIn = (loginToken, userName, power, allPathPowers) => {
  Cookie.set("login_token", loginToken);
  Cookie.set("user_power", power);
  Cookie.set("user_name", userName);
  localStorage.setItem("allPathPowers", JSON.stringify(allPathPowers));
};

const setLoginOut = () => {
  Cookie.remove("login_token");
  Cookie.remove("user_power");
  localStorage.removeItem("allPathPowers");
  allPathPowers = null;
};

const checkPower = (optionId, curPowers = []) => {
  return curPowers.some(cur => cur === optionId);
};

const getCurPowers = curPath => {
  if (!allPathPowers) {
    allPathPowers = JSON.parse(localStorage.getItem("allPathPowers"));
  }
  const curPathPower = allPathPowers && allPathPowers[curPath];
  //cur =2 检测查看页面内容权限
  if (!curPathPower || !curPathPower.find(cur => cur === 2)) {
    return false;
  }
  return curPathPower; //返回curPathPower，是为方便页面跳转验证权限后，dispatch当然权限
};

//延迟调用
const delay = (ms) => new Promise((resolve) => {

  setTimeout(resolve, ms);

});

export {
  Cookie,
  menu,
  equalSet,
  formateSeconds,
  isLogin,
  userName,
  setLoginIn,
  setLoginOut,
  checkPower,
  getCurPowers,
  delay
};
