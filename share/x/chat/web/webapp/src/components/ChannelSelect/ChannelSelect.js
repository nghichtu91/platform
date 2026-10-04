import React, { PropTypes } from "react";
import { Select } from "antd";

const Option = Select.Option;
import axios from "axios";

class ChannelSelect extends React.Component {
  constructor(props) {
    super(props);
    this.state = {
      selects: [],
      channel: []
    };
  }

  componentDidMount() {
    axios.post("/api/channel/query_simple_channel").then(response => {
      let res = response.data.data.list;
      if (this.props.mode == "multiple" || this.props.mode == "tags" || this.props.needAll == true) {
        res.splice(0, 0, { channel_id: "-1", channel_name: "全部" });
      }
      this.setState({ selects: res });
    });
  }

  triggerChange = changedValue => {
    const onChange = this.props.onChange;
    if (onChange) {
      onChange(changedValue.channel);
    }
  };
  getGameShard = value => {
    this.setState({ channel: value });
    this.triggerChange({ channel: value });
  };

  render() {
    let selectOptions = [];
    let arr = this.state.selects;
    for (let i in arr) {
      selectOptions.push(
        <Option value={arr[i].channel_id}>{arr[i].channel_name}</Option>
      );
    }

    var placeHolder;
    if (this.props.placeholder) {
      placeHolder = this.props.placeholder;
    } else if (this.props.mode == "multiple") {
      placeHolder = "不选默认全部";
    } else if (this.props.mode == "tags") {
      placeHolder = "不选默认全部, 可自定义输入";
    } else {
      placeHolder = "请选择渠道";
    }

    return (
      <div>
        <Select
          showSearch
          value={this.props.value}
          style={{ width: "100%" }}
          mode={this.props.mode}
          placeholder={placeHolder}
          onChange={this.getGameShard}
        >
          {selectOptions}
        </Select>
      </div>
    );
  }
}

export default ChannelSelect;
