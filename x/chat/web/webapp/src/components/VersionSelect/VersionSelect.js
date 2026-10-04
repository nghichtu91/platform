import React, { PropTypes } from "react";
import { Select } from "antd";

const Option = Select.Option;
import axios from "axios";

class VersionSelect extends React.Component {
  constructor(props) {
    super(props);
    this.state = {
      selects: [],
      version: ""
    };
  }

  componentDidMount() {
    // this.triggerChange({ version: this.props.versionValue });
    axios.post("/api/game_version/list").then(response => {
      let res = response.data.data.versions;
      this.setState({ selects: res });
    });
  }

  triggerChange = changedValue => {
    const onChange = this.props.onChange;
    if (onChange) {
      onChange(changedValue.version);
    }
  };
  getGameShard = value => {
    this.setState({ version: value });
    this.triggerChange({ version: value });
  };

  render() {
    let selectOptions = [];
    let arr = this.state.selects;
    for (let i in arr) {
      selectOptions.push(<Option key={arr[i]}>{arr[i]}</Option>);
    }

    var placeHolder;
    if (this.props.mode == "multiple") {
      placeHolder = "不选默认全部";
    } else if (this.props.mode == "tags") {
      placeHolder = "不选默认全部, 可自定义输入";
    } else {
      placeHolder = "请选择版本";
    }

    return (
      <div>
        <Select
          showSearch
          style={{ width: "100%" }}
          mode={this.props.mode}
          value={this.props.value}
          placeholder={placeHolder}
          onChange={this.getGameShard}
        >
          {selectOptions}
        </Select>
      </div>
    );
  }
}

export default VersionSelect;
