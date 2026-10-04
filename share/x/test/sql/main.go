package main

import (
	"fmt"

	_ "github.com/go-sql-driver/mysql"
)

const (
	ProjectID = uint64(10)
	BatchID   = uint64(1)
	GroupID   = uint64(1)
	Count     = uint64(10)
)

func main() {
	a := 1
	fmt.Println(fmt.Sprintf("'%v%%'", a))

	//var sum int
	//var code = "Hello123"
	//for index := 0; index < len(code); index++ {
	//	sum += int(code[index])
	//}
	//
	//fmt.Println(sum)

	// 与数据库建立链接。
	//conn, err := sql.Open("mysql", "root:123456@tcp(127.0.0.1:3306)/new_server?charset=utf8")
	//defer conn.Close()
	//if err != nil {
	//	fmt.Println(err)
	//	return
	//}
	//
	//result, err := conn.Query("show tables like 'new_table%'")
	//for result.Next() {
	//	var content string
	//	result.Scan(&content)
	//	fmt.Println(content)
	//}

	//if err, errMsg := db.InsertGenCodeList(codes, config, true); err != nil {
	//	fmt.Println(err, errMsg)
	//}
	//
	//fmt.Println("成功啦！")

	//fmt.Println("建立连接成功！")
	//
	//// 生成巨量的兑换码。
	//codes, nil, _ := logic.Gen_Gift_Code(ProjectID, BatchID, GroupID, Count)

	//if errConf := config.LoadGiftConfig("config.toml"); errConf != nil {
	//	tilogs.L().Errorf("Gift Server %v", errConf)
	//	return
	//}
	//
	//db.InitMySqlDB()
	//
	//// 生成兑换码。
	//codes, nil, _ := logic.Gen_Gift_Code(ProjectID, BatchID, GroupID, Count)
	//
	//codes = []string{"asds"}
	//
	//// 自定义配置。
	//config := &define.GiftCodeInfo{
	//	BatchID:    int(BatchID),
	//	GroupID:    int(GroupID),
	//	Gid:        []string{"304", "305", "307", "308", "309"},
	//	ChannelIds: []int{0, 1, 2, 3, 4, 5},
	//	GiftName:   "测试兑换码",
	//	StartTime:  1602661830,
	//	EndTime:    1605661830,
	//	GiftType:   1,
	//	GenCount:   int(Count),
	//	TotalCount: 0,
	//	GenTime:    1602661830,
	//	UseCount:   1,
	//	CustomCode: "",
	//}
	//
	//if err, errMsg := db.InsertGenCodeList(codes, config, true); err != nil {
	//	fmt.Println(err, errMsg)
	//}
	//
	//fmt.Println("成功啦！")

	//// 开启事务。
	//ts, err := conn.Begin()
	//if err != nil {
	//	fmt.Println(fmt.Sprintf("开启事务失败: %v", err))
	//}
	//
	//// 判断数据库与建表。
	//_, errConn := ts.Exec("create table if not exists `GiftCode` ( `giftcode` VARCHAR(20) NOT NULL, `users` MEDIUMTEXT NULL, PRIMARY KEY (`giftcode`));")
	//if errConn != nil {
	//	fmt.Println("Err", errConn.Error())
	//}
	//
	//for {
	//	if len(codes) == 0 {
	//		break
	//	}
	//
	//	tempCodes := make([]string, 0, 5)
	//	if len(codes) >= 5 {
	//		tempCodes = codes[:5]
	//		codes = codes[5:]
	//	} else {
	//		tempCodes = codes
	//		codes = make([]string, 0, 5)
	//	}
	//
	//	batchHeader := fmt.Sprintf("insert into %v(%v) values", "GiftCode", "giftcode")
	//
	//	buf := make([]byte, 0)
	//	buf = append(buf, batchHeader...)
	//
	//	for _, code := range tempCodes {
	//		buf = append(buf, " ( '"+code+"' ),"...)
	//	}
	//
	//	buf = buf[:len(buf)-1]
	//	buf = append(buf, ";"...)
	//
	//	_, errConn := ts.Exec(string(buf))
	//	if errConn == nil {
	//		fmt.Println(fmt.Sprintf("插入成功"))
	//	} else {
	//		fmt.Println(errConn)
	//		ts.Rollback()
	//		return
	//	}
	//
	//}
	////ts.Commit()
	////return
	//
	///*
	//	查询示例
	//*/
	//rows, err := ts.Query("select * from GiftCode")
	//if err != nil {
	//	fmt.Println(err)
	//	return
	//}
	//
	//for rows.Next() {
	//	var giftcode string
	//	var users string
	//	err := rows.Scan(&giftcode, &users)
	//	if err != nil {
	//		fmt.Println(err)
	//		return
	//	}
	//}
}
