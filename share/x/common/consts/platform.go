package consts

// notice使用的平台

const (
	NoticePlatformWin     = "Win"
	NoticePlatformAndroid = "Android"
	NoticePlatformIOS     = "iOS"
	NoticePlatformMac     = "Mac"
	NoticePlatformCount   = iota // 新平台添加到Count上边, 并添加到下方的切片中
	NoticePlatformAll     = "All"
)

var (
	allNoticePlatform = [NoticePlatformCount]string{
		NoticePlatformWin,
		NoticePlatformAndroid,
		NoticePlatformIOS,
		NoticePlatformMac,
	}
)

// GetAllNoticePlatform 拿到后请不要修改其中的值, 数据应该是只读的
func GetAllNoticePlatform() []string {
	return allNoticePlatform[:]
}
