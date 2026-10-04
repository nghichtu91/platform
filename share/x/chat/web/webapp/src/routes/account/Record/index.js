import React from "react";
import PropTypes from "prop-types";
import {connect} from "dva";
import RecordTable from "./Table";

function Record({record: {List}, loading}) {

    let source = [];
    if (Array.isArray(List) && List.length !== 0) {
        for (let i = 0; i < List.length; i++) {
            source.push({
                ID: List[i].id,
                user: List[i].user,
                IP: List[i].ip,
                time: new Date(List[i].time * 1000).format("yyyy-MM-dd HH:mm:ss"),
                operation_name: List[i].operation_name,
                operation_type: List[i].operation_type,
                operation_detail: List[i].operation_detail
            });
        }
    }

    return (
        <div className="content-inner">
            <RecordTable data={source}/>
        </div>
    );
}

Record.propTypes = {
    record: PropTypes.object,
    location: PropTypes.object,
    dispatch: PropTypes.func
};

function mapStateToProps({record, loading}) {
    return {record, loading: loading.models.record};
}

export default connect(mapStateToProps)(Record);
