import React from "react";
import {Router} from "dva/router";
import App from "./routes/App";
import {isLogin} from "./utils";

function redirectToLogin(nextState, replace) {
    if (!isLogin()) {
        replace({
            pathname: "/login",
            state: {
                nextPathname: nextState.location.pathname,
                nextSearch: location.search
            }
        });
    }
}

function redirectToDashboard(nextState, replace) {
    if (isLogin()) {
        replace("/dashboard");
    }
}

const cached = {};

function registerModel(app, model) {
    if (!cached[model.namespace]) {
        app.model(model);
        cached[model.namespace] = 1;
    }
}

export default function ({history, app}) {
    const routes = [
        {
            path: "/",
            component: App,
            onEnter: redirectToLogin,
            getIndexRoute(nextState, cb) {
                require.ensure(
                    [],
                    require => {
                        registerModel(app, require("./models/dashboard"));
                        cb(null, {component: require("./routes/Dashboard")});
                    },
                    "dashboard"
                );
            },
            childRoutes: [
                //dashboard
                {
                    path: "dashboard",
                    name: "dashboard",
                    getComponent(nextState, cb) {
                        require.ensure(
                            [],
                            require => {
                                registerModel(app, require("./models/dashboard"));
                                cb(null, require("./routes/Dashboard"));
                            },
                            "dashboard"
                        );
                    }
                },
                //account
                {
                    path: "account",
                    name: "account",
                    childRoutes: [
                        {
                            path: "admin",
                            name: "admin",
                            getComponent(nextState, cb) {
                                require.ensure(
                                    [],
                                    require => {
                                        registerModel(app, require("./models/account/admin"));
                                        cb(null, require("./routes/account/Admin"));
                                    },
                                    "account-admin"
                                );
                            }
                        },
                        {
                            path: "role",
                            name: "role",
                            getComponent(nextState, cb) {
                                require.ensure(
                                    [],
                                    require => {
                                        registerModel(app, require("./models/account/role"));
                                        cb(null, require("./routes/account/Role"));
                                    },
                                    "account-role"
                                );
                            }
                        },
                        {
                            path: "record",
                            name: "record",
                            getComponent(nextState, cb) {
                                require.ensure(
                                    [],
                                    require => {
                                        registerModel(app, require("./models/account/record"));
                                        cb(null, require("./routes/account/Record"));
                                    },
                                    "account-record"
                                );
                            }
                        }
                    ]
                },
                // chatSystem
                {
                    path: "chatSystem",
                    name: "chatSystem",
                    childRoutes: [
                        {
                            path: "sensitiveWords",
                            name: "sensitiveWords",
                            getComponent(nextState, cb) {
                                require.ensure(
                                    [],
                                    require => {
                                        registerModel(
                                            app,
                                            require("./models/chatSystem/sensitiveWords")
                                        );
                                        cb(null, require("./routes/ChatSystem/sensitiveWords"));
                                    },
                                    "sensitiveWords"
                                );
                            }
                        },
                        {
                            path: "searchChatInfo",
                            name: "searchChatInfo",
                            getComponent(nextState, cb) {
                                require.ensure(
                                    [],
                                    require => {
                                        registerModel(
                                            app,
                                            require("./models/chatSystem/searchChatInfo")
                                        );
                                        cb(null, require("./routes/ChatSystem/searchChatInfo"));
                                    },
                                    "searchChatInfo"
                                );
                            }
                        }, {
                            path: "editConfig",
                            name: "editConfig",
                            getComponent(nextState, cb) {
                                require.ensure(
                                    [],
                                    require => {
                                        registerModel(
                                            app,
                                            require("./models/chatSystem/editConfig")
                                        );
                                        cb(null, require("./routes/ChatSystem/editConfig"));
                                    },
                                    "editConfig"
                                );
                            }
                        },
                        {
                            path: "editChatStoreTime",
                            name: "editChatStoreTime",
                            getComponent(nextState, cb) {
                                require.ensure(
                                    [],
                                    require => {
                                        registerModel(
                                            app,
                                            require("./models/chatSystem/editChatStoreTime")
                                        );
                                        cb(null, require("./routes/ChatSystem/editChatStoreTime"));
                                    },
                                    "editChatStoreTime"
                                );
                            }
                        }
                    ]
                },
                //no-power
                {
                    path: "no-power",
                    name: "no-power",
                    getComponent(nextState, cb) {
                        require.ensure(
                            [],
                            require => {
                                cb(null, require("./routes/NoPower"));
                            },
                            "no-power"
                        );
                    }
                },
            ]
        },
        //login
        {
            path: "login",
            name: "login",
            onEnter: redirectToDashboard,
            getComponent(nextState, cb) {
                require.ensure(
                    [],
                    require => {
                        cb(null, require("./routes/Login"));
                    },
                    "login"
                );
            }
        },
        //*
        {
            path: "*",
            name: "error",
            getComponent(nextState, cb) {
                require.ensure(
                    [],
                    require => {
                        cb(null, require("./routes/Error"));
                    },
                    "error"
                );
            }
        }
    ];

    return <Router history={history} routes={routes}/>;
}
