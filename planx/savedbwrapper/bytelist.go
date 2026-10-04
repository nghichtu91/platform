package savedbwrapper

type IByteList interface {
	Len() int
	Get(index int) (byte, bool)
	Set(index int, v byte) bool
	Remove(index int) bool
	Append(v byte)
	Appends(v []byte)
	Insert(index int, v byte) bool
	Clear()
	GetAllReadOnly() []byte // 获取数组内容的拷贝，注意不要修改返回的数组
	setSubChange(chgr ISubChanger)
}

type SetByteList func([]byte)

func NewbyteListOld(setList SetByteList, l []byte, chgr ISubChanger) IByteList {
	_r := &ByteList{
		setList: setList,
		data:    l,
		chgr:    chgr,
	}
	_r.copy()
	return _r
}

type ByteList struct {
	setList      SetByteList
	data         []byte
	dataReadOnly []byte
	chgr         ISubChanger
}

func (l *ByteList) Len() int {
	return len(l.data)
}

func (l *ByteList) Get(index int) (byte, bool) {
	if index >= len(l.data) {
		return 0, false
	}
	return l.data[index], true
}

func (l *ByteList) Set(index int, v byte) bool {
	if index >= len(l.data) {
		return false
	}
	l.data[index] = v
	l.setList(l.data)
	if l.chgr != nil {
		l.chgr.SetSubChange()
	}
	l.copy()
	return true
}

func (l *ByteList) Remove(index int) bool {
	if index >= len(l.data) {
		return false
	}
	l.data = append(l.data[:index], l.data[index+1:]...)
	l.setList(l.data)
	if l.chgr != nil {
		l.chgr.SetSubChange()
	}
	l.copy()
	return true
}

func (l *ByteList) Append(v byte) {
	l.data = append(l.data, v)
	l.setList(l.data)
	if l.chgr != nil {
		l.chgr.SetSubChange()
	}
	l.copy()
}

func (l *ByteList) Appends(v []byte) {
	l.data = append(l.data, v...)
	l.setList(l.data)
	if l.chgr != nil {
		l.chgr.SetSubChange()
	}
	l.copy()
}

func (l *ByteList) Insert(index int, v byte) bool {
	if index > len(l.data) {
		return false
	}
	rear := append([]byte{}, l.data[index:]...)
	l.data = append(append(l.data[:index], v), rear...)
	l.setList(l.data)
	if l.chgr != nil {
		l.chgr.SetSubChange()
	}
	l.copy()
	return true
}

func (l *ByteList) Clear() {
	l.data = []byte{}
	l.setList(l.data)
	if l.chgr != nil {
		l.chgr.SetSubChange()
	}
	l.copy()
}

func (l *ByteList) GetAllReadOnly() []byte {
	return l.dataReadOnly
}

func (l *ByteList) copy() {
	ret := make([]byte, len(l.data))
	copy(ret, l.data)
	l.dataReadOnly = ret
}

func (l *ByteList) setSubChange(chgr ISubChanger) {
	l.chgr = chgr
}
