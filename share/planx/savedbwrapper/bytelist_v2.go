package savedbwrapper

func NewbyteList(setList SetByteList, l []byte, chgr ISubChanger) IByteList {
	// 如果初始化slice为空，给slice预分配默认长度
	if len(l) == 0 {
		l = make([]byte, 0, defaultSliceLen)
	}
	_r := &byteListV2{
		setList:      setList,
		data:         l,
		dataReadOnly: make([]byte, 0, len(l)),
		chgr:         chgr,
		changed:      true,
	}
	_r.copy()
	return _r
}

type byteListV2 struct {
	setList      SetByteList
	data         []byte
	dataReadOnly []byte
	chgr         ISubChanger
	changed      bool
}

func (l *byteListV2) Len() int {
	return len(l.data)
}

func (l *byteListV2) Get(index int) (byte, bool) {
	if index >= len(l.data) {
		return 0, false
	}
	return l.data[index], true
}

func (l *byteListV2) Set(index int, v byte) bool {
	if index >= len(l.data) {
		return false
	}
	l.data[index] = v
	l.setList(l.data)
	if l.chgr != nil {
		l.chgr.SetSubChange()
	}
	// l.copy()
	l.changed = true
	return true
}

func (l *byteListV2) Remove(index int) bool {
	if index >= len(l.data) {
		return false
	}
	l.data = append(l.data[:index], l.data[index+1:]...)
	l.setList(l.data)
	if l.chgr != nil {
		l.chgr.SetSubChange()
	}
	// l.copy()
	l.changed = true
	return true
}

func (l *byteListV2) Append(v byte) {
	l.data = append(l.data, v)
	l.setList(l.data)
	if l.chgr != nil {
		l.chgr.SetSubChange()
	}
	// l.copy()
	l.changed = true
}

func (l *byteListV2) Appends(v []byte) {
	l.data = append(l.data, v...)
	l.setList(l.data)
	if l.chgr != nil {
		l.chgr.SetSubChange()
	}
	// l.copy()
	l.changed = true
}

func (l *byteListV2) Insert(index int, v byte) bool {
	if index > len(l.data) {
		return false
	}
	// rear := append([]byte{}, l.data[index:]...)
	// l.data = append(append(l.data[:index], v), rear...)
	if len(l.data) == 0 && index == 0 {
		l.data = append(l.data, v)
	} else {
		l.data = append(l.data[:index+1], l.data[index:]...)
		l.data[index] = v
	}

	l.setList(l.data)
	if l.chgr != nil {
		l.chgr.SetSubChange()
	}
	// l.copy()
	l.changed = true
	return true
}

func (l *byteListV2) Clear() {
	// l.data = []byte{}
	l.data = l.data[:0]
	l.setList(l.data)
	if l.chgr != nil {
		l.chgr.SetSubChange()
	}
	// l.copy()
	l.changed = true
}

func (l *byteListV2) GetAllReadOnly() []byte {
	l.copy()
	return l.dataReadOnly
}

func (l *byteListV2) copy() {
	if l.changed {
		l.changed = false

		ret := make([]byte, len(l.data))
		copy(ret, l.data)
		l.dataReadOnly = ret
	}
}

func (l *byteListV2) setSubChange(chgr ISubChanger) {
	l.chgr = chgr
}
