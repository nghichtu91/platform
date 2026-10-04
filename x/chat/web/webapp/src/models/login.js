import { getGid } from "../services/app";

export default {
  namespace: "login",
  state: {
    gid: null
  },
  subscriptions: {
    setup({ dispatch, history }) {
      dispatch({ type: "getGid" });
    }
  },
  effects: {
    *getGid({ payload }, { call, put, select }) {
      const data = yield call(getGid);
      if (data.success) {
        yield put({
          type: "querySuccess",
          payload: {
            gid: data.gid
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
