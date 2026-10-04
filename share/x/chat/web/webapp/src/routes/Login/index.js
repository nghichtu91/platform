import React from "react";
import PropTypes from "prop-types";
import { connect } from "dva";
import LoginForm from "./LoginForm";
import styles from "./LoginForm.less";
import { Spin } from "antd";

function Login({ login, dispatch, loading = false }) {
  const loginProps = {
    login,
    loading,
    onOk(data) {
      dispatch({ type: "app/login", payload: data });
    }
  };
  return (
    <div className={styles.spin}>
      <Spin tip="加载用户信息..." spinning={loading} size="large">
        <LoginForm {...loginProps} />
      </Spin>
    </div>
  );
}

Login.propTypes = {
  dispatch: PropTypes.func,
  loading: PropTypes.bool
};

function mapStateToProps({ login, loading }) {
  return { login, loading: loading.models.app };
}

export default connect(mapStateToProps)(Login);
