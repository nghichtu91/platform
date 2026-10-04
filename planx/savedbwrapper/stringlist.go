package savedbwrapper

type IStringList interface {
	Len() int
	Get(index int) (string, bool)
	Set(index int, v string) bool
	Remove(index int) bool
	Append(v string)
	Appends(v []string)
	Insert(index int, v string) bool
	Clear()
	GetAllReadOnly() []string // 获取数组内容的拷贝，注意不要修改返回的数组
	setSubChange(chgr ISubChanger)
}

type SetStringList func([]string)

func NewstringListOld(setList SetStringList, l []string, chgr ISubChanger) IStringList {
	_r := &StringList{
		setList: setList,
		data:    l,
		chgr:    chgr,
	}
	_r.copy()
	return _r
}

type StringList struct {
	setList      SetStringList
	data         []string
	dataReadOnly []string
	chgr         ISubChanger
}

func (l *StringList) Len() int {
	return len(l.data)
}

func (l *StringList) Get(index int) (string, bool) {
	if index >= len(l.data) {
		return "", false
	}
	return l.data[index], true
}

func (l *StringList) Set(index int, v string) bool {
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

func (l *StringList) Remove(index int) bool {
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

func (l *StringList) Append(v string) {
	l.data = append(l.data, v)
	l.setList(l.data)
	if l.chgr != nil {
		l.chgr.SetSubChange()
	}
	l.copy()
}

func (l *StringList) Appends(v []string) {
	l.data = append(l.data, v...)
	l.setList(l.data)
	if l.chgr != nil {
		l.chgr.SetSubChange()
	}
	l.copy()
}

func (l *StringList) Insert(index int, v string) bool {
	if index > len(l.data) {
		return false
	}
	rear := append([]string{}, l.data[index:]...)
	l.data = append(append(l.data[:index], v), rear...)
	l.setList(l.data)
	if l.chgr != nil {
		l.chgr.SetSubChange()
	}
	l.copy()
	return true
}

func (l *StringList) Clear() {
	l.data = []string{}
	l.setList(l.data)
	if l.chgr != nil {
		l.chgr.SetSubChange()
	}
	l.copy()
}
func (l *StringList) GetAllReadOnly() []string {
	return l.dataReadOnly
}

func (l *StringList) copy() {
	ret := make([]string, len(l.data))
	copy(ret, l.data)
	l.dataReadOnly = ret
}

func (l *StringList) setSubChange(chgr ISubChanger) {
	l.chgr = chgr
}
