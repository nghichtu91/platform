package savedbwrapper

func NewstringList(setList SetStringList, l []string, chgr ISubChanger) IStringList {
	// 如果初始化slice为空，给slice预分配默认长度
	if len(l) == 0 {
		l = make([]string, 0, defaultSliceLen)
	}
	_r := &StringListV2{
		setList:      setList,
		data:         l,
		dataReadOnly: make([]string, 0, len(l)),
		chgr:         chgr,
		changed:      true,
	}
	_r.copy()
	return _r
}

type StringListV2 struct {
	setList      SetStringList
	data         []string
	dataReadOnly []string
	chgr         ISubChanger
	changed      bool
}

func (l *StringListV2) Len() int {
	return len(l.data)
}

func (l *StringListV2) Get(index int) (string, bool) {
	if index >= len(l.data) {
		return "", false
	}
	return l.data[index], true
}

func (l *StringListV2) Set(index int, v string) bool {
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

func (l *StringListV2) Remove(index int) bool {
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

func (l *StringListV2) Append(v string) {
	l.data = append(l.data, v)
	l.setList(l.data)
	if l.chgr != nil {
		l.chgr.SetSubChange()
	}
	// l.copy()
	l.changed = true
}

func (l *StringListV2) Appends(v []string) {
	l.data = append(l.data, v...)
	l.setList(l.data)
	if l.chgr != nil {
		l.chgr.SetSubChange()
	}
	// l.copy()
	l.changed = true
}

func (l *StringListV2) Insert(index int, v string) bool {
	if index > len(l.data) {
		return false
	}
	// rear := append([]string{}, l.data[index:]...)
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

func (l *StringListV2) Clear() {
	// l.data = []string{}
	l.data = l.data[:0]
	l.setList(l.data)
	if l.chgr != nil {
		l.chgr.SetSubChange()
	}
	// l.copy()
	l.changed = true
}
func (l *StringListV2) GetAllReadOnly() []string {
	l.copy()
	return l.dataReadOnly
}

func (l *StringListV2) copy() {
	if l.changed {
		l.changed = false

		ret := make([]string, len(l.data))
		copy(ret, l.data)
		l.dataReadOnly = ret
	}
}

func (l *StringListV2) setSubChange(chgr ISubChanger) {
	l.chgr = chgr
}
