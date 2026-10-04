package savedbwrapper

type IFloat32List interface {
	Len() int
	Get(index int) (float32, bool)
	Set(index int, v float32) bool
	Remove(index int) bool
	Append(v float32)
	Appends(v []float32)
	Insert(index int, v float32) bool
	Clear()
	GetAllReadOnly() []float32 // 获取数组内容的拷贝，注意不要修改返回的数组
	setSubChange(chgr ISubChanger)
}

type SetFloat32List func([]float32)

func Newfloat32ListOld(setList SetFloat32List, l []float32, chgr ISubChanger) IFloat32List {
	_r := &Float32List{
		setList: setList,
		data:    l,
		chgr:    chgr,
	}
	_r.copy()
	return _r
}

type Float32List struct {
	setList      SetFloat32List
	data         []float32
	dataReadOnly []float32
	chgr         ISubChanger
}

func (l *Float32List) Len() int {
	return len(l.data)
}

func (l *Float32List) Get(index int) (float32, bool) {
	if index >= len(l.data) {
		return 0, false
	}
	return l.data[index], true
}

func (l *Float32List) Set(index int, v float32) bool {
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

func (l *Float32List) Remove(index int) bool {
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

func (l *Float32List) Append(v float32) {
	l.data = append(l.data, v)
	l.setList(l.data)
	if l.chgr != nil {
		l.chgr.SetSubChange()
	}
	l.copy()
}

func (l *Float32List) Appends(v []float32) {
	l.data = append(l.data, v...)
	l.setList(l.data)
	if l.chgr != nil {
		l.chgr.SetSubChange()
	}
	l.copy()
}

func (l *Float32List) Insert(index int, v float32) bool {
	if index > len(l.data) {
		return false
	}
	rear := append([]float32{}, l.data[index:]...)
	l.data = append(append(l.data[:index], v), rear...)
	l.setList(l.data)
	if l.chgr != nil {
		l.chgr.SetSubChange()
	}
	l.copy()
	return true
}

func (l *Float32List) Clear() {
	l.data = []float32{}
	l.setList(l.data)
	if l.chgr != nil {
		l.chgr.SetSubChange()
	}
	l.copy()
}

func (l *Float32List) GetAllReadOnly() []float32 {
	return l.dataReadOnly
}

func (l *Float32List) copy() {
	ret := make([]float32, len(l.data))
	copy(ret, l.data)
	l.dataReadOnly = ret
}

func (l *Float32List) setSubChange(chgr ISubChanger) {
	l.chgr = chgr
}
