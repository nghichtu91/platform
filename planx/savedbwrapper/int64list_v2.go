package savedbwrapper

func Newint64List(setList SetInt64List, l []int64, chgr ISubChanger) IInt64List {
	// 如果初始化slice为空，给slice预分配默认长度
	if len(l) == 0 {
		l = make([]int64, 0, defaultSliceLen)
	}
	_r := &int64ListV2{
		setList:      setList,
		data:         l,
		dataReadOnly: make([]int64, 0, len(l)),
		chgr:         chgr,
		changed:      true,
	}
	_r.copy()
	return _r
}

type int64ListV2 struct {
	setList      SetInt64List
	data         []int64
	dataReadOnly []int64
	chgr         ISubChanger
	changed      bool
}

func (l *int64ListV2) Len() int {
	return len(l.data)
}

func (l *int64ListV2) Get(index int) (int64, bool) {
	if index >= len(l.data) {
		return 0, false
	}
	return l.data[index], true
}

func (l *int64ListV2) Set(index int, v int64) bool {
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

func (l *int64ListV2) Remove(index int) bool {
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

func (l *int64ListV2) Append(v int64) {
	l.data = append(l.data, v)
	l.setList(l.data)
	if l.chgr != nil {
		l.chgr.SetSubChange()
	}
	// l.copy()
	l.changed = true
}

func (l *int64ListV2) Appends(v []int64) {
	l.data = append(l.data, v...)
	l.setList(l.data)
	if l.chgr != nil {
		l.chgr.SetSubChange()
	}
	// l.copy()
	l.changed = true
}

func (l *int64ListV2) Insert(index int, v int64) bool {
	if index > len(l.data) {
		return false
	}
	// rear := append([]int64{}, l.data[index:]...)
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

func (l *int64ListV2) Clear() {
	// l.data = []int64{}
	l.data = l.data[:0]
	l.setList(l.data)
	if l.chgr != nil {
		l.chgr.SetSubChange()
	}
	// l.copy()
	l.changed = true
}

func (l *int64ListV2) GetAllReadOnly() []int64 {
	l.copy()
	return l.dataReadOnly[:]
}

func (l *int64ListV2) copy() {
	if l.changed {
		l.changed = false

		ret := make([]int64, len(l.data))
		copy(ret, l.data)
		l.dataReadOnly = ret
	}
}

func (l *int64ListV2) setSubChange(chgr ISubChanger) {
	l.chgr = chgr
}
