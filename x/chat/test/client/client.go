package client

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/nghichtu91/platform/share/planx/redispool"
	"github.com/nghichtu91/platform/share/planx/servers/chat"
	"github.com/nghichtu91/platform/share/planx/util"
	pb "github.com/nghichtu91/platform/share/x/chat/api/protogen"
	"github.com/nghichtu91/platform/share/x/chat/test/protocol"
	"github.com/nghichtu91/platform/share/x/chat/test/user_info"
)

var (
	handlerMap     map[string]protocol.ProtocolHandler
	respHandlerMap map[uint32]protocol.ProtocolHandler
)

var waitGroup util.WaitGroupWrapper

func NewChatRedis() redispool.IPool {
	return chat.NewRedisPool("chatredis",
		"127.0.0.1:6379", "", "", "chattest", nil, redispool.DefaultRedisPoolCapacity)
}

func StartRobot(robotIdStr string, id int) {
	var (
		n   int
		err error
		buf []byte
	)
	defer waitGroup.Done()

	netAddr := &net.TCPAddr{Port: id}
	dialer := net.Dialer{Timeout: time.Minute, LocalAddr: netAddr}
	c, err := dialer.Dial("tcp", user_info.GetCometAddr())
	if err != nil {
		// fmt.Println(err)
		return
	}

	robot := user_info.GetRobot(id)
	if robot == nil {
		fmt.Printf("start robot failed! robot:%d info is nil", id)
		return
	}

	// chatDb := user_info.GetChatDb()
	// 测试过期key 生效的
	// chat.ExpirePlayerMapping(chatDb, user_info.GetUserAcid(), 10)
	// defer chat.DelPlayerMapping(chatDb, robot.GetUserId())

	go OnAutoCommand(c, id)
	// go Heartbeat(c)
	t := time.NewTicker(10 * time.Second)
	defer t.Stop()

	for {
		select {
		case <-t.C:
			if handler, ok := getHandler("hb"); ok {
				buf = handler.FixedRequest("")
				n, err = c.Write(buf)
				if err != nil {
					fmt.Printf("heartbeat write package failed! n is %d, err:%v", n, err)
					return
				}
			}

			buf = nil
		}
	}
}

func HumanStart() {
	fileName := fmt.Sprintf("response_%s.txt", user_info.GetUserAcid())
	fmt.Printf("log exist at current dir, name is %s\n", fileName)
	f, err := os.Create(fileName)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	c, err := net.Dial("tcp", user_info.GetCometAddr())
	if err != nil {
		fmt.Println(err)
		return
	}

	/* 因为现在要是本地启动测试客户端的话 是访问不到QA的redis的 所以在logic上添加配置 允许测试机器人连接
	chatDb := user_info.GetChatDb()
	origin, encToken := chat.GenerateChatToken(user_info.GetUserAcid(), time.Now().Unix())
	user_info.SetUserToken(encToken)
	chat.SetPlayerMapping(chatDb, user_info.GetUserAcid(), origin)
	*/
	// 测试过期key 生效的
	// chat.ExpirePlayerMapping(chatDb, user_info.GetUserAcid(), 10)
	// defer chat.DelPlayerMapping(chatDb, user_info.GetUserAcid())

	user_info.SetUserToken("robot")

	go OnCommand(c)
	go Heartbeat(c)

	buf := make([]byte, 8)
	for {
		n, err := io.ReadFull(c, buf)
		if err != nil {
			fmt.Printf("read from conn failed, n is %d, buf is %v, err is %v", n, buf, err)
			break
		}

		//
		packLen := binary.LittleEndian.Uint32(buf[:4])
		messageId := binary.LittleEndian.Uint32(buf[4:8])
		bodyLen := packLen - 8
		bodyBuf := make([]byte, bodyLen)
		n, err = io.ReadFull(c, bodyBuf)
		if err != nil {
			fmt.Printf("read from conn failed, n is %d, buf is %v, err is %v", n, buf, err)
			break
		}

		f.WriteString(strconv.Itoa(int(packLen)) + "\n")
		f.WriteString(strconv.Itoa(int(messageId)) + "\n")

		handler, ok := getRespHandler(messageId)
		if ok {
			str := handler.ResponseString(bodyBuf[:n])
			f.WriteString(str + "\n")
		}
	}
}

func Do() {
	initHandler()
	chatDb := NewChatRedis()
	user_info.SetChatDb(chatDb)

	// 机器人聊天
	if user_info.IsMultiple() {
		startId := user_info.GetMultiStartId()
		count := user_info.GetMultiCount()
		for i := startId; i < startId+count; i++ {
			newId := GenerateId(i)
			/*
				origin, encToken := chat.GenerateChatToken(newId, time.Now().Unix())
				chat.SetPlayerMapping(chatDb, newId, origin)
			*/
			user_info.AddRobot(newId, "robot", i)
			waitGroup.Add(1)
			go func(idStr string, id int) {
				StartRobot(idStr, id)
			}(newId, i)
		}
		// 手动聊天
	} else {
		HumanStart()
	}

	waitGroup.Wait()
	// close todo
}

func OnAutoCommand(c net.Conn, id int) {
	var (
		n       int
		err     error
		buf     []byte
		handler protocol.ProtocolHandler
		ok      bool
	)

	// 先登陆
	handler, ok = getHandler("login")
	index := fmt.Sprintf("%d", id)
	buf = handler.FixedRequest(index)
	n, err = c.Write(buf)
	if err != nil {
		fmt.Printf("robot:%d send login package failed,err:%v", id, err)
		return
	}

	time.Sleep(time.Second * 3)

	// 加入房间
	handler, ok = getHandler("jr")
	// 向世界频道发送聊天
	// buf = handler.FixedRequest("3")
	buf = handler.FixedRequest("4")
	n, err = c.Write(buf)
	if err != nil {
		fmt.Printf("robot:%d send join room package failed,err:%v", id, err)
		return
	}

	t := time.NewTicker(5 * time.Second)
	defer t.Stop()

	for {
		select {
		case <-t.C:
			if handler, ok = getHandler("msg"); ok {
				// 随机聊天
				randomIndex := rand.Intn(10) + 11
				r := fmt.Sprintf("%d", randomIndex)
				buf = handler.FixedRequest(r)
				n, err = c.Write(buf)
				if err != nil {
					fmt.Printf("write package failed! err%v, n is %d", err, n)
					return
				}
			}

			buf = nil
		}
	}
}

func OnCommand(c net.Conn) {
	var (
		n       int
		err     error
		buf     []byte
		handler protocol.ProtocolHandler
		ok      bool
	)

	// 还是有个web的工具会更舒服一些吧
	for {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print(">> ")
		text, _ := reader.ReadString('\n')

		command := strings.TrimSpace(string(text))
		params := strings.Split(command, " ")
		if len(params) < 2 {
			fmt.Println("command error, size less 2")
			continue
		}

		mode := params[0]
		params = params[1:]
		if handler, ok = getHandler(params[0]); !ok {
			fmt.Println("command not exist! please type new command")
			continue
		}

		if mode == "f" {
			buf = handler.FixedRequest(params[1])
		} else {
			params = params[1:]
			buf = handler.ControlledRequest(params)
		}

		n, err = c.Write(buf)
		if err != nil {
			fmt.Printf("on command, write package failed! n is %d,err:%v", n, err)
			break
		}

		buf = nil
	}
}

func Heartbeat(c net.Conn) {
	var (
		n   int
		err error
		buf []byte
	)

	t := time.NewTicker(10 * time.Second)
	defer t.Stop()

	for {
		select {
		case <-t.C:
			if handler, ok := getHandler("hb"); ok {
				buf = handler.FixedRequest("")
				n, err = c.Write(buf)
				if err != nil {
					fmt.Printf("heartbeat write package failed! n is %d, err:%v", n, err)
					break
				}
			}

			buf = nil
		}
	}
}

func initHandler() {
	handlerMap = make(map[string]protocol.ProtocolHandler, 32)
	respHandlerMap = make(map[uint32]protocol.ProtocolHandler, 32)

	// create handler
	login := &protocol.LoginHandler{}
	reg("login", uint32(pb.ChatOperation_S2CLogin), login)

	chat := &protocol.ChatMsgHandler{}
	reg("msg", uint32(pb.ChatOperation_S2CChatMsg), chat)

	heartbeat := &protocol.HeartbeatHandler{}
	reg("hb", uint32(pb.ChatOperation_S2CHeartbeat), heartbeat)

	joinroom := &protocol.JoinRoomHandler{}
	reg("jr", uint32(pb.ChatOperation_S2CJoinRoom), joinroom)

	leaveroom := &protocol.LeaveRoomHandler{}
	reg("lr", uint32(pb.ChatOperation_S2CLeaveRoom), leaveroom)

	history := &protocol.HistoryHandler{}
	reg("hi", uint32(pb.ChatOperation_S2CHistroyMsg), history)
}

func reg(command string, respnseId uint32, handler protocol.ProtocolHandler) {
	handler.Init()
	handlerMap[command] = handler
	respHandlerMap[respnseId] = handler
}

func getHandler(command string) (v protocol.ProtocolHandler, ok bool) {
	v, ok = handlerMap[command]
	return
}

func getRespHandler(messageId uint32) (v protocol.ProtocolHandler, ok bool) {
	v, ok = respHandlerMap[messageId]
	return
}
