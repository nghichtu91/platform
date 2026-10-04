import {routerRedux} from "dva/router";
import {getCurPowers} from "../../utils";
import {downloadSensitiveWordsFile, uploadSensitiveWordsFile} from "../../services/chatSystem";
import {message} from "antd";

export default {
    namespace: "sensitiveWords",
    state: {
        showStatus: false,
        online: true,
    },
    subscriptions: {
        setup({dispatch, history}) {
            history.listen(location => {
                const pathName = location.pathname;
                if (pathName === "/chatSystem/sensitiveWords") {
                    const curPowers = getCurPowers(pathName);
                    if (curPowers) {
                        dispatch({type: "app/changeCurPowers", payload: {curPowers}});
                    } else {
                        dispatch(routerRedux.push({pathname: "/no-power"}));
                    }
                }
            });
        }
    },

    effects: {
        * upload({payload}, {call}) {
            const data = yield call(uploadSensitiveWordsFile, payload);
            if (data.success === true) {
                message.success('热更成功！')
            }
        },
        * downloadFile(_, {call}) {
            yield call(downloadSensitiveWordsFile);
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
