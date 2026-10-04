package savedbwrapper

type IBoolList interface {
	Len() int
	Get(index int) (bool, bool)
	Set(index int, v bool) bool
	Remove(index int) bool
	Append(v bool)
	Appends(v []bool)
	Insert(index int, v bool) bool
	Clear()
	GetAllReadOnly() []bool // 获取数组内容的拷贝，注意不要修改返回的数组
	setSubChange(chgr ISubChanger)
}

type SetBoolList func([]bool)

func NewboolListOld(setList SetBoolList, l []bool, chgr ISubChanger) IBoolList {
	_r := &BoolList{
		setList: setList,
		data:    l,
		chgr:    chgr,
	}
	_r.copy()
	return _r
}

type BoolList struct {
	setList      SetBoolList
	data         []bool
	dataReadOnly []bool
	chgr         ISubChanger
}

func (l *BoolList) Len() int {
	return len(l.data)
}

func (l *BoolList) Get(index int) (bool, bool) {
	if index >= len(l.data) {
		return false, false
	}
	return l.data[index], true
}

func (l *BoolList) Set(index int, v bool) bool {
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

func (l *BoolList) Remove(index int) bool {
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

func (l *BoolList) Append(v bool) {
	l.data = append(l.data, v)
	l.setList(l.data)
	if l.chgr != nil {
		l.chgr.SetSubChange()
	}
	l.copy()
}

func (l *BoolList) Appends(v []bool) {
	l.data = append(l.data, v...)
	l.setList(l.data)
	if l.chgr != nil {
		l.chgr.SetSubChange()
	}
	l.copy()
}

func (l *BoolList) Insert(index int, v bool) bool {
	if index > len(l.data) {
		return false
	}
	rear := append([]bool{}, l.data[index:]...)
	l.data = append(append(l.data[:index], v), rear...)
	l.setList(l.data)
	if l.chgr != nil {
		l.chgr.SetSubChange()
	}
	l.copy()
	return true
}

func (l *BoolList) Clear() {
	l.data = []bool{}
	l.setList(l.data)
	if l.chgr != nil {
		l.chgr.SetSubChange()
	}
	l.copy()
}

func (l *BoolList) GetAllReadOnly() []bool {
	return l.dataReadOnly
}

func (l *BoolList) copy() {
	ret := make([]bool, len(l.data))
	copy(ret, l.data)
	l.dataReadOnly = ret
}

func (l *BoolList) setSubChange(chgr ISubChanger) {
	l.chgr = chgr
}
