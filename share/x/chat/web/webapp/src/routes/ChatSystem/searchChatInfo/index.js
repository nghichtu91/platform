import React from "react";
import {connect} from "dva";
import {Button, Col, DatePicker, Form, Input, Row, Select, Table} from "antd";
import XLSX from 'xlsx'


const RangePicker = DatePicker.RangePicker;
const FormItem = Form.Item;
const Option = Select.Option
const formItemLayout = {
    labelCol: {
        xs: {span: 24},
        sm: {span: 8}
    },
    wrapperCol: {
        xs: {span: 24},
        sm: {span: 12}
    }
};
const channels = [
    {
        id: '-2',
        name: '无选择',
    }, {
        id: '-1',
        name: '私聊',
    }, {
        id: 'CurrentChannel',
        name: '当前频道',
    }, {
        id: 'LegionChannel',
        name: '军团频道',
    }, {
        id: 'SystemChannel',
        name: '系统频道',
    }, {
        id: 'TeamChannel',
        name: '队伍频道',
    }, {
        id: 'WorldChannel',
        name: '世界频道',
    }, {
        id: 'ZhaoMuChannel',
        name: '招募频道',
    }, {
        id: 'GuildPartyChannel',
        name: '军团聚会频道',
    }, {
        id: 'GuildWarChannel',
        name: '军团战频道',
    }, {
        id: 'CrossServerChannel',
        name: '跨服频道',
    }, {
        id: 'WarFieldChannel',
        name: '战区频道',
    },{
        id: 'ActivityChannel',
        name: '活动频道',
    },{
        id: 'CountryChannel',
        name: '国家频道',
    },{
        id: 'GVGFightChannel',
        name: '攻城频道',
    },
];
const tailFormItemLayout = {
    wrapperCol: {
        xs: {
            span: 24,
            offset: 0
        },
        sm: {
            span: 16,
            offset: 8
        }
    }
};
const columns = [
    {
        title: "大区ID",
        dataIndex: "gid",
        key: "gid"
    },
    {
        title: "服务器",
        dataIndex: "gameserver_id",
        key: "gameserver_id"
    },
    {
        title: "时间",
        dataIndex: "cn_time",
        key: "cn_time"
    },
    {
        title: "内容",
        dataIndex: "content",
        key: "content"
    },
    {
        title: "频道",
        dataIndex: "receiver",
        key: "receiver",
        render:(text, record)=>{
             return getChannelName(record)
        }
    },
    {
        title: "接收人",
        dataIndex: "isroom",
        key: "isroom",
        render:(text, record)=>{
            return record.receiver
        }
    },
    {
        title: "发送人",
        dataIndex: 'sender',
        key: "sender",
    },
    {
        title: "发送设备",
        dataIndex: 'sender_device',
        key: "sender_device",
    }
];

function getChannelName(record) {
    for (let item of channels){
        if (record.receiver.toLowerCase().includes(item.id.toLowerCase()) === true){
            return item.name
        }
    }
}


class SearchChatInfo extends React.Component {
    constructor(props) {
        super(props);
        this.props = props;
        this.state = {
            fileList: [],
            channel: '-2',
        }
    }

    beforeUpload = (file) => {
        this.setState({
            fileList: [file]
        })
        return false;
    }

    onSubmit = () => {
        const {form: {validateFields, resetFields}, dispatch} = this.props;
        validateFields((errors, values) => {
            if (errors) {
                return;
            }
            const reqData = {}
            Object.entries(values).forEach(([key, value]) => {
                if (value !== undefined) {
                    if (key === 'time') {
                        reqData.gte = value[0].toISOString()
                        reqData.lte = value[1].toISOString()
                        reqData.times = [reqData.gte, reqData.lte]
                    } else if (key === 'channel' && value !== '-1') {
                        // 非私聊
                        reqData['receiver'] = (value + "").toLowerCase()
                    } else {
                        reqData[key] = (value + "").toLowerCase()
                    }
                }
            })
            dispatch({
                type: "searchChatInfo/query",
                payload: reqData
            });
        });

        // resetFields()
    }

    onExport = () => {
        const {searchChatInfo: {list}} = this.props;
        let aoa = [
            ['时间', '发送者', '频道', '聊天内容'],
            ...list.map(item=>[item.cn_time, item.sender, getChannelName(item), item.content])
        ];
        let sheet = XLSX.utils.aoa_to_sheet(aoa);
        this.openDownloadDialog(this.sheet2blob(sheet), '聊天文件.xlsx');

        // resetFields()
    }

    openDownloadDialog=(url, saveName)=> {
        if(typeof url == 'object' && url instanceof Blob)
        {
            url = URL.createObjectURL(url); // 创建blob地址
        }
        let aLink = document.createElement('a');
        aLink.href = url;
        aLink.download = saveName || ''; // HTML5新增的属性，指定保存文件名，可以不要后缀，注意，file:///模式下不会生效
        let event;
        if(window.MouseEvent) event = new MouseEvent('click');
        else
        {
            event = document.createEvent('MouseEvents');
            event.initMouseEvent('click', true, false, window, 0, 0, 0, 0, 0, false, false, false, false, 0, null);
        }
        aLink.dispatchEvent(event);
    }

    sheet2blob=(sheet, sheetName) =>{
        sheetName = sheetName || 'sheet1';
        let workbook = {
            SheetNames: [sheetName],
            Sheets: {}
        };
        workbook.Sheets[sheetName] = sheet;
        // 生成excel的配置项
        let wopts = {
            bookType: 'xlsx', // 要生成的文件类型
            bookSST: false, // 是否生成Shared String Table，官方解释是，如果开启生成速度会下降，但在低版本IOS设备上有更好的兼容性
            type: 'binary'
        };
        let wbout = XLSX.write(workbook, wopts);
        // 字符串转ArrayBuffer
        function s2ab(s) {
            let buf = new ArrayBuffer(s.length);
            let view = new Uint8Array(buf);
            for (let i=0; i!==s.length; ++i) view[i] = s.charCodeAt(i) & 0xFF;
            return buf;
        }
        return new Blob([s2ab(wbout)], {type: "application/octet-stream"});
    }

    channelChange = (channel) => {
        this.setState({channel})
    }

    render() {
        // 不支持中文词 和 特殊字符
        let {form, searchChatInfo: {list, sids}} = this.props;
        const getFieldDecorator = form.getFieldDecorator;
        return (
            <div className="content-inner">
                <Form form={form}>
                    <Row align='middle'>
                        <Col span={6}>
                            <FormItem label="时间" {...formItemLayout}>
                                {getFieldDecorator("time")(<RangePicker showTime format="YYYY-MM-DD HH:mm:ss"/>)}
                            </FormItem>
                        </Col>
                        <Col span={6} offset={2}>
                            <FormItem {...formItemLayout} name="operator" label="服务器">
                                {getFieldDecorator("gameserver_id")(
                                    <Select>
                                        {sids.map(item => <Select.Option key={item.shardId}
                                                                         value={item.shardId}>{item.shardName}</Select.Option>)}
                                    </Select>)}
                            </FormItem>
                        </Col>
                        <Col span={6}>
                            <FormItem {...formItemLayout}
                                      name="sender"
                                      label="发送人"
                            >
                                {getFieldDecorator("sender")(<Input placeholder="不支持特殊字符"/>)}
                            </FormItem>
                        </Col>
                        <Col span={6}>
                            <FormItem {...formItemLayout}
                                      name="sender_device"
                                      label="发送人设备ID"
                            >
                                {getFieldDecorator("sender_device")(<Input placeholder="不支持特殊字符"/>)}
                            </FormItem>
                        </Col>
                        <Col span={6}>
                            <FormItem {...formItemLayout}
                                      name="content"
                                      label="关键字">
                                {getFieldDecorator("content")(<Input placeholder="不支持特殊字符"/>)}
                            </FormItem>
                        </Col>

                        <Col span={4}>
                            <FormItem {...formItemLayout}
                                      name="channel"
                                      label="频道选择"
                            > {getFieldDecorator("channel", {
                                initialValue: this.state.channel
                            })(
                                <Select onChange={this.channelChange}>
                                    {channels.map(item => <Option key={item.id} value={item.id}>{item.name}</Option>)}
                                </Select>
                            )}
                            </FormItem>
                        </Col>
                        {this.state.channel === '-1' ?
                            <Col span={4}>
                                <FormItem {...formItemLayout}
                                          name="receiver"
                                          label="接收人"
                                > {getFieldDecorator("receiver", {
                                    rules: [
                                        {
                                            required: true,
                                            message: "私聊必须选择接收人！"
                                        }
                                    ]
                                })(<Input placeholder="不支持特殊字符"/>)}
                                </FormItem>
                            </Col> : null
                        }
                        {/*<Col span={6}>*/}
                        {/*    <FormItem {...formItemLayout}*/}
                        {/*              name="dataStatus"*/}
                        {/*              label="数据状态"*/}
                        {/*    >*/}
                        {/*        {getFieldDecorator("data_status")(<Input disabled={true}/>)}*/}
                        {/*    </FormItem>*/}
                        {/*</Col>*/}
                        {/*<Col span={6}>*/}
                        {/*    <FormItem {...formItemLayout}*/}
                        {/*              name="interceptCause"*/}
                        {/*              label="拦截原因"*/}
                        {/*    >*/}
                        {/*        {getFieldDecorator("interceptCause")(<Input disabled={true}/>)}*/}
                        {/*    </FormItem>*/}
                        {/*</Col>*/}
                        <Col span={24} pull={6}>
                            <FormItem {...tailFormItemLayout}>
                                <Button onClick={this.onSubmit} type="primary" style={{marginRight: 20}}>
                                    查询
                                </Button>
                                <Button onClick={this.onExport} type="primary" disabled={!(list && list.length>0)}>
                                    导出
                                </Button>
                            </FormItem>
                        </Col>
                    </Row>
                </Form>
                <Table columns={columns} dataSource={list}/>
            </div>
        );
    }
}

function mapStateToProps({searchChatInfo, loading}) {
    return {searchChatInfo, loading: loading.models.searchChatInfo};
}

export default connect(mapStateToProps)(Form.create()(SearchChatInfo))


