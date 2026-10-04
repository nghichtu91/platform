import React from "react";
import { routerRedux } from "dva/router";
import { connect } from "dva";
import AdminList from "./List";
import AdminSearch from "./Search";
import AdminModal from "./ModalForm";
import { checkPower } from "../../../utils";
import { ADD, UPDATE, DELETE } from "../../../constants/options";
import PropTypes from "prop-types";

function Admin({
  location,
  curPowers,
  dispatch,
  accountAdmin,
  modal,
  loading
}) {
  const addPower = checkPower(ADD, curPowers);
  const updatePower = checkPower(UPDATE, curPowers);
  const deletePower = checkPower(DELETE, curPowers);

  const { field, keyword } = location.query;

  const searchProps = {
    field,
    keyword,
    addPower,
    onSearch(fieldsValue) {
      const { pathname } = location;
      !!fieldsValue.keyword.length
        ? dispatch(
            routerRedux.push({
              pathname: pathname,
              query: {
                ...fieldsValue
              }
            })
          )
        : dispatch(routerRedux.push({ pathname: pathname }));
    },
    onAdd() {
      dispatch({
        type: "accountAdmin/showModal",
        payload: {
          type: "create"
        }
      });
    }
  };

  const listProps = {
    accountAdmin,
    loading,
    updatePower,
    deletePower,
    location,
    onDeleteItem(email) {
      dispatch({ type: "accountAdmin/delete", payload: { email } });
    },
    onEditItem(item) {
      dispatch({
        type: "accountAdmin/showModal",
        payload: {
          type: "update",
          curItem: item
        }
      });
    },
    onStatusItem(item) {
      item.update_type = 2;
      dispatch({
        type: "accountAdmin/updateStatus",
        payload: {
          curItem: item
        }
      });
    },
    changePwd(item) {
      item.update_type = 3;
      dispatch({
        type: "accountAdmin/showModal",
        payload: {
          type: "set_pwd",
          curItem: item
        }
      });
    }
  };

  const modalProps = {
    modal,
    loading,
    onOk(data) {
      if (data.type === "update") {
        dispatch({
          type: "accountAdmin/update",
          payload: {
            curItem: { ...data, update_type: 1 }
          }
        });
      } else if (data.type === "create") {
        dispatch({
          type: "accountAdmin/create",
          payload: {
            curItem: { ...data, update_type: 0 }
          }
        });
      } else if (data.type === "set_pwd") {
        dispatch({
          type: "accountAdmin/update",
          payload: {
            curItem: { ...data, update_type: 3 }
          }
        });
      }
    },
    onCancel() {
      dispatch({ type: "modal/hideModal" });
    }
  };

  return (
    <div className="content-inner">
      <AdminSearch {...searchProps} />
      <AdminList {...listProps} />
      <AdminModal {...modalProps} />
    </div>
  );
}

Admin.propTypes = {
  accountAdmin: PropTypes.object,
  location: PropTypes.object,
  dispatch: PropTypes.func
};

function mapStateToProps({ accountAdmin, modal, loading }) {
  return { accountAdmin, modal, loading: loading.models.accountAdmin };
}

export default connect(mapStateToProps)(Admin);
