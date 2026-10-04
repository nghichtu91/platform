import React from "react";
import {connect} from "dva";
import {Button, Form, Icon, message, Upload} from "antd";

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

class SensitiveWords extends React.Component {
    constructor(props) {
        super(props);
        this.props = props;
        this.state = {
            fileList: []
        }
    }

    handleChange = (info) => {
        if (info.file.name.indexOf('.csv') === -1) {
            message.error('请上传csv文件！')
            return
        }
        let fileList = [...info.fileList]
        fileList = fileList.slice(-1);
        fileList = fileList.map(file => {
            if (file.response) {
                file.url = file.response.url;
            }
            return file;
        });

        this.setState({fileList});
    }

    handleUpload = () => {
        let {form, dispatch} = this.props

        form.validateFields((errors, values) => {
            if (errors) {
                return;
            }
            const formData = new FormData();
            formData.append("sensitiveWordsFile", this.state.fileList[0].originFileObj);
            dispatch({
                type: "sensitiveWords/upload",
                payload: {data: formData, ContentType: "multipart/form-data"}
            });
            form.resetFields();
            this.setState({
                fileList: []
            })
        })
    }

    downloadFile = () => {
        let {dispatch} = this.props
        dispatch({
            type: "sensitiveWords/downloadFile",
        });
    }

    render() {
        const uploadProps = {
            multiple: true,
            beforeUpload: () => false,
            onChange: this.handleChange,
        }

        return (
            <div className="content-inner">
                <Form>
                    <FormItem {...tailFormItemLayout}>
                        <Upload {...uploadProps} fileList={this.state.fileList}>
                            <Button>
                                <Icon type="upload"/> 选择文件
                            </Button>
                        </Upload>
                    </FormItem>
                    <FormItem {...tailFormItemLayout}>
                        <Button style={{marginRight: '20'}} type="primary" onClick={this.handleUpload}>
                            上传文件
                        </Button>
                        <Button type="primary" onClick={this.downloadFile}>
                            导出文件
                        </Button>
                    </FormItem>
                </Form>
            </div>
        );
    }
}

function mapStateToProps({sensitiveWords, loading}) {
    return {sensitiveWords, loading: loading.models.uploadGiftCode};
}

export default connect(mapStateToProps)(Form.create()(SensitiveWords))


