package protogen

import (
	"encoding/json"
	"os"
)

var (
	dataC *GameDataCollection
)

func init() {
	dataC = &GameDataCollection{}
}

type GameDataCollection struct {
	randomdistributions   []*RANDOMDISTRIBUTION
	randomdistributionmap map[uint32]*RANDOMDISTRIBUTION
	randominputs          []*RANDOMINPUT
	randominputmap        map[string]*RANDOMINPUT
	randomlinks           []*RANDOMLINK
	rewardstatics         []*REWARDSTATIC
	rewardweights         []*REWARDWEIGHT
}

func LoadRANDOMDISTRIBUTIONData() {
	ar := new(RANDOMDISTRIBUTION_ARRAY)
	buffer, err := os.Open("./fortest/data/RandomDistribution.json")
	panicIfError(err)

	decoder := json.NewDecoder(buffer)
	e := decoder.Decode(ar)
	panicIfError(e)

	dataC.randomdistributions = ar.GetItems()
	if len(dataC.randomdistributions) > 100 {
		dataC.randomdistributionmap = make(map[uint32]*RANDOMDISTRIBUTION, len(ar.GetItems()))
		for _, item := range dataC.randomdistributions {
			dataC.randomdistributionmap[item.GetID()] = item
		}
	}
}

func GetAllRANDOMDISTRIBUTIONS() []*RANDOMDISTRIBUTION {
	return dataC.randomdistributions
}

func LoadRANDOMLINKData() {
	ar := new(RANDOMLINK_ARRAY)
	buffer, err := os.Open("./fortest/data/RandomLink.json")
	panicIfError(err)

	decoder := json.NewDecoder(buffer)
	e := decoder.Decode(ar)
	panicIfError(e)

	dataC.randomlinks = ar.GetItems()
}

func GetAllRANDOMLINK() []*RANDOMLINK {
	return dataC.randomlinks
}

func LoadREWARDSTATICData() {
	ar := new(REWARDSTATIC_ARRAY)
	buffer, err := os.Open("./fortest/data/RewardStatic.json")
	panicIfError(err)

	decoder := json.NewDecoder(buffer)
	e := decoder.Decode(ar)
	panicIfError(e)

	dataC.rewardstatics = ar.GetItems()

}

func GetAllREWARDSTATIC() []*REWARDSTATIC {
	return dataC.rewardstatics
}

func LoadREWARDWEIGHTData() {
	ar := new(REWARDWEIGHT_ARRAY)
	buffer, err := os.Open("./fortest/data/RewardWeight.json")
	panicIfError(err)

	decoder := json.NewDecoder(buffer)
	e := decoder.Decode(ar)
	panicIfError(e)

	dataC.rewardweights = ar.GetItems()
}

func GetAllREWARDWEIGHT() []*REWARDWEIGHT {
	return dataC.rewardweights
}

func panicIfError(err error) {
	if err != nil {
		panic(err)
	}
}
