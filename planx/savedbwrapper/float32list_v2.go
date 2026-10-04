package savedbwrapper

func Newfloat32List(setList SetFloat32List, l []float32, chgr ISubChanger) IFloat32List {
	// 如果初始化slice为空，给slice预分配默认长度
	if len(l) == 0 {
		l = make([]float32, 0, defaultSliceLen)
	}
	_r := &float32ListV2{
		setList:      setList,
		data:         l,
		dataReadOnly: make([]float32, 0, len(l)),
		chgr:         chgr,
		changed:      true,
	}
	_r.copy()
	return _r
}

type float32ListV2 struct {
	setList      SetFloat32List
	data         []float32
	dataReadOnly []float32
	chgr         ISubChanger
	changed      bool
}

func (l *float32ListV2) Len() int {
	return len(l.data)
}

func (l *float32ListV2) Get(index int) (float32, bool) {
	if index >= len(l.data) {
		return 0, false
	}
	return l.data[index], true
}

func (l *float32ListV2) Set(index int, v float32) bool {
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

func (l *float32ListV2) Remove(index int) bool {
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

func (l *float32ListV2) Append(v float32) {
	l.data = append(l.data, v)
	l.setList(l.data)
	if l.chgr != nil {
		l.chgr.SetSubChange()
	}
	// l.copy()
	l.changed = true
}

func (l *float32ListV2) Appends(v []float32) {
	l.data = append(l.data, v...)
	l.setList(l.data)
	if l.chgr != nil {
		l.chgr.SetSubChange()
	}
	// l.copy()
	l.changed = true
}

func (l *float32ListV2) Insert(index int, v float32) bool {
	if index > len(l.data) {
		return false
	}
	// rear := append([]float32{}, l.data[index:]...)
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

func (l *float32ListV2) Clear() {
	// l.data = []float32{}
	l.data = l.data[:0]
	l.setList(l.data)
	if l.chgr != nil {
		l.chgr.SetSubChange()
	}
	// l.copy()
	l.changed = true
}

func (l *float32ListV2) GetAllReadOnly() []float32 {
	l.copy()
	return l.dataReadOnly
}

func (l *float32ListV2) copy() {
	if l.changed {
		l.changed = false

		ret := make([]float32, len(l.data))
		copy(ret, l.data)
		l.dataReadOnly = ret
	}
}

func (l *float32ListV2) setSubChange(chgr ISubChanger) {
	l.chgr = chgr
}
