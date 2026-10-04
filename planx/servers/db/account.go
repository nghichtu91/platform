package db

import (
	"fmt"
	"strconv"
	"strings"

	uuid "github.com/satori/go.uuid"
)

type UserID struct{ uuid.UUID }

var InvalidUserID = UserID{UUID: uuid.Nil}

/*
	func (uid UserID) String() string {
		return uid.UUID.String()
	}

	func (uid UserID) MarshalText() (text []byte, err error) {
		return uid.UUID.MarshalText()
	}

	func (uid *UserID) UnmarshalText(text []byte) (err error) {
		return uid.UUID.UnmarshalText(text)
	}
*/
func (uid UserID) IsValid() bool {
	// return uint64(uid) > 0
	return uid.UUID != uuid.Nil
}

func NewUserID() UserID {
	return UserID{UUID: uuid.NewV4()}
}

func NewUserIDWithName(name string) UserID {
	return UserID{UUID: uuid.NewV5(uuid.NamespaceDNS, name)}
}

func UserIDFromStringOrNil(uid string) UserID {
	return UserID{UUID: uuid.FromStringOrNil(uid)}
}

type InvalidAcid string

// Error
func (i InvalidAcid) Error() string {
	return fmt.Sprintf("account format ilegal, parse %s failed", string(i))
}

type Account struct {
	GameId  uint
	ShardId uint
	UserId  UserID
}

func (a Account) String() string {
	// return fmt.Sprintf("%d:%d:%s", a.GameId, a.ShardId, a.UserId)
	return strconv.Itoa(int(a.GameId)) + ":" + strconv.Itoa(int(a.ShardId)) + ":" + a.UserId.String()
}

func (a Account) ServerString() string {
	return fmt.Sprintf("%d:%d", a.GameId, a.ShardId)
}

func ParseAccount(account string) (Account, error) {
	infos := strings.Split(account, ":")
	if len(infos) != 3 {
		return Account{}, InvalidAcid(account)
	}
	gid, _ := strconv.ParseUint(infos[0], 10, 0)
	sid, _ := strconv.ParseUint(infos[1], 10, 0)
	// uid, _ := strconv.ParseUint(infos[2], 10, 0)
	uid := uuid.FromStringOrNil(infos[2])
	return Account{uint(gid), uint(sid), UserID{UUID: uid}}, nil
}

// ParseAccountFast 解析账号, zero alloc
// 都正常的情况下, 快了一点...
// CPU: 121.4 ns/op -> 85.54 ns/op
// Mem:  96    B/op -> 48     B/op
func ParseAccountFast(account string) (Account, error) {
	src := account
	var _cache [3]string
	var info = _cache[:0]
	for idx := strings.Index(account, ":"); idx >= 0; idx = strings.Index(account, ":") {
		info = append(info, account[:idx])
		account = account[idx+1:]
	}
	info = append(info, account)
	if len(info) != 3 {
		return Account{}, InvalidAcid(src)
	}
	gid, _ := strconv.ParseUint(info[0], 10, 0)
	sid, _ := strconv.ParseUint(info[1], 10, 0)
	// uid, _ := strconv.ParseUint(info[2], 10, 0)
	uid := uuid.FromStringOrNil(info[2])
	return Account{
		GameId:  uint(gid),
		ShardId: uint(sid),
		UserId:  UserID{UUID: uid},
	}, nil
}

type ProfileDBKey struct {
	Account
	Prefix string
}

func (p ProfileDBKey) String() string {
	return fmt.Sprintf("%s:%s", p.Prefix, p.Account)
}

func ParseProfileDbKey(dbkey string) (ProfileDBKey, error) {
	pos := strings.Index(dbkey, ":")
	if pos < 0 {
		return ProfileDBKey{}, fmt.Errorf("ProfileDBKey format ilegal, Parse %s failed!", dbkey)
	}
	account, err := ParseAccount(dbkey[pos+1:])
	if err != nil {
		return ProfileDBKey{}, fmt.Errorf("ProfileDBKey format ilegal, Parse %s failed!", dbkey)
	}

	p := &ProfileDBKey{Prefix: dbkey[:pos]}
	p.Account = account
	return *p, nil
}
