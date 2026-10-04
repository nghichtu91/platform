package sensitive

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSensitiveLib_Split2Words(t *testing.T) {
	var (
		in, out    string
		sensitives []string
		lib        = NewEmptyLib()
		has        bool
	)

	sensitives = []string{"will", "be", "tnnd", "mb", "中文", "及其"}

	t.Run("Test1", func(t *testing.T) {
		in = "This is a sentence with \n English, 中文，以及\t其他字符，" +
			"but only characters and numbers, e.g. 1,2,45 will be recorded.\n" +
			"Combines as iphone15 will also be recorded. \n\r"

		t.Logf("Original input %s", in)

		t.Run("China", func(t *testing.T) {
			time.LoadLocation("Asia/Shanghai")
			lib.AddSensitivesToMap(sensitives...)

			has = lib.HasSensitive(in)
			out = lib.Replace(in)

			t.Logf("original has sensitive: %v, replace: %s", has, out)
		})

		t.Run("Abroad", func(t *testing.T) {
			time.LoadLocation("Asia/Hong_Kong")
			lib.SetSensitiveWords(sensitives...)

			has = lib.HasSensitive(in)
			out = lib.Replace(in)

			t.Logf("original has sensitive: %v, replace: %s", has, out)
		})
	})

	t.Run("Test2", func(t *testing.T) {
		in = "tnnd tnndd ttnndd tn\tnd ttnndd tnd tnnd tnnd tnnd tnd 他nnd 去他tnnd tnnd没事 tn哒nd " +
			"tNND tNnDd tTnnDd tn\tND TTNNDD tNd tNNd tNnD TNND TND 他NnD 去他TNND tNnD没事 Tn哒nd"

		t.Logf("Original input 2 %s", in)

		t.Run("China", func(t *testing.T) {
			time.LoadLocation("Asia/Shanghai")
			lib.AddSensitivesToMap(sensitives...)

			has = lib.HasSensitive(in)
			out = lib.Replace(in)

			t.Logf("original 2 has sensitive: %v, replace: %s", has, out)
		})

		t.Run("Abroad", func(t *testing.T) {
			time.LoadLocation("Asia/Hong_Kong")
			lib.SetSensitiveWords(sensitives...)

			has = lib.HasSensitive(in)
			out = lib.Replace(in)

			t.Logf("original 2 has sensitive: %v, replace: %s", has, out)
		})
	})

	t.Run("HasSensitive", func(t *testing.T) {
		ins := []string{"及其", "及/其", "be", "tabel", " b e "}
		cHas := []bool{true, true, true, true, true}
		eHas := []bool{true, true, true, false, false}

		time.LoadLocation("Asia/Shanghai")
		lib.AddSensitivesToMap(sensitives...)

		for i := 0; i < len(ins); i++ {
			assert.Equal(t, cHas[i], lib.HasSensitive(ins[i]))
		}

		time.LoadLocation("Asia/Hong_Kong")
		lib.SetSensitiveWords(sensitives...)

		for i := 0; i < len(ins); i++ {
			assert.Equal(t, eHas[i], lib.HasSensitive(ins[i]))
		}

	})

	t.Run("大小写处理", func(t *testing.T) {
		ins := []string{"TNND OPPS", "tnnd opps", "Tnnd Opps", "tNND oPPS"}
		outs := []string{"**** OPPS", "**** opps", "**** Opps", "**** oPPS"}

		time.LoadLocation("Asia/Shanghai")
		lib.AddSensitivesToMap(sensitives...)

		for i := 0; i < len(ins); i++ {
			assert.Equal(t, outs[i], lib.Replace(ins[i]))
		}
	})

	t.Run("原词替换", func(t *testing.T) {
		orgs := []string{"", "Tnnd Opps", "这他妹不对"}
		tars := []string{"", "**** opps", "这**不对"}
		outs := []string{"", "**** Opps", "这**不对"}

		for i := 0; i < len(orgs); i++ {
			assert.Equal(t, outs[i], replaceOrg(orgs[i], tars[i]), i)
		}
	})
}

func (l SensitiveLib) isLorN(r uint8) bool {
	_, isNum := l.SkipNum[string(r)]
	_, isChar := l.SkipChar[string(r)]
	return !isNum && !isChar
}

func BenchmarkMatch(b *testing.B) {
	var (
		lib = NewEmptyLib()
		in  string
	)

	in = "This is a sentence with \n English, 中文，以及\t其他字符，" +
		"but only characters and numbers, e.g. 1,2,45 will be recorded.\n" +
		"Combines as iphone15 will also be recorded. \n\r"

	b.Run("lib", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			for j := 0; j < len(in); j++ {
				lib.isLorN(in[j])
			}
		}
	})

	b.Run("rune", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			for j := 0; j < len(in); j++ {
				lib.isDigitOrLetter(in[j])
			}
		}
	})
}
