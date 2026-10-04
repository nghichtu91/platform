export default {
  name: "gm_tools",
  prefix: "",
  footerText: "Copyright © 2019 钛核互动技术部出品",
  logoSrc:
    "http://nzr2ybsda.qnssl.com/images/48838/Fh1xIa4zBqBQv1W504gBOiW7nbu1.png?imageMogr2/strip/thumbnail/300x300%3E/format/png",
  iconFontUrl: "//at.alicdn.com/t/font_c4y7asse3q1cq5mi.js",
  logoText: "聊天统计系统",
  needLogin: true,
  api: {
    userLogin: "/user/login",
    userLogout: "/user/logout",
    userInfo: "/userInfo",
    users: "/users",
    user: "/user/:id",
    dashboard: "/dashboard"
  }
};
