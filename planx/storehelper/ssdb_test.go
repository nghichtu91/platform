package storehelper

//"sync"
//"testing"
//"github.com/nghichtu91/platform/share/planx/util/logs"

/*
func TestScanAll(t *testing.T) {
	store := NewStoreSSDB("127.0.0.1", 18888)

	db, _ := store.GetConn()

	res, _ := db.Keys("", "", 100)

	var wg sync.WaitGroup

	for i := 0; i < len(res); i++ {
		tilogs.L().Errorf("res %s", res[i])
		wg.Add(1)
		go func(re string) {
			d, _ := store.GetConn()
			defer d.Close()
			res, _ := d.Get(re)
			tilogs.L().Infof("res %v", res)

			wg.Done()
		}(res[i])
	}

	wg.Wait()

	logs.Close()
}
*/
