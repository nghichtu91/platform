import React from "react";
import {connect} from "dva";
import {Button, Form, Input} from "antd";

const FormItem = Form.Item;
const formItemLayout = {
    labelCol: {
        xs: {span: 24},
        sm: {span: 6}
    },
    wrapperCol: {
        xs: {span: 24},
        sm: {span: 6}
    }
};

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

class EditConfig extends React.Component {
    constructor(props) {
        super(props);
        this.props = props;
        this.state = {
            fileList: []
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
            dispatch({
                type: "editConfig/editChannelEtcd",
                payload: values
            });
        });

        // resetFields()
    }

    render() {
        // 不支持中文词 和 特殊字符
        let {form, editConfig: {curItem}} = this.props;
        const getFieldDecorator = form.getFieldDecorator;
        return (
            <div className="content-inner">
                <Form form={form}>
                    <FormItem {...formItemLayout} name="CurrentChannel" label="当前频道">
                        {getFieldDecorator("CurrentChannel", {
                            initialValue: curItem.CurrentChannel,
                            rules: [{required: true, message: "不能为空"}]
                        })(<Input/>)}
                    </FormItem>
                    <FormItem {...formItemLayout} name="GuildChannel" label="军团频道">
                        {getFieldDecorator("GuildChannel", {
                            initialValue: curItem.GuildChannel,
                            rules: [{required: true, message: "不能为空"}]
                        })(<Input/>)}
                    </FormItem>
                    <FormItem {...formItemLayout} name="SystemChannel" label="系统频道">
                        {getFieldDecorator("SystemChannel", {
                            initialValue: curItem.SystemChannel,
                            rules: [{required: true, message: "不能为空"}]
                        })(<Input/>)}
                    </FormItem>
                    <FormItem {...formItemLayout} name="TeamChannel" label="队伍频道">
                        {getFieldDecorator("TeamChannel", {
                            initialValue: curItem.TeamChannel,
                            rules: [{required: true, message: "不能为空"}]
                        })(<Input/>)}
                    </FormItem>
                    <FormItem {...formItemLayout} name="WorldChannel" label="世界频道">
                        {getFieldDecorator("WorldChannel", {
                            initialValue: curItem.WorldChannel,
                            rules: [{required: true, message: "不能为空"}]
                        })(<Input/>)}
                    </FormItem>
                    <FormItem {...formItemLayout} name="ZhaoMuChannel" label="招募频道">
                        {getFieldDecorator("ZhaoMuChannel", {
                            initialValue: curItem.ZhaoMuChannel,
                            rules: [{required: true, message: "不能为空"}]
                        })(<Input/>)}
                    </FormItem>
                    <FormItem {...formItemLayout} name="GuildWarChannel" label="军团战频道">
                        {getFieldDecorator("GuildWarChannel", {
                            initialValue: curItem.GuildWarChannel,
                            rules: [{required: true, message: "不能为空"}]
                        })(<Input/>)}
                    </FormItem>
                    <FormItem {...formItemLayout} name="GuildPartyChannel" label="军团聚会频道">
                        {getFieldDecorator("GuildPartyChannel", {
                            initialValue: curItem.GuildPartyChannel,
                            rules: [{required: true, message: "不能为空"}]
                        })(<Input/>)}
                    </FormItem>
                    <FormItem {...formItemLayout} name="CrossServerChannel" label="跨服频道">
                        {getFieldDecorator("CrossServerChannel", {
                            initialValue: curItem.CrossServerChannel,
                            rules: [{required: true, message: "不能为空"}]
                        })(<Input/>)}
                    </FormItem>
                    <FormItem {...formItemLayout} name="WarFieldChannel" label="战区频道">
                        {getFieldDecorator("WarFieldChannel", {
                            initialValue: curItem.WarFieldChannel,
                            rules: [{required: true, message: "不能为空"}]
                        })(<Input/>)}
                    </FormItem>
                    <FormItem {...formItemLayout} name="ActivityChannel" label="活动频道">
                        {getFieldDecorator("ActivityChannel", {
                            initialValue: curItem.ActivityChannel,
                            rules: [{required: true, message: "不能为空"}]
                        })(<Input/>)}
                    </FormItem>
					<FormItem {...formItemLayout} name="CountryChannel" label="国家频道">
                        {getFieldDecorator("CountryChannel", {
                            initialValue: curItem.CountryChannel,
                            rules: [{required: true, message: "不能为空"}]
                        })(<Input/>)}
                    </FormItem>
                    <FormItem {...formItemLayout} name="GVGFightChannel" label="攻城频道">
                        {getFieldDecorator("GVGFightChannel", {
                            initialValue: curItem.GVGFightChannel,
                            rules: [{required: true, message: "不能为空"}]
                        })(<Input/>)}
                    </FormItem>
                    <FormItem {...tailFormItemLayout}>
                        <Button onClick={this.onSubmit} type="primary">
                            修改
                        </Button>
                    </FormItem>
                </Form>
            </div>
        );
    }
}

function mapStateToProps({editConfig, loading}) {
    return {editConfig, loading: loading.models.editConfig};
}

export default connect(mapStateToProps)(Form.create()(EditConfig))


