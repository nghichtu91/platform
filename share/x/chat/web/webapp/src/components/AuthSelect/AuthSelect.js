import React, { PropTypes } from "react";
import { Form, Select, Button, Table, Input } from "antd";

const Option = Select.Option;
const FormItem = Form.Item;

class AuthSelect extends React.Component {
  static propTypes = {
    selects: PropTypes.array
  };

  constructor(props) {
    super(props);
    const value = props.list || { dataList: [] };
    this.state = {
      dataList: value.dataList
    };
    this.triggerChange({ dataList: value.dataList });
  }

  componentWillReceiveProps(nextProps) {
    // Should be a controlled component.
    if (nextProps.value != undefined) {
      const value = nextProps.value;
      this.setState(value);
    } else {
      this.triggerChange({ dataList: nextProps.list.dataList });
    }
  }

  triggerChange = changedValue => {
    // Should provide an event to pass value to Form.
    const onChange = this.props.onChange;
    if (onChange) {
      onChange(Object.assign({}, this.state, changedValue));
    }
  };

  render() {
    const {
      getFieldDecorator,
      getFieldsError,
      getFieldError,
      isFieldTouched
    } = this.props.form;
    const formItemLayout = {
      labelCol: {
        span: 6
      },
      wrapperCol: {
        span: 14
      }
    };
    const tailFormItemLayout = {
      wrapperCol: {
        xs: {
          span: 2,
          offset: 0
        },
        sm: {
          span: 3,
          offset: 6
        }
      }
    };

    const delAuthUrl = record => {
      var list = this.state.dataList;
      list.splice(list.findIndex(v => v.name == record.name), 1);
      this.setState({ dataList: list });
      this.triggerChange({ dataList: list });
    };

    const addAuthUrl = data => {
      var list = this.state.dataList;

      let authKey = this.props.form.getFieldValue("authUrlKey");
      let authValue;
      if (authKey && authKey != "") {
        authValue = this.props.form.getFieldValue("authUrlValue");
      } else {
        authKey = this.props.form.getFieldValue("authUrls");
        authValue = this.props.selects[authKey];
      }

      for (var k in list) {
        if (list[k].name === authKey) {
          list[k].url = authValue;
          this.setState({ dataList: list });
          this.triggerChange({ dataList: list });
          return;
        }
      }

      list.push({
        k: authKey,
        name: authKey,
        url: authValue
      });
      this.setState({ dataList: list });
      this.triggerChange({ dataList: list });
    };

    const authColumns = [
      {
        title: "名称",
        dataIndex: "name",
        key: "name"
      },
      {
        title: "登录地址",
        dataIndex: "url",
        key: "url"
      },
      {
        title: "操作",
        key: "operation",
        render: (text, record) => (
          <div>
            <Button type="danger" onClick={() => delAuthUrl(record)}>
              删除
            </Button>
          </div>
        )
      }
    ];

    let selectOptions = [];
    for (var k in this.props.selects) {
      selectOptions.push(<Option key={k}>{k}</Option>);
    }

    return (
      <div>
        <Form>
          <FormItem label="auth地址" {...formItemLayout}>
            {getFieldDecorator("authUrls", {})(
              <Select style={{ width: "100%" }} placeholder="请选择auth地址">
                {selectOptions}
              </Select>
            )}
          </FormItem>
          <FormItem label="自定义地址-键" {...formItemLayout}>
            {getFieldDecorator("authUrlKey", {})(
              <Input
                style={{ width: "100%" }}
                placeholder="请输入auth地址的键值"
              />
            )}
          </FormItem>
          <FormItem label="自定义地址-值" {...formItemLayout}>
            {getFieldDecorator("authUrlValue", {})(
              <Input
                style={{ width: "100%" }}
                placeholder="请输入auth地址的内容"
              />
            )}
          </FormItem>
          <FormItem {...tailFormItemLayout}>
            <Button type="primary" onClick={addAuthUrl}>
              添加
            </Button>
          </FormItem>
          <Table columns={authColumns} dataSource={this.state.dataList} />
        </Form>
      </div>
    );
  }
}

AuthSelect = Form.create({})(AuthSelect);

export default AuthSelect;
