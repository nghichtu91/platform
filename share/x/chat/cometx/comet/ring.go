package comet

import (
	"errors"
	// log "github.com/nghichtu91/platform/share/planx/tilogs"
	"sync/atomic"

	pb "github.com/nghichtu91/platform/share/x/chat/api/protogen"
)

// Ring ring proto buffer.
type Ring struct {
	// read index
	rp uint64
	// ring length
	num uint64
	// 掩码 用于索引时超出范围后类似取余的作用
	mask uint64
	// TODO split cacheline, many cpu cache line size is 64
	// pad [40]byte
	// write index
	wp   uint64
	data []pb.ClientPackage
}

// NewRing new a ring buffer.
func NewRing(num int) *Ring {
	r := new(Ring)
	r.init(uint64(num))
	return r
}

// Init init ring.
func (r *Ring) Init(num int) {
	r.init(uint64(num))
}

func (r *Ring) init(num uint64) {
	// 2^N
	if num&(num-1) != 0 {
		for num&(num-1) != 0 {
			num &= (num - 1)
		}
		num = num << 1
	}
	r.data = make([]pb.ClientPackage, num)
	for i := 0; i < int(num); i++ {
		r.data[i].RawData = make([]byte, 0, pb.MaxBodySize)
	}
	r.num = num
	r.mask = r.num - 1
}

// Get get a proto from ring.
func (r *Ring) Get() (proto *pb.ClientPackage, err error) {
	if r.rp == atomic.LoadUint64(&r.wp) {
		return nil, errors.New(pb.ErrorCode_RingBufferEmpty.String())
	}
	proto = &r.data[r.rp&r.mask]
	return
}

// GetAdv incr read index.
func (r *Ring) GetAdv() {
	atomic.AddUint64(&r.rp, 1)
	// log.L().Debugf("ring rp: %d, idx: %d", r.rp, r.rp&r.mask)
}

// Set get a proto to write.
func (r *Ring) Set() (proto *pb.ClientPackage, err error) {
	if r.wp-atomic.LoadUint64(&r.rp) > r.num {
		return nil, errors.New(pb.ErrorCode_RingBufferFull.String())
	}
	proto = &r.data[r.wp&r.mask]
	return
}

// SetAdv incr write index.
func (r *Ring) SetAdv() {
	atomic.AddUint64(&r.wp, 1)
	// log.L().Debugf("ring wp: %d, idx: %d", r.wp, r.wp&r.mask)
}

// Reset reset ring.
func (r *Ring) Reset() {
	atomic.StoreUint64(&r.rp, 0)
	atomic.StoreUint64(&r.wp, 0)
	// prevent pad compiler optimization
	// r.pad = [40]byte{}
}
