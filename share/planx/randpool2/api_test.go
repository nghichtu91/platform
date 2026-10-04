package randpool2

import (
	"fmt"
	"log"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/nghichtu91/platform/share/planx/timeutil"
	"github.com/nghichtu91/platform/share/planx/util"

	"github.com/nghichtu91/platform/share/planx/savedbwrapper"

	protogen "github.com/nghichtu91/platform/share/planx/randpool2/fortest"
)

type counterForTest struct {
	ExpireTime  int64
	Count       uint64
	TotalWeight int64
	Copies      savedbwrapper.IUint32List
	Weights     savedbwrapper.IInt64List
	WWeights    savedbwrapper.IInt64List
	WeightIndex savedbwrapper.IInt32List
	SumCount    uint32
	PrizeCount  uint32
	PrizeIndex  uint32
	CurIndex    uint32
}

func (c *counterForTest) GetExpireTime() int64 {
	return c.ExpireTime
}
func (c *counterForTest) SetExpireTime(t int64) {
	c.ExpireTime = t
}
func (c *counterForTest) GetCount() uint64 {
	return c.Count
}
func (c *counterForTest) SetCount(n uint64) {
	c.Count = n
}

// arrays counter
func (c *counterForTest) GetTotalWeight() int64 {
	return c.TotalWeight
}
func (c *counterForTest) SetTotalWeight(tw int64) {
	c.TotalWeight = tw
}
func (c *counterForTest) GetCopies() savedbwrapper.IUint32List {
	if c.Copies == nil {
		c.Copies = savedbwrapper.Newuint32List(func([]uint32) {}, make([]uint32, 0, 16), nil)
	}
	return c.Copies
}
func (c *counterForTest) GetWeights() savedbwrapper.IInt64List {
	if c.Weights == nil {
		c.Weights = savedbwrapper.Newint64List(func([]int64) {}, make([]int64, 0, 16), nil)
	}
	return c.Weights
}
func (c *counterForTest) GetWWeights() savedbwrapper.IInt64List {
	if c.WWeights == nil {
		c.WWeights = savedbwrapper.Newint64List(func([]int64) {}, make([]int64, 0, 16), nil)
	}
	return c.WWeights
}
func (c *counterForTest) GetWeightIndex() savedbwrapper.IInt32List {
	if c.WeightIndex == nil {
		c.WeightIndex = savedbwrapper.Newint32List(func([]int32) {}, make([]int32, 0, 16), nil)
	}
	return c.WeightIndex
}

// seed counter
func (c *counterForTest) GetSumCount() uint32 {
	return c.SumCount
}
func (c *counterForTest) SetSumCount(sc uint32) {
	c.SumCount = sc
}
func (c *counterForTest) GetPrizeCount() uint32 {
	return c.PrizeCount
}
func (c *counterForTest) SetPrizeCount(pc uint32) {
	c.PrizeCount = pc
}
func (c *counterForTest) GetPrizeIndex() uint32 {
	return c.PrizeIndex
}
func (c *counterForTest) SetPrizeIndex(pi uint32) {
	c.PrizeIndex = pi
}
func (c *counterForTest) GetCurIndex() uint32 {
	return c.CurIndex
}
func (c *counterForTest) SetCurIndex(ci uint32) {
	c.CurIndex = ci
}

type counterMgrForTest struct {
	counters map[string]*counterForTest
}

func (cs *counterMgrForTest) NewCounter(id string) ICounterData {
	c := &counterForTest{}
	cs.counters[id] = c
	return c
}
func (cs *counterMgrForTest) GetCounter(id string) (ICounterData, bool) {
	c, ok := cs.counters[id]
	return c, ok
}
func (cs *counterMgrForTest) DelCounter(id string) {
	delete(cs.counters, id)
}

type profileForTest struct {
	lvl uint32
	vip uint32
	r   *rand.Rand
}

func (p *profileForTest) GetLevel() uint32 {
	return p.lvl
}
func (p *profileForTest) GetVip() uint32 {
	return p.vip
}
func (p *profileForTest) GetTimeStamp() int64 {
	return time.Now().Unix()
}
func (p *profileForTest) GetRand() *rand.Rand {
	return p.r
}

func (p *profileForTest) GetServerOpenDaysOrWarZoneLongest() uint32 {
	return 2
}

func (p *profileForTest) FitCond(rewardCondId uint32) bool {
	return true
}

func prepare() {
	protogen.LoadRANDOMDISTRIBUTIONData()
	protogen.LoadRANDOMLINKData()
	protogen.LoadREWARDSTATICData()
	protogen.LoadREWARDWEIGHTData()

	rdis = make([]IDistribution, 0, len(protogen.GetAllRANDOMDISTRIBUTIONS()))
	for _, r := range protogen.GetAllRANDOMDISTRIBUTIONS() {
		rdis = append(rdis, r)
	}

	rls = make([]IRewardLink, 0, len(protogen.GetAllRANDOMLINK()))
	for _, r := range protogen.GetAllRANDOMLINK() {
		rls = append(rls, r)
	}

	rss = make([]IRewardStaticOutput, 0, len(protogen.GetAllREWARDSTATIC()))
	for _, r := range protogen.GetAllREWARDSTATIC() {
		rss = append(rss, r)
	}

	rws = make([]IRewardWeightOutput, 0, len(protogen.GetAllREWARDWEIGHT()))
	for _, r := range protogen.GetAllREWARDWEIGHT() {
		rws = append(rws, r)
	}

	cmgrs = &counterMgrForTest{
		counters: make(map[string]*counterForTest, 16),
	}

	rng := util.Kiss64Rng{}
	rng.Seed(timeutil.Now().UnixNano())
	p = &profileForTest{
		r: rand.New(&rng),
	}
}

var (
	rss   []IRewardStaticOutput
	rws   []IRewardWeightOutput
	rls   []IRewardLink
	rdis  []IDistribution
	cmgrs *counterMgrForTest
	p     *profileForTest
)

func TestOutput(t *testing.T) {
	prepare()
	_, _, err := Load(rss, rws, rls, rdis)
	if err != nil {
		t.Fatalf("Load err %s", err.Error())
	}

	// output weight
	log.Println("----------------- output weight -----------------")
	runForTest("Weight_Rnd_1", 10, t)
	runForTest("Weight_Rnd_2_Fill", 10, t)

	// output static
	log.Println("----------------- output static -----------------")
	runForTest("Static_Rnd_1", 10, t)
	runForTest("Static_Rnd_3_Fill_1", 10, t)

	// Times
	log.Println("----------------- Times -----------------")
	runForTest("Link_Rnd_2", 30, t)

	// arrays
	log.Println("----------------- arrays -----------------")
	runForTest("Link_Rnd_2_T1", 20, t)
	runForTest("Link_Rnd_2_T10", 20, t)

	// Random
	log.Println("----------------- Random -----------------")
	runForTest("Link_Rnd_2_T11", 100, t)

	//redo
	log.Println("----------------- redo -----------------")
	runForTest("Link_Rnd_2_T16", 10, t)

	//Multiple
	log.Println("----------------- Multiple -----------------")
	runForTest("Static_Rnd_2_Big2", 10, t)

	//Seed
	log.Println("----------------- Seed -----------------")
	runForTest("Link_Rnd_3_Judge", 200, t)

	//SeedTimes
	log.Println("----------------- SeedTimes -----------------")
	runForTest("Link_Rnd_3_Hero_4", 10, t)
	runForTest("Link_Rnd_3_Hero_10", 30, t)
	runForTest("Link_Rnd_3_Hero_17", 30, t)
}

func runForTest(id string, n int, t *testing.T) {
	for i := 0; i < n; i++ {
		ls, err := Run(id, cmgrs, p)
		if err != nil {
			t.Fatalf("Run %s err %s", id, err.Error())
		}
		log.Println("run", id, i, lootString(ls))
	}
}
func lootString(ls []*Loot) string {
	sb := strings.Builder{}
	for _, l := range ls {
		sb.WriteString(fmt.Sprintf("%v ", l))
	}
	return sb.String()
}

func BenchmarkRun(b *testing.B) {
	prepare()
	_, _, err := Load(rss, rws, rls, rdis)
	if err != nil {
		b.Fatalf("Load err %s", err.Error())
	}

	b.Run("weight", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			Run("Weight_Rnd_2_Fill", cmgrs, p)
		}
	})

	b.Run("static", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			Run("Static_Rnd_3_Fill_1", cmgrs, p)
		}
	})

	b.Run("times", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			Run("Link_Rnd_2", cmgrs, p)
		}
	})

	b.Run("arrays", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			Run("Link_Rnd_2_T10", cmgrs, p)
		}
	})

	b.Run("random", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			Run("Link_Rnd_2_T11", cmgrs, p)
		}
	})

	b.Run("redo", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			Run("Link_Rnd_2_T16", cmgrs, p)
		}
	})

	b.Run("multi", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			Run("Static_Rnd_2_Big2", cmgrs, p)
		}
	})

	b.Run("seed", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			Run("Link_Rnd_3_Judge", cmgrs, p)
		}
	})

	b.Run("seed times", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			Run("Link_Rnd_3_Hero_17", cmgrs, p)
		}
	})
}
