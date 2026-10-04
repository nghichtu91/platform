package savedbwrapper

func NewboolList(setList SetBoolList, l []bool, chgr ISubChanger) IBoolList {
	// 如果初始化slice为空，给slice预分配默认长度
	if len(l) == 0 {
		l = make([]bool, 0, defaultSliceLen)
	}
	_r := &boolListV2{
		setList:      setList,
		data:         l,
		dataReadOnly: make([]bool, 0, len(l)),
		chgr:         chgr,
		changed:      true,
	}
	_r.copy()
	return _r
}

type boolListV2 struct {
	setList      SetBoolList
	data         []bool
	dataReadOnly []bool
	chgr         ISubChanger
	changed      bool
}

func (l *boolListV2) Len() int {
	return len(l.data)
}

func (l *boolListV2) Get(index int) (bool, bool) {
	if index >= len(l.data) {
		return false, false
	}
	return l.data[index], true
}

func (l *boolListV2) Set(index int, v bool) bool {
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

func (l *boolListV2) Remove(index int) bool {
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

func (l *boolListV2) Append(v bool) {
	l.data = append(l.data, v)
	l.setList(l.data)
	if l.chgr != nil {
		l.chgr.SetSubChange()
	}
	// l.copy()
	l.changed = true
}

func (l *boolListV2) Appends(v []bool) {
	l.data = append(l.data, v...)
	l.setList(l.data)
	if l.chgr != nil {
		l.chgr.SetSubChange()
	}
	// l.copy()
	l.changed = true
}

func (l *boolListV2) Insert(index int, v bool) bool {
	if index > len(l.data) {
		return false
	}
	// rear := append([]bool{}, l.data[index:]...)
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

func (l *boolListV2) Clear() {
	// l.data = []bool{}
	l.data = l.data[:0]
	l.setList(l.data)
	if l.chgr != nil {
		l.chgr.SetSubChange()
	}
	// l.copy()
	l.changed = true
}

func (l *boolListV2) GetAllReadOnly() []bool {
	l.copy()
	return l.dataReadOnly
}

func (l *boolListV2) copy() {
	if l.changed {
		l.changed = false

		ret := make([]bool, len(l.data))
		copy(ret, l.data)
		l.dataReadOnly = ret
	}
}

func (l *boolListV2) setSubChange(chgr ISubChanger) {
	l.chgr = chgr
}
