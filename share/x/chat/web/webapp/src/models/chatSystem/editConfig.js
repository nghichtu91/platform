import {routerRedux} from "dva/router";
import {getCurPowers} from "../../utils";
import {editChannelEtcd, queryChannelEtcd} from "../../services/chatSystem";
import {message} from "antd";

export default {
    namespace: "editConfig",
    state: {
        curItem: {},
    },
    subscriptions: {
        setup({dispatch, history}) {
            history.listen(location => {
                const pathName = location.pathname;
                if (pathName === "/chatSystem/editConfig") {
                    const curPowers = getCurPowers(pathName);
                    if (curPowers) {
                        dispatch({type: "query"});
                        dispatch({type: "app/changeCurPowers", payload: {curPowers}});
                    } else {
                        dispatch(routerRedux.push({pathname: "/no-power"}));
                    }
                }
            });
        }
    },

    effects: {
        * query(_, {call, put}) {
            const data = yield call(queryChannelEtcd);
            if (data.success === true) {
                let curItem = {}
                Object.entries(data.data).map(([key, value]) => {
                    curItem[key.split('/').pop()] = value
                })
                yield put({
                    type: "querySuccess",
                    payload: {
                        curItem: curItem,
                    }
                });
                message.success('成功！')
            }
        },
        * editChannelEtcd({payload}, {call, put}) {
            const data = yield call(editChannelEtcd, payload);
            if (data.success === true) {
                yield put({type: "query"});
                message.success('成功！')
            }
        }
    },

    reducers: {
        querySuccess(state, action) {
            return {
                ...state,
                ...action.payload
            };
        }
    }
};
