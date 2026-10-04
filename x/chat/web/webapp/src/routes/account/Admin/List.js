import React from "react";
import PropTypes from "prop-types";
import {Button, Menu, Modal, Table} from "antd";
import {DropMenu} from "../../../components/";
import {CANCEL, DELETE, STATUS, UPDATE} from "../../../constants/options";

const confirm = Modal.confirm;

function List({
                  accountAdmin: {list},
                  loading,
                  updatePower,
                  deletePower,
                  onDeleteItem,
                  onEditItem,
                  onStatusItem,
                  changePwd
              }) {
    const handleDeleteItem = record => {
        confirm({
            title: "您确定要删除这条记录吗?",
            onOk() {
                onDeleteItem(record.email);
            }
        });
    };

    const handleMenuClick = (key, record) => {
        return {
            [UPDATE]: onEditItem,
            [STATUS]: onStatusItem,
            [DELETE]: handleDeleteItem,
            [CANCEL]: changePwd
        }[key](record);
    };

    const columns = [
        {
            title: "邮箱",
            dataIndex: "email",
            key: "email"
        },
        {
            title: "权限分组",
            dataIndex: "roleName",
            key: "roleName"
        },
        {
            title: "创建时间",
            dataIndex: "createTime",
            key: "createTime"
        },
        {
            title: "状态",
            dataIndex: "status",
            key: "status",
            render: status => (
                <div>
                    {status === 1 ? (
                        <Button type="danger">已禁用</Button>
                    ) : (
                        <Button>已启用</Button>
                    )}
                </div>
            )
        },
        {
            title: "操作",
            key: "operation",
            // width: 100,
            render: (text, record) => (
                <DropMenu>
                    <Menu onClick={({key}) => handleMenuClick(key, record)}>
                        {updatePower && (
                            <Menu.Item key={STATUS}>
                                {record.status === 0 ? "禁用" : "启用"}
                            </Menu.Item>
                        )}
                        {updatePower && <Menu.Item key={UPDATE}>编辑</Menu.Item>}
                        {deletePower && <Menu.Item key={DELETE}>删除</Menu.Item>}
                        {updatePower && <Menu.Item key={CANCEL}>修改密码</Menu.Item>}
                    </Menu>
                </DropMenu>
            )
            // fixed: 'right'
        }
    ];

    return (
        <Table
            bordered
            columns={columns}
            dataSource={list}
            loading={loading}
            rowKey={record => record.email}
        />
    );
}

List.propTypes = {
    accountAdmin: PropTypes.object.isRequired,
    onDeleteItem: PropTypes.func.isRequired,
    onEditItem: PropTypes.func.isRequired
};

export default List;
