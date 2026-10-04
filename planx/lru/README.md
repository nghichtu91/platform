# lru
提供了通用的lru缓存实现

Cache为无锁缓存，不支持并发

IntCache进行了部分优化，指定key类型为int64，相比较Cache大约提升了20%的性能，同样不支持并发

MuxCache使用锁避免race，相比较而言性能下降了30%，并且会有锁竞争性能问题，应避免在高并发场景使用

 ```
goos: darwin
goarch: amd64
pkg: vcs.taiyouxi.net/jws2/common/lru
cpu: Intel(R) Core(TM) i7-8569U CPU @ 2.80GHz
BenchmarkCache
BenchmarkCache/cache_put-8         	26158824	        44.94 ns/op	       0 B/op	       0 allocs/op
BenchmarkCache/cache_get-8         	27460354	        38.56 ns/op	       0 B/op	       0 allocs/op
BenchmarkCache/cache_prune-8       	   21103	     59231 ns/op	   57416 B/op	       3 allocs/op
BenchmarkCache/int_cache_put-8     	31949641	        32.13 ns/op	       0 B/op	       0 allocs/op
BenchmarkCache/int_cache_get-8     	35176322	        29.08 ns/op	       0 B/op	       0 allocs/op
BenchmarkCache/int_cache_prune-8   	   27168	     44928 ns/op	   41032 B/op	       3 allocs/op
BenchmarkCache/mux_cache_put-8     	21063753	        50.54 ns/op	       0 B/op	       0 allocs/op
BenchmarkCache/mux_cache_get-8     	19341176	        57.96 ns/op	       0 B/op	       0 allocs/op
BenchmarkCache/mux_cache_prune-8   	   19338	     59730 ns/op	   57416 B/op	       3 allocs/op
 ```