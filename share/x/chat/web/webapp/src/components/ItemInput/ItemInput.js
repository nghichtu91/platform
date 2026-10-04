import React, { PropTypes } from "react";
import {Form, Select, Button, Table, Icon, Tag, InputNumber} from "antd";

const Option = Select.Option;
const FormItem = Form.Item;
import axios from "axios";

class ItemSelect extends React.Component {
    static propTypes = {
        dataList: PropTypes.array,
        funcId: PropTypes.string
    };

    constructor(props) {
        super(props);

        const value = { dataList: [] };
        this.state = {
            selects: [],
            dataList: value.dataList,
            limit: {},
            placeholder: "",
            allLimit: {},
            initDataList: props.value == undefined ? [] : props.value.initDataList,
            firstCreate: true
        };
        this.triggerChange({ dataList: value.dataList });
    }

    componentDidMount() {
        axios.get(`../asset/item2name.json`).then(res => {
            this.setState({ selects: res.data.ItemData });

            var selects = res.data.ItemData;
            axios.post("/api/itemlimit/query").then(res => {
                let list = res.data.data.list[this.props.funcId];
                var limit = {};
                for (var i in list) {
                    limit[list[i].itemId] = {
                        warnCount: list[i].warnCount,
                        maxCount: list[i].maxCount
                    };
                }
                this.setState({ allLimit: res.data.data.list, limit: limit });

                var initDataList = this.state.initDataList;
                if (initDataList) {
                    var list1 = [];
                    for (var k in initDataList) {
                        var matchIndex = selects.findIndex(
                            v => v.ItemId == initDataList[k].Id
                        );
                        list1.push({
                            k: initDataList[k].Id,
                            id: initDataList[k].Id,
                            name: selects[matchIndex].ItemName,
                            count: initDataList[k].Count,
                            state: 0
                        });
                    }
                    this.setState({ dataList: list1 });
                    this.triggerChange({ dataList: list1 });
                }
                this.setState({ firstCreate: false, initDataList: undefined });
            });
        });
    }

    componentWillReceiveProps(nextProps) {
        // Should be a controlled component.
        if (nextProps.value != undefined) {
            const value = nextProps.value;
            this.setState({ dataList: value.dataList });

            if (!this.state.firstCreate && value.initDataList) {
                var list1 = [];
                for (var k in value.initDataList) {
                    var matchIndex = this.state.selects.findIndex(
                        v => v.ItemId == value.initDataList[k].Id
                    );
                    list1.push({
                        k: value.initDataList[k].Id,
                        id: value.initDataList[k].Id,
                        name: this.state.selects[matchIndex].ItemName,
                        count: value.initDataList[k].Count,
                        state: 0
                    });
                }
                this.setState({ dataList: list1 });
                this.triggerChange({ dataList: list1 });
            }
        }

        // if (nextProps.reset) {
        //   this.setState({ dataList: [] });
        //   this.props.form.resetFields();
        // }

        if (nextProps.funcId) {
            var limit = {};
            var allLimit = this.state.allLimit;
            var list = allLimit[nextProps.funcId];
            for (var i in list) {
                limit[list[i].itemId] = {
                    warnCount: list[i].warnCount,
                    maxCount: list[i].maxCount
                };
            }
            this.setState({ limit: limit });
        }
    }

    triggerChange = changedValue => {
        // Should provide an event to pass value to Form.
        const onChange = this.props.onChange;
        if (onChange) {
            onChange(Object.assign({}, this.state, changedValue));
        }
    };

    onSelectChange = itemId => {
        var itemLimit = this.state.limit;
        if (itemLimit[itemId]) {
            this.setState({
                placeholder:
                    "警戒:" +
                    itemLimit[itemId].warnCount +
                    ",最大:" +
                    itemLimit[itemId].maxCount
            });
        } else {
            this.setState({ placeholder: "" });
        }
    };

    render() {
        const {
            getFieldDecorator,
            validateFields,
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
                    offset: 0
                }
            }
        };

        const delItemUrl = record => {
            var list = this.state.dataList;
            list.splice(list.findIndex(v => v.name == record.name), 1);
            this.setState({ dataList: list });
            this.triggerChange({ dataList: list });
        };

        const addItemUrl = data => {
            validateFields((errors, values) => {
                if (errors) {
                    return;
                }

                const formData = {
                    ...values
                };

                const itemId = formData.ids;
                const itemCount = formData.count;

                var itemLimit = this.state.limit;
                var state = 0;
                if (
                    itemLimit[itemId] &&
                    itemLimit[itemId].warnCount != 0 &&
                    itemCount > itemLimit[itemId].warnCount
                ) {
                    state = 1;
                }

                var list = this.state.dataList;
                if (!list) {
                    list = [];
                }

                for (var k in list) {
                    if (list[k].id === itemId) {
                        return; // 避免重复添加
                    }
                }
                if (itemCount == undefined || itemCount <= 0) {
                    return;
                }
                var matchIndex = this.state.selects.findIndex(v => v.ItemId == itemId);
                list.push({
                    k: itemId,
                    id: itemId,
                    name: this.state.selects[matchIndex].ItemName,
                    count: itemCount,
                    state: state
                });
                this.setState({ dataList: list });
                this.triggerChange({ dataList: list });
            });
        };

        const itemColumns = [
            {
                title: "物品ID",
                dataIndex: "id",
                key: "id"
            },
            {
                title: "状态",
                dataIndex: "state",
                key: "state",
                render: val => (
                    <div>
                        <Tag color={val == 1 ? "red" : "green"}>
                            {val == 1 ? "超警戒线" : "正常"}
                        </Tag>
                    </div>
                )
            },
            {
                title: "物品名称",
                dataIndex: "name",
                key: "name"
            },
            {
                title: "物品数量",
                dataIndex: "count",
                key: "count"
            },
            {
                title: "操作",
                key: "operation",
                render: (text, record) => (
                    <div>
                        <Button type="danger" onClick={() => delItemUrl(record)}>
                            <Icon type={"minus"} />
                        </Button>
                    </div>
                )
            }
        ];

        var selectOptions = [];
        for (var k in this.state.selects) {
            selectOptions.push(
                <Option key={this.state.selects[k].ItemId}>
                    {this.state.selects[k].ItemName}
                </Option>
            );
        }

        const handleItemLimit = (rule, value, callback) => {
            const { getFieldValue } = this.props.form;
            var itemId = getFieldValue("ids");
            if (!itemId) {
                callback("需要选择物品");
            } else {
                var itemLimit = this.state.limit[itemId];
                if (itemLimit) {
                    if (itemLimit.maxCount != 0 && value > itemLimit.maxCount) {
                        callback("超出最大数量");
                    }
                }
            }

            // Note: 必须总是返回一个 callback，否则 validateFieldsAndScroll 无法响应
            callback();
        };

        return (
            <div>
                <Form>
                    <FormItem label="物品ID" {...formItemLayout}>
                        {getFieldDecorator("ids", {})(
                            <Select
                                showSearch
                                style={{ width: "100%" }}
                                placeholder="请选择物品"
                                filterOption={(input, option) =>
                                    option.props.children
                                        .toLowerCase()
                                        .indexOf(input.toLowerCase()) >= 0
                                }
                                onChange={this.onSelectChange}
                            >
                                {selectOptions}
                            </Select>
                        )}
                    </FormItem>
                    <FormItem label={"数量"} {...formItemLayout}>
                        {getFieldDecorator("count", {
                            rules: [
                                {
                                    validator: handleItemLimit
                                }
                            ]
                        })(
                            <InputNumber
                                type={"number"}
                                min={1}
                                max={99999}
                                placeholder={this.state.placeholder}
                            />
                        )}
                    </FormItem>
                    <FormItem {...formItemLayout}>
                        <Button onClick={addItemUrl}>
                            <Icon type={"plus"} />添加
                        </Button>
                    </FormItem>
                    <Table columns={itemColumns} dataSource={this.state.dataList} />
                </Form>
            </div>
        );
    }
}

ItemSelect = Form.create({})(ItemSelect);

export default ItemSelect;
