export default {
  namespace: "dashboard",
  state: {},
  reducers: {
    saveState(state, action) {
      return {
        ...state,
        ...action.payload
      };
    }
  },

  effects: {},
  subscriptions: {
    setup({ history, dispatch }) {
      // 监听 history 变化，当进入 `/` 时触发 `load` action
      return history.listen(({ pathname }) => {
        if (pathname === "/") {
        }
      });
    }
  }
};
