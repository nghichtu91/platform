package savedbwrapper

import (
	"fmt"
	"testing"
)

func BenchmarkSaveDB_SaveDB(b *testing.B) {
	keyName := "testSaveDBKeyName"
	param := []string{"testParam"}
	flag := fmt.Sprintf("<SaveDB><%s>", keyName)

	b.Run("fmt", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_ = func() string {
				return fmt.Sprintf("<SaveDB><%s>", keyName)
			}
		}
	})

	b.Run("var", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_ = func() string {
				return flag
			}
		}
	})

	b.Run("fmt 2", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_ = func() string {
				return fmt.Sprintf("EXPIRE %s %s,", keyName, param[0])
			}
		}
	})

	b.Run("string plus 2", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_ = func() string {
				return "EXPIRE" + " " + keyName + " " + param[0]
			}
		}
	})
}
