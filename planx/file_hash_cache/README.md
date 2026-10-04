# file_cache
主要用于工具比较文件的历史差异

内部实现了3个cache

### cache
不支持并发，是另外两个cache的基础实现

### cache_m
使用mutex实现并发支持的cache

### cache_c
使用channel实现并发支持的cache