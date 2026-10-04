package util

import (
	"bytes"
	"go/format"
	"log"
	"math/rand"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/nghichtu91/platform/share/planx/tilogs"

	"github.com/cenk/backoff"
)

const (
	ASyncCmdTimeOut = time.Millisecond * 500
)

func RandByOffsetUInt32(main, offset uint32, r *rand.Rand) uint32 {
	if offset == 0 {
		return main
	}

	if offset < 0 {
		offset = -offset
	}

	rv := uint32(r.Int63n(int64(offset) * 2))
	if main+rv < offset {
		return 0
	}
	return main + rv - offset
}

// 重试参数 三次, 总时间2s内
const (
	DefaultInitialInterval     = 500 * time.Millisecond
	DefaultRandomizationFactor = 0.5
	DefaultMultiplier          = 2
	DefaultMaxInterval         = 2 * time.Second
	DefaultMaxElapsedTime      = 2 * time.Second
)

func New2SecBackOff() *backoff.ExponentialBackOff {
	b := &backoff.ExponentialBackOff{
		InitialInterval:     DefaultInitialInterval,
		RandomizationFactor: DefaultRandomizationFactor,
		Multiplier:          DefaultMultiplier,
		MaxInterval:         DefaultMaxInterval,
		MaxElapsedTime:      DefaultMaxElapsedTime,
		Clock:               backoff.SystemClock,
	}
	if b.RandomizationFactor < 0 {
		b.RandomizationFactor = 0
	} else if b.RandomizationFactor > 1 {
		b.RandomizationFactor = 1
	}
	b.Reset()
	return b
}

// Proper ShutDown; 用来判断是否正常停服，若不正常停服，则不然启动
const running = "RUNNING"

func ProperShutDownCheck() {
	if FileExists(running) {
		tilogs.L().Errorf("last not shutdown properly， still has RUNNING file")
		// logs.Flush()
		// os.Exit(1)
		return
	}

	f, err := os.OpenFile(running, os.O_CREATE, os.FileMode(0644))
	if err != nil {
		f.Close()
		// logs.Flush()
		// os.Exit(1)
		return
	}
}

func ProperShutDownFinish() {
	os.Remove(running)
}

// GenFileWithTemplates 拼接模版到指定文件
func GenFileWithTemplates(outFile string, templates []string, params interface{}) {
	buffer := bytes.NewBuffer(make([]byte, 0, 4096))

	for _, tmpl := range templates {
		buffer.WriteString(tmpl)
	}

	WriteWithTemplateFile(outFile, buffer.String(), params)
}

// WriteWithTemplateFile 将模版内容写入文件，变量从param中读取
func WriteWithTemplateFile(filename, content string, params interface{}) {
	tmpl := template.New(filename)

	// 导入自定义函数
	// 这里后续可以加入更多自定义函数
	tmplMap := make(map[string]interface{}, 8)
	tmplMap["Title"] = Title

	tmpl.Funcs(tmplMap)

	_, err := tmpl.Parse(content)
	if err != nil {
		log.Fatal(err)
		return
	}

	f, err := os.Create(filename)
	if err != nil {
		log.Fatal(err)
		return
	}

	// 如果是go代码，格式化
	if strings.HasSuffix(filename, ".go") {
		buffer := bytes.NewBuffer(make([]byte, 0, 4096))

		err = tmpl.Execute(buffer, params)
		if err != nil {
			log.Fatalln("Execute template failed, ", err)
			return
		}

		src, err := format.Source(buffer.Bytes())
		if err != nil {
			log.Fatalln("format source failed,", err, string(buffer.Bytes()))
			return
		}

		_, err = f.Write(src)
		if err != nil {
			log.Fatal(err)
			return
		}
	} else {
		err = tmpl.Execute(f, params)
		if err != nil {
			log.Fatal(err)
			return
		}
	}
}

// Title 用于模版导入使用，效果 name -> Name
func Title(in string) string {
	return strings.Title(in)
}
