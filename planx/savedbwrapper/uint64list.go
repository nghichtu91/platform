package savedbwrapper

type IUint64List interface {
	Len() int
	Get(index int) (uint64, bool)
	Set(index int, v uint64) bool
	Remove(index int) bool
	Append(v uint64)
	Appends(v []uint64)
	Insert(index int, v uint64) bool
	Clear()
	GetAllReadOnly() []uint64 // 获取数组内容的拷贝，注意不要修改返回的数组
	setSubChange(chgr ISubChanger)
}

type SetUint64List func([]uint64)

func Newuint64ListOld(setList SetUint64List, l []uint64, chgr ISubChanger) IUint64List {
	_r := &Uint64List{
		setList: setList,
		data:    l,
		chgr:    chgr,
	}
	_r.copy()
	return _r
}

type Uint64List struct {
	setList      SetUint64List
	data         []uint64
	dataReadOnly []uint64
	chgr         ISubChanger
}

func (l *Uint64List) Len() int {
	return len(l.data)
}

func (l *Uint64List) Get(index int) (uint64, bool) {
	if index >= len(l.data) {
		return 0, false
	}
	return l.data[index], true
}

func (l *Uint64List) Set(index int, v uint64) bool {
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

func (l *Uint64List) Remove(index int) bool {
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

func (l *Uint64List) Append(v uint64) {
	l.data = append(l.data, v)
	l.setList(l.data)
	if l.chgr != nil {
		l.chgr.SetSubChange()
	}
	l.copy()
}

func (l *Uint64List) Appends(v []uint64) {
	l.data = append(l.data, v...)
	l.setList(l.data)
	if l.chgr != nil {
		l.chgr.SetSubChange()
	}
	l.copy()
}

func (l *Uint64List) Insert(index int, v uint64) bool {
	if index > len(l.data) {
		return false
	}
	rear := append([]uint64{}, l.data[index:]...)
	l.data = append(append(l.data[:index], v), rear...)
	l.setList(l.data)
	if l.chgr != nil {
		l.chgr.SetSubChange()
	}
	l.copy()
	return true
}

func (l *Uint64List) Clear() {
	l.data = []uint64{}
	l.setList(l.data)
	if l.chgr != nil {
		l.chgr.SetSubChange()
	}
	l.copy()
}
func (l *Uint64List) GetAllReadOnly() []uint64 {
	return l.dataReadOnly
}

func (l *Uint64List) copy() {
	ret := make([]uint64, len(l.data))
	copy(ret, l.data)
	l.dataReadOnly = ret
}

func (l *Uint64List) setSubChange(chgr ISubChanger) {
	l.chgr = chgr
}
