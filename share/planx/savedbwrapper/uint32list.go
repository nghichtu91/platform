package savedbwrapper

type IUint32List interface {
	Len() int
	Get(index int) (uint32, bool)
	Set(index int, v uint32) bool
	Remove(index int) bool
	Append(v uint32)
	Appends(v []uint32)
	Insert(index int, v uint32) bool
	Clear()
	GetAllReadOnly() []uint32 // 获取数组内容的拷贝，注意不要修改返回的数组
	setSubChange(chgr ISubChanger)
}

type SetUint32List func([]uint32)

func Newuint32ListOld(setList SetUint32List, l []uint32, chgr ISubChanger) IUint32List {
	_r := &Uint32List{
		setList: setList,
		data:    l,
		chgr:    chgr,
	}
	_r.copy()
	return _r
}

type Uint32List struct {
	setList      SetUint32List
	data         []uint32
	dataReadOnly []uint32
	chgr         ISubChanger
}

func (l *Uint32List) Len() int {
	return len(l.data)
}

func (l *Uint32List) Get(index int) (uint32, bool) {
	if index >= len(l.data) {
		return 0, false
	}
	return l.data[index], true
}

func (l *Uint32List) Set(index int, v uint32) bool {
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

func (l *Uint32List) Remove(index int) bool {
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

func (l *Uint32List) Append(v uint32) {
	l.data = append(l.data, v)
	l.setList(l.data)
	if l.chgr != nil {
		l.chgr.SetSubChange()
	}
	l.copy()
}

func (l *Uint32List) Appends(v []uint32) {
	l.data = append(l.data, v...)
	l.setList(l.data)
	if l.chgr != nil {
		l.chgr.SetSubChange()
	}
	l.copy()
}

func (l *Uint32List) Insert(index int, v uint32) bool {
	if index > len(l.data) {
		return false
	}
	rear := append([]uint32{}, l.data[index:]...)
	l.data = append(append(l.data[:index], v), rear...)
	l.setList(l.data)
	if l.chgr != nil {
		l.chgr.SetSubChange()
	}
	l.copy()
	return true
}

func (l *Uint32List) Clear() {
	l.data = []uint32{}
	l.setList(l.data)
	if l.chgr != nil {
		l.chgr.SetSubChange()
	}
	l.copy()
}

func (l *Uint32List) GetAllReadOnly() []uint32 {
	return l.dataReadOnly
}

func (l *Uint32List) copy() {
	ret := make([]uint32, len(l.data))
	copy(ret, l.data)
	l.dataReadOnly = ret
}

func (l *Uint32List) setSubChange(chgr ISubChanger) {
	l.chgr = chgr
}
