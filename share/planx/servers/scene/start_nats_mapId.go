package scene

// mapIdNeedStartNats 这个slice控制地图是否创建Nats，有Nats使用需求的直接往里加，不要在代码里修改
var mapIdNeedStartNats = []int32{
	371021, // 龙脉的等待地图
	602001, // 跨服场景答题地图
	602011, // 单服场景答题地图
	8,      // 主城长安
}
