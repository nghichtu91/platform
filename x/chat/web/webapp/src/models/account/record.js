import { routerRedux } from "dva/router";
import { getCurPowers } from "../../utils";
import { queryRecord } from "../../services/account/admin";

export default {
  namespace: "record",
  state: {
    List: []
  },
  subscriptions: {
    setup({ dispatch, history }) {
      history.listen(location => {
        const pathName = location.pathname;
        if (pathName === "/account/record") {
          const curPowers = getCurPowers(pathName);
          if (curPowers) {
            dispatch({ type: "app/changeCurPowers", payload: { curPowers } });
            dispatch({ type: "query" });
          } else {
            dispatch(routerRedux.push({ pathname: "/no-power" }));
          }
        }
      });
    }
  },
  effects: {
    *query({ payload }, { select, call, put }) {
      const data = yield call(queryRecord);
      if (data.success) {
        yield put({
          type: "querySuccess",
          payload: {
            List: data.list
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
