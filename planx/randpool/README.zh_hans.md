# randpool

为游戏掉落设计的随机池

## 使用

### 初始化

```golang
    rc = RandomContext{
        RandomConfig:  DefaultConfig(),
        RandomProfile: DefaultProfile(),
        RandomParams:  DefaultParam(rand.New(rand.NewSource(time.Now().Unix()))),
    }
```

### 存储配置

```golang
    data, err := MarshalSource(source)
    if err != nil {
        log.Fatalf("marshal from %v fail", source)
        return
    }
    rc.SetData(source.ID, data)
```

### 生成掉落

```golang
    result, err := Run("groupID", rc)
    if err != nil {
        log.Fatalf("marshal from %v fail", source)
        return nil
    }
    return result
```

## 功能实现和结构

### RandomContex

- 用于控制所有随机变量
- 包括3个接口: RandomConfig, RandomProfile, RandomParams

### RandomResult

- 用于随机结果传出
- 包括两个部分: Loots和RandomParams

### RandomParams

```golang
    // RandomParams 随机参数，包括需要从外部获取的参数，时间戳，随机种子，递归次数等等
    type RandomParams interface {
        GetValue(vType RangeVariantType) int
        GetTimestamp() int64
        GetRand() *rand.Rand
        AddRecursionCount(n int) error
        GetRecursionCount() int
    }
```

- 用于传递随机过程需要的参数
- 同时也传递用于约束和检查的参数

### RandomData

```golang
    // RandomData 单条随机数据
    type RandomData interface {
        SetBaseInfo(bi *BaseInfo)  // 设置基础信息
        GetType() RandomType       // 获取随机类型
        GenCounter() RandomCounter // 生成计数器
    }
```

- 单条随机数据
- 用于生成计数器RandomCounter，以及产生随机过程结果

### RandomConfig

```golang
    // RandomConfig 存储RandomData
    type RandomConfig interface {
        SetData(id string, rd RandomData)
        GetData(id string) RandomData
    }
```

- 管理从策划数据生成的RandomData

### RandomCounter

```golang
    // RandomCounter 计数器
    type RandomCounter interface {
        IsAvailable(tNow int64) bool // 判断Counter是否有效
        GetCount() int               // 获取当前计数，同时counter增加一次记录。Times类型：返回计数 Shuffle类型：返回序号
    }
```

- 计数器
- 本身带有过期标记
- 部分类型用于生成随机结果

### RandomProfile

```golang
// RandomProfile 管理Counter
    type RandomProfile interface {
        GetCounter(id string) RandomCounter
        SetCounter(id string, rc RandomCounter)
        DelCounter(id string)
        CleanUpCounters(timeNow int64)
    }
```

- 管理Counter

### RandomSource

```golang
    // RandomSource 提供配置的接口，目前的pb文件可以直接使用接口
    type RandomSource interface {
        GetID() string
        GetSaveType() string
        GetSaveParam() uint32
        GetRewardType() string
        GetRewardParam() string
    }
```

- 提供配置的接口
- 兼容从proto文件生成的pb.go

### RandomMiddleware

```golang
// RandomMiddleware 原始json -> RandomMiddleware -> RandomData过程的中间件
    type RandomMiddleware interface {
        // FromJson 解析策划表传递的配置json字符串
        FromJson(jsonStr string) error
        // ToData 生成不带BaseInfo的RandomData结构
        ToData() (RandomData, error)
    }
```

- 处理json文件的中间件
- 单独为每个类型写转换逻辑
