export default {
  namespace: "relModal",
  state: {
    relModalVisible: false,
    boardModalVisible: false,
    curItem: {}
  },
  effects: {
    *changeStateAction({ payload }, { put }) {
      yield put({ type: "changeState", payload });
    }
  },
  reducers: {
    relShowModal(state, action) {
      return { ...state, relModalVisible: true, ...action.payload };
    },
    hideModal(state) {
      return { ...state, relModalVisible: false, curItem: {} };
    },
    changeState(state, action) {
      return { ...state, ...action.payload };
    },
    setItem(state, action) {
      const { curItem } = action.payload;
      return { ...state, curItem };
    }
  }
};
