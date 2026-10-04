package etcd_ser_discovery

// GamexExtra gamex服务发现extra字段，用于动态更新状态
type GamexExtra struct {
	IsMaintaining bool `json:"is_maintaining,omitempty"`
	PhysicsSid    uint `json:"physics_sid,omitempty"`
}

type GateExtra struct {
	Host string `json:"host,omitempty"`
}
