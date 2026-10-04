import React from "react";
import PropTypes from "prop-types";
import { Form, Input, Modal, Icon, Select } from "antd";

const FormItem = Form.Item;

const Option = Select.Option;

const formItemLayout = {
  labelCol: {
    span: 6
  },
  wrapperCol: {
    span: 14
  }
};

const ModalForm = ({
  modal: { curItem, type, visible },
  loading,
  form: { getFieldDecorator, validateFields, resetFields },
  onOk,
  onCancel
}) => {
  if (!curItem.roleList) {
    curItem.roleList = [];
  }

  const handleOk = () => {
    validateFields((errors, values) => {
      if (errors) {
        return;
      }
      const data = {
        ...values,
        type
      };
      onOk(data);
    });
  };

  const modalFormOpts = {
    title:
      type === "create" ? (
        <div>
          <Icon type="plus-circle-o" /> 新建管理员
        </div>
      ) : (
        <div>
          <Icon type="edit" /> 修改管理员
        </div>
      ),
    visible,
    wrapClassName: "vertical-center-modal",
    confirmLoading: loading,
    onOk: handleOk,
    onCancel,
    afterClose() {
      resetFields(); //必须项，编辑后如未确认保存，关闭时必须重置数据
    },
    type: type
  };

  return (
    <Modal {...modalFormOpts}>
      {type !== "" && (
        <Form>
          <FormItem label="邮箱：" hasFeedback {...formItemLayout}>
            {getFieldDecorator("email", {
              initialValue: curItem.email,
              rules: [
                {
                  required: true,
                  message: "邮箱不能为空"
                },
                {
                  type: "email",
                  message: "邮箱格式不正确"
                }
              ]
            })(<Input type="email" disabled={!curItem.email ? false : true} />)}
          </FormItem>

          {type !== "set_pwd" && (
            <FormItem label="权限分组：" hasFeedback {...formItemLayout}>
              {getFieldDecorator("roleId", {
                initialValue: curItem.roleId && curItem.roleId.toString(),
                rules: [
                  {
                    required: true,
                    message: "角色不能为空"
                  }
                ]
              })(
                <Select placeholder="--请选择角色--">
                  {curItem.roleList.map(item => (
                    <Option key={item.id} value={item.id.toString()}>
                      {item.name}
                    </Option>
                  ))}
                </Select>
              )}
            </FormItem>
          )}

          {type === "set_pwd" && (
            <FormItem label="密码：" hasFeedback {...formItemLayout}>
              {getFieldDecorator("pwd", {
                initialValue: "",
                rules: [
                  {
                    required: true,
                    message: "密码不能为空"
                  }
                ]
              })(<Input type="password" />)}
            </FormItem>
          )}
        </Form>
      )}
    </Modal>
  );
};

ModalForm.propTypes = {
  modal: PropTypes.object.isRequired,
  form: PropTypes.object.isRequired
};

export default Form.create()(ModalForm);
