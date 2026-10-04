import React, { PropTypes } from "react";
import { Select } from "antd";

const Option = Select.Option;
import axios from "axios";

class SimpleItemSelect extends React.Component {
  constructor(props) {
    super(props);
    this.state = {
      selects: [],
      list: []
    };
  }

  componentDidMount() {
    axios.get(`../asset/item2name.json`).then(res => {
      this.setState({ selects: res.data.ItemData });
    });
  }

  triggerChange = changedValue => {
    const onChange = this.props.onChange;
    if (onChange) {
      onChange(changedValue.list);
    }
  };
  getGameShard = value => {
    this.setState({ list: value });
    this.triggerChange({ list: value });
  };

  render() {
    let selectOptions = [];
    for (let k in this.state.selects) {
      selectOptions.push(
        <Option key={this.state.selects[k].ItemId}>
          {this.state.selects[k].ItemName}
        </Option>
      );
    }

    let placeHolder;
    if (this.props.mode == "multiple") {
      placeHolder = "不选默认全部";
    } else if (this.props.mode == "tags") {
      placeHolder = "不选默认全部, 可自定义输入";
    } else {
      placeHolder = "请选择物品";
    }

    return (
      <div>
        <Select
          showSearch
          style={{ width: "100%" }}
          value={this.props.value}
          placeholder={placeHolder}
          filterOption={(input, option) =>
            option.props.children.toLowerCase().indexOf(input.toLowerCase()) >=
            0
          }
          onChange={this.getGameShard}
          mode={this.props.mode}
        >
          {selectOptions}
        </Select>
      </div>
    );
  }
}

export default SimpleItemSelect;
