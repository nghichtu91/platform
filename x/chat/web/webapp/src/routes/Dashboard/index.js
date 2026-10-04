import React from "react";
import { connect } from "dva";

function Dashboard({ dashboard, dispatch }) {
  return <div className="content-inner">Welcome!</div>;
}

export default connect(({ dashboard }) => ({ dashboard }))(Dashboard);
