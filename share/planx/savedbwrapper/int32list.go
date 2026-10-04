package savedbwrapper

type IInt32List interface {
	Len() int
	Get(index int) (int32, bool)
	Set(index int, v int32) bool
	Remove(index int) bool
	Append(v int32)
	Appends(v []int32)
	Insert(index int, v int32) bool
	Clear()
	GetAllReadOnly() []int32 // 获取数组内容的拷贝，注意不要修改返回的数组
	setSubChange(chgr ISubChanger)
}

type SetInt32List func([]int32)

func Newint32ListOld(setList SetInt32List, l []int32, chgr ISubChanger) IInt32List {
	_r := &Int32List{
		setList: setList,
		data:    l,
		chgr:    chgr,
	}
	_r.copy()
	return _r
}

type Int32List struct {
	setList      SetInt32List
	data         []int32
	dataReadOnly []int32
	chgr         ISubChanger
}

func (l *Int32List) Len() int {
	return len(l.data)
}

func (l *Int32List) Get(index int) (int32, bool) {
	if index >= len(l.data) {
		return 0, false
	}
	return l.data[index], true
}

func (l *Int32List) Set(index int, v int32) bool {
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

func (l *Int32List) Remove(index int) bool {
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

func (l *Int32List) Append(v int32) {
	l.data = append(l.data, v)
	l.setList(l.data)
	if l.chgr != nil {
		l.chgr.SetSubChange()
	}
	l.copy()
}

func (l *Int32List) Appends(v []int32) {
	l.data = append(l.data, v...)
	l.setList(l.data)
	if l.chgr != nil {
		l.chgr.SetSubChange()
	}
	l.copy()
}

func (l *Int32List) Insert(index int, v int32) bool {
	if index > len(l.data) {
		return false
	}
	rear := append([]int32{}, l.data[index:]...)
	l.data = append(append(l.data[:index], v), rear...)
	l.setList(l.data)
	if l.chgr != nil {
		l.chgr.SetSubChange()
	}
	l.copy()
	return true
}

func (l *Int32List) Clear() {
	l.data = []int32{}
	l.setList(l.data)
	if l.chgr != nil {
		l.chgr.SetSubChange()
	}
	l.copy()
}

func (l *Int32List) GetAllReadOnly() []int32 {
	return l.dataReadOnly
}

func (l *Int32List) copy() {
	ret := make([]int32, len(l.data))
	copy(ret, l.data)
	l.dataReadOnly = ret
}

func (l *Int32List) setSubChange(chgr ISubChanger) {
	l.chgr = chgr
}
