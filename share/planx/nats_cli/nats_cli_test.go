package nats_cli

import (
	"bytes"
	"strconv"
	"strings"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
	"github.com/nghichtu91/platform/share/planx/etcd"
	"github.com/nghichtu91/platform/share/planx/nats_cli/pb"
	"github.com/nghichtu91/platform/share/planx/pbbuff"
	"github.com/nghichtu91/platform/share/planx/servers/planxprotogen"
)

func TestProtoEncode(t *testing.T) {
	var (
		w           *pb.NatsWrapper
		buff, buff2 *bytes.Buffer
		wBytes      []byte
		err         error
	)

	// old
	nw := new(pb.NatsWrapper)

	msg1 := &planxprotogen.BattleMsg{
		MsgData: []byte(strings.Repeat("strWithLength_16", 256)),
	}

	data1, err := proto.Marshal(msg1)
	assert.Nil(t, err)

	nw.Content = data1
	data3, err := proto.Marshal(nw)
	assert.Nil(t, err)

	// new

	w = GetNatsWrapper(w)
	buff = pbbuff.GetBuffer()
	buff2 = pbbuff.GetBuffer()
	defer func() {
		pbbuff.PutBuffer(buff2, &wBytes)
		pbbuff.PutBuffer(buff, &wBytes)
		PutNatsWrapper(w)
	}()

	data2, err := pmo.MarshalAppend(buff.Bytes(), msg1)
	assert.Nil(t, err)

	w.Content = data2

	data4, err := pmo.MarshalAppend(buff2.Bytes(), w)
	assert.Nil(t, err)

	assert.Equal(t, data1, data2)
	assert.Equal(t, data3, data4)
}

func TestSendMsg(t *testing.T) {
	InitNats("0", "nats://127.0.0.1:4222", false)

	msg0 := &planxprotogen.BattleMsg{
		MsgData: []byte("strWithLength_16"),
	}

	msg1 := &planxprotogen.BattleMsg{
		MsgData: []byte(strings.Repeat("strWithLength_16", 256)),
	}

	msg2 := &planxprotogen.BattleMsg{
		MsgData: []byte(strings.Repeat("strWithLength_16", 256*64)),
	}

	msg3 := &planxprotogen.BattleMsg{
		MsgData: []byte(strings.Repeat("strWithLength_16", 256*255)),
	}

	// span := opentracing.SpanFromContext(context.Background())
	scope := BattleFunc("12", BattleSubj_CreateRoom)
	msgs := []*planxprotogen.BattleMsg{msg0, msg1, msg2, msg3}

	for _, msg := range msgs {
		assert.Nil(t, SendMsg(scope, msg))
		assert.Nil(t, SendMsgWithCtx(nil, scope, msg))
	}
}

func BenchmarkSendMsg(b *testing.B) {
	InitNats("0", "nats://127.0.0.1:4222", false)

	msg0 := &planxprotogen.BattleMsg{
		MsgData: []byte("strWithLength_16"),
	}

	msg1 := &planxprotogen.BattleMsg{
		MsgData: []byte(strings.Repeat("strWithLength_16", 256)),
	}

	msg2 := &planxprotogen.BattleMsg{
		MsgData: []byte(strings.Repeat("strWithLength_16", 256*64)),
	}

	msg3 := &planxprotogen.BattleMsg{
		MsgData: []byte(strings.Repeat("strWithLength_16", 256*255)),
	}

	// span := opentracing.SpanFromContext(context.Background())
	scope := BattleFunc("12", BattleSubj_CreateRoom)

	b.Run("SendMsg", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			SendMsg(scope, msg0)
		}
	})

	b.Run("SendMsgCtx", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			SendMsgWithCtx(nil, scope, msg0)
		}
	})

	b.Run("SendMsg4", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			SendMsg(scope, msg1)
		}
	})

	b.Run("SendMsgCtx4", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			SendMsgWithCtx(nil, scope, msg1)
		}
	})

	b.Run("SendMsg256", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			SendMsg(scope, msg2)
		}
	})

	b.Run("SendMsgCtx256", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			SendMsgWithCtx(nil, scope, msg2)
		}
	})

	b.Run("SendMsg1024", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			SendMsg(scope, msg3)
		}
	})

	b.Run("SendMsgCtx1024", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			SendMsgWithCtx(nil, scope, msg3)
		}
	})
}

func BenchmarkNatsTopic(b *testing.B) {
	InitNats("0", "nats://127.0.0.1:4222", false)

	msg2 := &planxprotogen.BattleMsg{
		MsgData: []byte(strings.Repeat("strWithLength_16", 256*64)),
	}

	scope := []string{etcd.Server_Crossx, "12", "1"}
	var topic int

	b.Run("Topic 256", func(b *testing.B) {
		b.ReportAllocs()

		topic = 256

		for i := 0; i < b.N; i++ {
			SendMsg(append(scope, strconv.Itoa(i%topic)), msg2)
		}
	})

	b.Run("Topic 2048", func(b *testing.B) {
		b.ReportAllocs()

		topic = 2048

		for i := 0; i < b.N; i++ {
			SendMsg(append(scope, strconv.Itoa(i%topic)), msg2)
		}
	})

	b.Run("Topic 16384", func(b *testing.B) {
		b.ReportAllocs()

		topic = 16384

		for i := 0; i < b.N; i++ {
			SendMsg(append(scope, strconv.Itoa(i%topic)), msg2)
		}
	})

	b.Run("Topic 65536", func(b *testing.B) {
		b.ReportAllocs()

		topic = 65536

		for i := 0; i < b.N; i++ {
			SendMsg(append(scope, strconv.Itoa(i%topic)), msg2)
		}
	})
}

func BenchmarkSendMsgParallel(b *testing.B) {
	InitNats("0", "nats://127.0.0.1:4222", false)

	msg1 := &planxprotogen.BattleMsg{
		MsgData: []byte(strings.Repeat("strWithLength_16", 256)),
	}

	scope := BattleFunc("12", BattleSubj_CreateRoom)

	b.Run("SendMsg", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			SendMsg(scope, msg1)
		}
	})

	b.Run("SendMsg Parallel 8", func(b *testing.B) {
		b.ReportAllocs()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				SendMsg(scope, msg1)
			}
		})
	})

	b.Run("SendMsg Parallel 256", func(b *testing.B) {
		b.ReportAllocs()

		b.SetParallelism(256)

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				SendMsg(scope, msg1)
			}
		})
	})

	b.Run("SendMsg Parallel 2048", func(b *testing.B) {
		b.ReportAllocs()

		b.SetParallelism(4096)

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				SendMsg(scope, msg1)
			}
		})
	})
}
