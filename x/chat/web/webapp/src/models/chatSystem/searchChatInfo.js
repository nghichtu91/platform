import {routerRedux} from "dva/router";
import {getCurPowers} from "../../utils";
import {getSid, searchChatInfo} from "../../services/chatSystem";
import {message} from "antd";

export default {
    namespace: "searchChatInfo",
    state: {
        list: [],
        sids: [],
    },
    subscriptions: {
        setup({dispatch, history}) {
            history.listen(location => {
                const pathName = location.pathname;
                if (pathName === "/chatSystem/searchChatInfo") {
                    const curPowers = getCurPowers(pathName);
                    if (curPowers) {
                        dispatch({ type: "getSid" });
                        dispatch({type: "app/changeCurPowers", payload: {curPowers}});
                    } else {
                        dispatch(routerRedux.push({pathname: "/no-power"}));
                    }
                }
            });
        }
    },

    effects: {
        * getSid({payload}, {call, put}) {
            const sid_data = yield call(getSid);
            if (sid_data.success === true) {
                yield put({
                    type: "querySuccess",
                    payload: {
                        sids: sid_data.list
                    }
                });
            }
        },

        * query({payload}, {call, put}) {
            const data = yield call(searchChatInfo, payload);
            if (data.success === true) {
                yield put({
                    type: "querySuccess",
                    payload: {
                        list: data.data,
                    }
                });
                message.success('成功！')
            } else {
                yield put({
                    type: "querySuccess",
                    payload: {
                        list: []
                    }
                });
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
