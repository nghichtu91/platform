import React, { PropTypes } from "react";
import { Select } from "antd";

const Option = Select.Option;
import axios from "axios";

class ShardSelect extends React.Component {
    constructor(props) {
        super(props);
        this.state = {
            selects: [],
            shard: ""
        };
    }

    componentDidMount() {
        axios.post("/api/common/getshards").then(response => {
            let res = response.data.data.List;
            if (this.props.mode === "multiple" || this.props.mode === "tags") {
                res.splice(0, 0, { shardId: 0, shardName: "全部" });
            }
            this.setState({ selects: res });
            this.props.onRef && this.props.onRef(this.state.selects);
        });
    }

    triggerChange = changedValue => {
        const onChange = this.props.onChange;
        if (onChange) {
            onChange(changedValue.shard);
        }
    };
    getGameShard = value => {
        this.setState({ shard: value });
        this.triggerChange({ shard: value });
        this.props.onChange(value)
    };

    render() {
        let selectOptions = [];
        let arr = this.state.selects;
        for (let i in arr) {
            if (!arr.hasOwnProperty(i)) continue;
            selectOptions.push(
                <Option key={i} value={arr[i].shardId}>{arr[i].shardName}</Option>
            );
        }

        return (
            <div>
                <Select
                    showSearch
                    style={{ width: "100%" }}
                    value={this.props.value}
                    placeholder="请选择服务器"
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

export default ShardSelect;
