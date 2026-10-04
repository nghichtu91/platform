package savedbwrapper

type IInt64List interface {
	Len() int
	Get(index int) (int64, bool)
	Set(index int, v int64) bool
	Remove(index int) bool
	Append(v int64)
	Appends(v []int64)
	Insert(index int, v int64) bool
	Clear()
	GetAllReadOnly() []int64 // 获取数组内容的拷贝，注意不要修改返回的数组
	setSubChange(chgr ISubChanger)
}

type SetInt64List func([]int64)

func Newint64ListOld(setList SetInt64List, l []int64, chgr ISubChanger) IInt64List {
	_r := &Int64List{
		setList: setList,
		data:    l,
		chgr:    chgr,
	}
	_r.copy()
	return _r
}

type Int64List struct {
	setList      SetInt64List
	data         []int64
	dataReadOnly []int64
	chgr         ISubChanger
}

func (l *Int64List) Len() int {
	return len(l.data)
}

func (l *Int64List) Get(index int) (int64, bool) {
	if index >= len(l.data) {
		return 0, false
	}
	return l.data[index], true
}

func (l *Int64List) Set(index int, v int64) bool {
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

func (l *Int64List) Remove(index int) bool {
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

func (l *Int64List) Append(v int64) {
	l.data = append(l.data, v)
	l.setList(l.data)
	if l.chgr != nil {
		l.chgr.SetSubChange()
	}
	l.copy()
}

func (l *Int64List) Appends(v []int64) {
	l.data = append(l.data, v...)
	l.setList(l.data)
	if l.chgr != nil {
		l.chgr.SetSubChange()
	}
	l.copy()
}

func (l *Int64List) Insert(index int, v int64) bool {
	if index > len(l.data) {
		return false
	}
	rear := append([]int64{}, l.data[index:]...)
	l.data = append(append(l.data[:index], v), rear...)
	l.setList(l.data)
	if l.chgr != nil {
		l.chgr.SetSubChange()
	}
	l.copy()
	return true
}

func (l *Int64List) Clear() {
	l.data = []int64{}
	l.setList(l.data)
	if l.chgr != nil {
		l.chgr.SetSubChange()
	}
	l.copy()
}

func (l *Int64List) GetAllReadOnly() []int64 {
	return l.dataReadOnly
}

func (l *Int64List) copy() {
	ret := make([]int64, len(l.data))
	copy(ret, l.data)
	l.dataReadOnly = ret
}

func (l *Int64List) setSubChange(chgr ISubChanger) {
	l.chgr = chgr
}
