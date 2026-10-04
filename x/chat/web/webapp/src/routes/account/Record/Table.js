import React from 'react';
import {Table, Input, Button, Icon} from 'antd';
import Highlighter from 'react-highlight-words';

class RecordTable extends React.Component {
    state = {
        searchText: '',
        searchedColumn: '',
    };

    getColumnSearchProps = dataIndex => ({
        filterDropdown: ({setSelectedKeys, selectedKeys, confirm, clearFilters}) => (
            <div style={{padding: 8}}>
                <Input
                    ref={node => {
                        this.searchInput = node;
                    }}
                    placeholder={`Search ${dataIndex}`}
                    value={selectedKeys[0]}
                    onChange={e => setSelectedKeys(e.target.value ? [e.target.value] : [])}
                    onPressEnter={() => this.handleSearch(selectedKeys, confirm, dataIndex)}
                    style={{width: 188, marginBottom: 8, display: 'block'}}
                />
                <Button
                    type="primary"
                    onClick={() => this.handleSearch(selectedKeys, confirm,dataIndex)}
                    icon="search"
                    size="small"
                    style={{width: 90, marginRight: 8}}
                >
                    搜索
                </Button>
                <Button onClick={() => this.handleReset(clearFilters)} size="small" style={{width: 90}}>
                    重置
                </Button>
            </div>
        ),
        filterIcon: filtered => (
            <Icon type="search" style={{color: filtered ? '#1890ff' : undefined}}/>
        ),
        onFilter: (value, record) =>
            record[dataIndex]
                .toString()
                .toLowerCase()
                .includes(value.toLowerCase()),
        onFilterDropdownVisibleChange: visible => {
            if (visible) {
                setTimeout(() => this.searchInput.select());
            }
        },
        render: text =>
            this.state.searchedColumn === dataIndex ? (
                <Highlighter
                    highlightStyle={{backgroundColor: '#ffc069', padding: 0}}
                    searchWords={[this.state.searchText]}
                    autoEscape
                    textToHighlight={text.toString()}
                />
            ) : (
                text
            ),
    });

    handleSearch = (selectedKeys, confirm, dataIndex) => {
        confirm();
        this.setState({
            searchText: selectedKeys[0],
            searchedColumn: dataIndex,
        });

    };

    handleReset = clearFilters => {
        clearFilters();
        this.setState({searchText: ''});
    };

    render() {
        const columns = [
            {
                title: "ID",
                dataIndex: "ID",
                key: "ID",
                sorter: (a, b) => a.ID - b.ID
            },
            {
                title: "用户",
                dataIndex: "user",
                key: "user",
                ...this.getColumnSearchProps('user'),
            },
            {
                title: "IP",
                dataIndex: "IP",
                key: "IP"
            },
            {
                title: "操作时间",
                dataIndex: "time",
                key: "time",
                sorter: (a, b) =>
                    Math.round(new Date(a.time).getTime() / 1000) -
                    Math.round(new Date(b.time).getTime() / 1000)
            },
            {
                title: "协议名称",
                dataIndex: "operation_name",
                key: "operation_name",
                ...this.getColumnSearchProps('operation_name'),
            },
            {
                title: "操作协议",
                dataIndex: "operation_type",
                key: "operation_type"
            },
            {
                title: "详细数据",
                dataIndex: "operation_detail",
                key: "operation_detail",
                ...this.getColumnSearchProps('operation_detail'),
            }
        ];
        return <Table columns={columns} dataSource={this.props.data}/>;
    }
}

export default RecordTable
