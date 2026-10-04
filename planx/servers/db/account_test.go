package db

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseAccount(t *testing.T) {
	t.Run("ParseAccount", func(t *testing.T) {

		var testCases = []struct {
			account string
			want    Account
			wantErr bool
		}{
			{account: "1:2:" + NewUserID().String()},
			{account: "1:abc:" + NewUserID().String()},
			{account: "123a:abc:" + NewUserID().String()},
			{account: "1:2:"},
			{account: "1::3"},
			{account: ":2:3"},
			{account: "1:2"},
			{account: "1:"},
			{account: ":3"},
			{account: "1:2:3:4"},
		}

		for _, tc := range testCases {
			acc, err := ParseAccount(tc.account)
			acc2, err2 := ParseAccountFast(tc.account)
			require.Equal(t, acc, acc2, tc.account)
			require.Equal(t, err, err2, tc.account)
		}
	})
}

func BenchmarkParseAccount(b *testing.B) {
	var accountId = []string{
		"1:2:" + NewUserID().String(),
		//"1:",
	}

	b.Run("ParseAccount", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, acc := range accountId {
				_, _ = ParseAccount(acc)
			}
		}
		b.ReportAllocs()
	})

	b.ResetTimer()

	b.Run("ParseAccountFast", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, acc := range accountId {
				_, _ = ParseAccountFast(acc)
			}
		}
		b.ReportAllocs()
	})
}
