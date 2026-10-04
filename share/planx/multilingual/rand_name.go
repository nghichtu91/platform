package multilingual

import (
	"bufio"
	"fmt"
	"io"
	"math/rand"
	"os"
	"strings"
)

func init() {
	multilingualNameLib = make(map[string]*NameLib)
}

var multilingualNameLib map[string]*NameLib

type NameLib struct {
	Territory  []string // 所属地。
	FirstPart  []string // 第一部分。
	SecondPart []string // 第二部分。
}

// LoadAppointLanguageNameLibByCSV
//
// 起服读入后，不再有写操作，无需加锁。
func LoadAppointLanguageNameLibByCSV(language string, csvPath string) error {
	fileHandler, openErr := os.Open(csvPath)
	if openErr != nil {
		return fmt.Errorf("LoadAppointLanguageNameLibByCSV open file failed, "+
			"language: %v, csvPath: %v, openErr: %v", language, csvPath, openErr)
	}
	defer fileHandler.Close()

	multilingualNameLib[language] = &NameLib{
		Territory:  make([]string, 0, 512),
		FirstPart:  make([]string, 0, 512),
		SecondPart: make([]string, 0, 512),
	}

	reader := bufio.NewReader(fileHandler)
	// TODO 此处仿照原方案实现，但并不代表认同这是一种合理的方案。
	for {
		line, readErr := reader.ReadString('\n')
		if readErr != nil {
			if readErr == io.EOF {
				break
			} else {
				return fmt.Errorf("LoadAppointLanguageNameLibByCSV ReadString failed, "+
					"language: %v, csvPath: %v, readErr: %v", language, csvPath, readErr)
			}
		}

		line = strings.TrimSpace(line)
		splitResult := strings.Split(line, "，")
		if len(splitResult) != 3 {
			continue
		}

		territory := strings.TrimSpace(splitResult[0])
		firstPart := strings.TrimSpace(splitResult[1])
		secondPart := strings.TrimSpace(splitResult[2])

		if territory != "" {
			multilingualNameLib[language].Territory = append(multilingualNameLib[language].Territory, territory)
		}
		if firstPart != "" {
			multilingualNameLib[language].FirstPart = append(multilingualNameLib[language].FirstPart, firstPart)
		}
		if secondPart != "" {
			multilingualNameLib[language].SecondPart = append(multilingualNameLib[language].SecondPart, secondPart)
		}
	}

	return nil
}

// RandomAccountName 指定语种随机起名。
func RandomAccountName(r *rand.Rand, language string) string {
	//特殊处理，如果是印尼语则用英语代替
	if language == Indonesia {
		language = English
	}

	nameLib, ok := multilingualNameLib[language]
	if !ok {
		// 没有找到对应language的名字库。
		nameLib, ok = multilingualNameLib[Chinese]
		if !ok {
			return ""
		}
	}

	territoryLib := nameLib.Territory
	firstPartLib := nameLib.FirstPart
	secondPartLib := nameLib.SecondPart

	territorySize := len(territoryLib)
	var territory string
	if territorySize == 1 {
		territory = territoryLib[0]
	} else if territorySize > 1 {
		territory = territoryLib[r.Int63n(int64(territorySize)-1)]
	}

	firstPartSize := len(firstPartLib)
	var firstPart string
	if firstPartSize == 1 {
		firstPart = firstPartLib[0]
	} else if firstPartSize > 1 {
		firstPart = firstPartLib[r.Int63n(int64(firstPartSize)-1)]
	}

	secondPartSize := len(secondPartLib)
	var secondPart string
	if secondPartSize == 1 {
		secondPart = secondPartLib[0]
	} else if secondPartSize > 1 {
		secondPart = secondPartLib[r.Int63n(int64(secondPartSize)-1)]
	}

	return territory + firstPart + secondPart
}

// RandomAccountNameWithCount 指定语种和数量随机起名。
func RandomAccountNameWithCount(r *rand.Rand, language string, count int) []string {
	var result []string
	for i := 0; i < count; i++ {
		result = append(result, RandomAccountName(r, language))
	}
	return result
}
