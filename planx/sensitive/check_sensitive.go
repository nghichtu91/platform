package sensitive

import (
	"strings"
	"unicode"

	"github.com/nghichtu91/platform/share/planx/timeutil"
)

const (
	SkipSymbols = " ,~,!,@,#,$,%,^,&,*,(,),_,-,+,=,?,<,>,，,。,/,\\,|,《,》,？,;,:,：,',‘,；,“,\t,\r,\n,~,!,@,#,$,%,^,&,*,(,),_,+,-,=,【,】,、,{,},|,;,',:,\",，,。,、,《,》,？,α,β,γ,δ,ε,ζ,η,θ,ι,κ,λ,μ,ν,ξ,ο,π,ρ,σ,τ,υ,φ,χ,ψ,ω,Α,Β,Γ,Δ,Ε,Ζ,Η,Θ,Ι,Κ,Λ,Μ,Ν,Ξ,Ο,Π,Ρ,Σ,Τ,Υ,Φ,Χ,Ψ,Ω,。,，,、,；,：,？,！,…,—,·,ˉ,¨,‘,’,“,”,々,～,‖,∶,＂,＇,｀,｜,〃,〔,〕,〈,〉,《,》,「,」,『,』,．,〖,〗,【,】,（,）,［,］,｛,｝,Ⅰ,Ⅱ,Ⅲ,Ⅳ,Ⅴ,Ⅵ,Ⅶ,Ⅷ,Ⅸ,Ⅹ,Ⅺ,Ⅻ,⒈,⒉,⒊,⒋,⒌,⒍,⒎,⒏,⒐,⒑,⒒,⒓,⒔,⒕,⒖,⒗,⒘,⒙,⒚,⒛,㈠,㈡,㈢,㈣,㈤,㈥,㈦,㈧,㈨,㈩,①,②,③,④,⑤,⑥,⑦,⑧,⑨,⑩,⑴,⑵,⑶,⑷,⑸,⑹,⑺,⑻,⑼,⑽,⑾,⑿,⒀,⒁,⒂,⒃,⒄,⒅,⒆,⒇,≈,≡,≠,＝,≤,≥,＜,＞,≮,≯,∷,±,＋,－,×,÷,／,∫,∮,∝,∞,∧,∨,∑,∏,∪,∩,∈,∵,∴,⊥,∥,∠,⌒,⊙,≌,∽,√,§,№,☆,★,○,●,◎,◇,◆,□,℃,‰,€,■,△,▲,※,→,←,↑,↓,〓,¤,°,＃,＆,＠,＼,︿,＿,￣,―,♂,♀,┌,┍,┎,┐,┑,┒,┓,─,┄,┈,├,┝,┞,┟,┠,┡,┢,┣,│,┆,┊,┬,┭,┮,┯,┰,┱,┲,┳,┼,┽,┾,┿,╀,╁,╂,╃,└,┕,┖,┗,┘,┙,┚,┛,━,┅,┉,┤,┥,┦,┧,┨,┩,┪,┫,┃,┇,┋,┴,┵,┶,┷,┸,┹,┺,┻,╋,╊,╉,╈,╇,╆,╅,╄,"
	SkipNums    = "1,2,3,4,5,6,7,8,9,0"
	ASCIILowerA = 97
	ASCIILowerZ = 122
	ASCIIUpperA = 65
	ASCIIUpperZ = 90
)

const (
	sliceLimit = 256 // 整词屏蔽功能超过多少时使用map

	replaceChar = byte('*')
)

// SensitiveLib 屏蔽词库
type SensitiveLib struct {
	sensitiveWord map[string]interface{}
	SkipSymbol    map[string]struct{} // 不参与敏感词汇判断直接忽略
	SkipNum       map[string]struct{} // 不参与敏感词汇判断直接忽略
	SkipChar      map[string]struct{} // 不参与敏感词汇判断直接忽略

	useMap         bool
	wholeWordMap   map[string]struct{} // 整词屏蔽map
	wholeWordSlice []string            // 整词屏蔽slice
}

// NewEmptyLib 构建一个默认的屏蔽词库
// sensitiveWord 需要后续自行添加
func NewEmptyLib() *SensitiveLib {
	lib := new(SensitiveLib)
	lib.sensitiveWord = make(map[string]interface{}, 1024)
	lib.SkipSymbol = make(map[string]struct{}, 64)
	lib.SkipNum = make(map[string]struct{}, 64)
	lib.SkipChar = make(map[string]struct{}, 64)

	for _, v := range strings.Split(SkipSymbols, ",") {
		lib.SkipSymbol[v] = struct{}{}
	}
	for _, v := range strings.Split(SkipNums, ",") {
		lib.SkipNum[v] = struct{}{}
	}
	for i := ASCIIUpperA; i <= ASCIIUpperZ; i++ {
		lib.SkipChar[string(rune(i))] = struct{}{}
	}
	for i := ASCIILowerA; i <= ASCIILowerZ; i++ {
		lib.SkipChar[string(rune(i))] = struct{}{}
	}

	return lib
}

// AddSensitiveToMap 敏感词字典树插入元素。
func (l *SensitiveLib) AddSensitiveToMap(key string) {
	str := []rune(strings.ToLower(key))
	nowMap := l.sensitiveWord
	for i := 0; i < len(str); i++ {
		if _, ok := nowMap[string(str[i])]; !ok { // 如果该key不存在，
			thisMap := make(map[string]interface{}, 16)
			thisMap["isEnd"] = false
			nowMap[string(str[i])] = thisMap
			nowMap = thisMap
		} else {
			nowMap = nowMap[string(str[i])].(map[string]interface{})
		}
		if i == len(str)-1 {
			nowMap["isEnd"] = true
		}
	}
}

// AddSensitivesToMap 敏感词字典树批量插入元素。
func (l *SensitiveLib) AddSensitivesToMap(keys ...string) {
	for i := 0; i < len(keys); i++ {
		l.AddSensitiveToMap(keys[i])
	}
}

// DeleteSensitiveFromMap 敏感词从字典树中删除。
//
// Note: 删除节点若无子节点不会被清理，因为下次起服树会被重新构建。
//
// Param-key: 要从字典树中删除的敏感词。
func (l *SensitiveLib) DeleteSensitiveFromMap(key string) {
	uft8s := []rune(strings.ToLower(key))

	// 查询起始点赋值。
	nowMap := l.sensitiveWord

	for index := 0; index < len(uft8s); index++ {
		if nextMap, ok := nowMap[string(uft8s[index])].(map[string]interface{}); !ok {
			// 删除的目标敏感词，字典树中不存在，直接返回。
			return
		} else {
			if index == len(uft8s)-1 {
				// 如果已经到了字符串的最后一个字符。
				nextMap["isEnd"] = false
			}
			nowMap = nextMap
		}
	}
}

// Replace 检查并替换敏感字符为'*'。
//
// Note:
// 国服使用字典树
// 假设"卧槽"为敏感词，"卧@槽"也会被检查替换为***。
// 敏感词检查将会查找所有的敏感词和替换后出现的敏感词。
// 查找替换的终点为：所有的敏感词被找到，或所有的字符被替换为'*'。
// 海外使用全词匹配
// 只有子字符会被替换为'*'
func (l SensitiveLib) Replace(originStr string) string {
	needContinueCheck := true
	replaceStartIndex, replaceEndIndex := 0, 0
	tempStr := strings.ToLower(originStr)

	for needContinueCheck {
		needContinueCheck, replaceStartIndex, replaceEndIndex = l.CheckSensitive(tempStr)

		if needContinueCheck {
			// 如果检查出敏感词，执行*替换。
			trfStr := []rune(tempStr)
			for index := replaceStartIndex; index <= replaceEndIndex; index++ {
				trfStr[index] = '*'
			}
			tempStr = string(trfStr)

			// 防止*被加入敏感词导致的死循环。
			allStar := true
			for _, runeNum := range trfStr {
				if runeNum != '*' {
					allStar = false
					break
				}
			}
			if allStar {
				break
			}
		}
	}

	afterStr := replaceOrg(originStr, tempStr)

	// 如果是海外，再过一次全词匹配
	if !timeutil.IsAsiaShanghaiTZ() {
		return l.ReplaceWholeWord(afterStr)
	}

	return afterStr
}

// HasSensitive 封装是否有敏感词的函数
// 海外和国服通用
func (l SensitiveLib) HasSensitive(str string) bool {
	var has bool
	has, _, _ = l.CheckSensitive(str)
	if timeutil.IsAsiaShanghaiTZ() {
		return has
	}

	// 海外地区，如果字典树没问题，再过一次全词匹配
	if !has {
		has = l.HasWholeWordSensitive(str)
	}

	return has
}

// CheckSensitive 敏感词检测
// 这里原来检查多次 为了防止纯数字 纯字符的敏感字如 250 fuck
// return: bool->是否是敏感词 int,int->敏感词的索引起始结束位置
func (l SensitiveLib) CheckSensitive(txt string) (bool, int, int) {
	str := strings.ToLower(txt)

	ret, begin, end := checkSensitiveWords(str, l.sensitiveWord, false)
	if ret {
		return ret, begin, end
	}

	ret, begin, end = checkSensitiveWords(str, l.sensitiveWord, false, l.SkipNum)
	if ret {
		return ret, begin, end
	}

	ret, begin, end = checkSensitiveWords(str, l.sensitiveWord, false, l.SkipSymbol)
	if ret {
		return ret, begin, end
	}

	ret, begin, end = checkSensitiveWords(str, l.sensitiveWord, false, l.SkipChar)
	if ret {
		return ret, begin, end
	}

	ret, begin, end = checkSensitiveWords(str, l.sensitiveWord, true, l.SkipNum, l.SkipChar, l.SkipSymbol)
	if ret {
		return ret, begin, end
	}

	return false, 0, 0
}

// flag : true表跳过InvalidWords中字符 false表示只检查InvalidWords中字符
func checkSensitiveWords(txt string, sensitive map[string]interface{}, flag bool, InvalidWords ...map[string]struct{}) (bool, int, int) {
	str := []rune(strings.ToLower(txt))
	nowMap := sensitive
	start := -1
	tag := -1
	length := len(str)
	for i := 0; i < length; i++ {
		con := false
		for _, invalidWord := range InvalidWords {
			if _, ok := invalidWord[(string(str[i]))]; ok {
				if flag {
					con = true // 如果是无效词汇直接跳过
				}
			} else {
				if !flag {
					con = true
				}
			}
		}
		if con {
			continue
		}

		if thisMap, ok := nowMap[string(str[i])].(map[string]interface{}); ok {
			// 记录敏感词第一个文字的位置
			tag++
			if tag == 0 {
				start = i
			}
			// 判断是否为敏感词的最后一个文字
			if isEnd, _ := thisMap["isEnd"].(bool); isEnd {
				// 将敏感词的第一个文字到最后一个文字全部替换为“*”
				return true, start, i
			} else { // 不是最后一个，则将其包含的map赋值给nowMap
				nowMap = nowMap[string(str[i])].(map[string]interface{})
			}
		} else { // 如果敏感词不是全匹配，则终止此敏感词查找。从开始位置的第二个文字继续判断
			if start != -1 {
				// 这里如果+1 再加上i++ 开始位置的第二个了
				// i = start + 1
				i = start
			}
			// 重置标志参数
			nowMap = sensitive
			start = -1
			tag = -1
		}
	}

	// 还有一种情况就是 最后一个字符是敏感词 这里之后因为i++就直接返回false了
	if thisMap, ok := sensitive[string(str[length-1])].(map[string]interface{}); ok {
		if isEnd, _ := thisMap["isEnd"].(bool); isEnd {
			// 将敏感词的第一个文字到最后一个文字全部替换为“*”
			return true, length - 1, length - 1
		}
	}
	return false, 0, 0
}

// SetSensitiveWords 设置屏蔽词，海外地区使用
// 会将非英文屏蔽词放入字典树，英文单词放入全词匹配
// 需要一次性将所有敏感词传入
// 和其他函数一样，不保证线程安全，需要外部加锁
func (l *SensitiveLib) SetSensitiveWords(words ...string) {
	wholeWords := make([]string, 0, len(words))
	treeWords := make([]string, 0, len(words))

	for i := 0; i < len(words); i++ {
		if l.isWholeWord(words[i]) {
			wholeWords = append(wholeWords, words[i])
		} else {
			treeWords = append(treeWords, words[i])
		}
	}

	// 整词匹配
	l.initWholeWords(wholeWords)

	// 字典树
	l.sensitiveWord = make(map[string]interface{}, len(treeWords))
	l.AddSensitivesToMap(treeWords...)
}

func (l *SensitiveLib) initWholeWords(wholeWords []string) {
	if len(wholeWords) <= sliceLimit {
		l.wholeWordSlice = make([]string, 0, sliceLimit)
		l.useMap = false
		for i := 0; i < len(wholeWords); i++ {
			l.wholeWordSlice = append(l.wholeWordSlice, strings.ToLower(wholeWords[i]))
		}
	} else {
		l.wholeWordMap = make(map[string]struct{}, 1024)
		l.useMap = true
		for i := 0; i < len(wholeWords); i++ {
			l.wholeWordMap[strings.ToLower(wholeWords[i])] = struct{}{}
		}
	}
}

// SetSensitiveWordsMap 添加屏蔽整词，海外地区使用
// 会将非英文屏蔽词放入字典树，英文单词放入全词匹配
// 和其他函数一样，不保证线程安全，需要外部加锁
func (l *SensitiveLib) SetSensitiveWordsMap(words map[string]struct{}) {
	wholeWords := make([]string, 0, len(words))
	treeWords := make([]string, 0, len(words))

	for word := range words {
		if l.isWholeWord(word) {
			wholeWords = append(wholeWords, word)
		} else {
			treeWords = append(treeWords, word)
		}
	}

	// 整词匹配
	l.initWholeWords(wholeWords)

	// 字典树
	l.sensitiveWord = make(map[string]interface{}, len(treeWords))
	l.AddSensitivesToMap(treeWords...)
}

// checkWholeWord 以整词模式检查是否存在屏蔽词
func (l SensitiveLib) checkWholeWord(txt string) bool {
	word := strings.ToLower(txt)
	if l.useMap {
		_, ok := l.wholeWordMap[word]
		return ok
	}

	for i := 0; i < len(l.wholeWordSlice); i++ {
		if l.wholeWordSlice[i] == word {
			return true
		}
	}

	return false
}

// CheckWholeWord 检查并替换敏感字符为'*'
// 海外专用，只进行整词匹配
// 如果needReplace为true，表明为聊天模式，将所有敏感词替换为*
// 否则直接返回true，不进行替换
func (l SensitiveLib) CheckWholeWord(str string, needReplace bool) (bool, string) {
	var (
		word         string
		hasSensitive bool
		s, e         int
		byteSlice    []byte
	)
	txt := strings.ToLower(str)

	// 这里多循环一次，让下面的判断逻辑可以覆盖到最后一个字符
	for i := 0; i < len(txt)+1; i++ {
		// 跳过最后一次idx
		if i != len(txt) {
			// 只有数字和字母认为是有效输入
			if l.isDigitOrLetter(txt[i]) {
				// 标记本次单词的开始和结束
				if word == "" {
					s = i
				} else {
					e = i
				}
				word += string(txt[i])
				continue
			}
		}

		// 如果成词的对应逻辑
		if word != "" {
			if l.checkWholeWord(word) {
				hasSensitive = true
				// 如果不需要替换，这里直接返回即可
				if !needReplace {
					return hasSensitive, txt
				}
				// 如果需要替换，这里再初始化byte slice，最大程度节省资源
				if len(byteSlice) == 0 {
					byteSlice = []byte(txt)
				}
				// 进行替换
				for i := s; i <= e; i++ {
					byteSlice[i] = replaceChar
				}
			}
			word = ""
		}
	}

	if needReplace && hasSensitive {
		return hasSensitive, replaceOrg(str, string(byteSlice))
	}

	return hasSensitive, str
}

// isDigitOrLetter 判断是否为数字或字母
func (l SensitiveLib) isDigitOrLetter(r uint8) bool {
	return unicode.IsDigit(rune(r)) || unicode.IsLetter(rune(r))
}

// isWholeWord 检查是否应该放到全词匹配
// 如果只有字母和数字，放到全词匹配列表
// 否则用字典树
func (l SensitiveLib) isWholeWord(word string) bool {
	for i := 0; i < len(word); i++ {
		if !l.isDigitOrLetter(word[i]) {
			return false
		}
	}
	return true
}

// HasWholeWordSensitive 封装只检查是否有敏感词的函数
func (l SensitiveLib) HasWholeWordSensitive(txt string) bool {
	has, _ := l.CheckWholeWord(txt, false)
	return has
}

// ReplaceWholeWord 封装只替换的函数
func (l SensitiveLib) ReplaceWholeWord(txt string) string {
	_, ret := l.CheckWholeWord(txt, true)
	return ret
}

// 将替换后的文本置换回原文本
func replaceOrg(org, tar string) string {
	orgRune := []rune(org)
	tarRune := []rune(tar)
	for i := range tarRune {
		if tarRune[i] == rune(replaceChar) {
			orgRune[i] = rune(replaceChar)
		}
	}
	return string(orgRune)
}
