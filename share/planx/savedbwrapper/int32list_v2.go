package savedbwrapper

func Newint32List(setList SetInt32List, l []int32, chgr ISubChanger) IInt32List {
	// 如果初始化slice为空，给slice预分配默认长度
	if len(l) == 0 {
		l = make([]int32, 0, defaultSliceLen)
	}
	_r := &int32ListV2{
		setList:      setList,
		data:         l,
		dataReadOnly: make([]int32, 0, len(l)),
		chgr:         chgr,
		changed:      true,
	}
	_r.copy()
	return _r
}

type int32ListV2 struct {
	setList      SetInt32List
	data         []int32
	dataReadOnly []int32
	chgr         ISubChanger
	changed      bool
}

func (l *int32ListV2) Len() int {
	return len(l.data)
}

func (l *int32ListV2) Get(index int) (int32, bool) {
	if index >= len(l.data) {
		return 0, false
	}
	return l.data[index], true
}

func (l *int32ListV2) Set(index int, v int32) bool {
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

func (l *int32ListV2) Remove(index int) bool {
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

func (l *int32ListV2) Append(v int32) {
	l.data = append(l.data, v)
	l.setList(l.data)
	if l.chgr != nil {
		l.chgr.SetSubChange()
	}
	// l.copy()
	l.changed = true
}

func (l *int32ListV2) Appends(v []int32) {
	l.data = append(l.data, v...)
	l.setList(l.data)
	if l.chgr != nil {
		l.chgr.SetSubChange()
	}
	// l.copy()
	l.changed = true
}

func (l *int32ListV2) Insert(index int, v int32) bool {
	if index > len(l.data) {
		return false
	}
	// rear := append([]int32{}, l.data[index:]...)
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

func (l *int32ListV2) Clear() {
	// l.data = []int32{}
	l.data = l.data[:0]
	l.setList(l.data)
	if l.chgr != nil {
		l.chgr.SetSubChange()
	}
	// l.copy()
	l.changed = true
}

func (l *int32ListV2) GetAllReadOnly() []int32 {
	l.copy()
	return l.dataReadOnly
}

func (l *int32ListV2) copy() {
	if l.changed {
		l.changed = false

		ret := make([]int32, len(l.data))
		copy(ret, l.data)
		l.dataReadOnly = ret
	}
}

func (l *int32ListV2) setSubChange(chgr ISubChanger) {
	l.chgr = chgr
}
