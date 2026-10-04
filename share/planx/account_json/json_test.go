package accountJson

import (
	"testing"

	"github.com/ngaut/log"
)

func TestPrueJsonBase(t *testing.T) {
	jsonstr := `{
  "AA1": {
    "BB1": 1234567,
    "BB2": "sddsfa",
    "BB3": "1234567",
     "BB31": "",
      "BB32": "null",
       "BB33": "{}",
       "BB34": "[]",
    "BB4": [
      1,
      2,
      3,
      4,
      6
    ],
    "BB7": "{\"ddd\":\"dsfsdfds\"}"
  },
  "AA2": {
    "BB5": "asasas",
    "BB6": {
      "CC1": 234523432,
      "CC2": "{\"ddd\":\"dsfsdfds\"}"
    }
  }
}`

	res, err := MkPrueJson(jsonstr)
	if err != nil {
		log.Error("MkPrueJson Err By %s", err.Error())
		return
	}

	resStr, err := res.Encode()

	if res != nil && err == nil {
		log.Info("res %v", string(resStr))
	}

	resres, err := FromPureJsonToOld(string(resStr))
	if err != nil {
		log.Error("FromPureJsonToOld Err By %s", err.Error())
		return
	}

	resresStr, err := resres.Encode()

	if resres != nil && err == nil {
		log.Info("resres %v", string(resresStr))
	}

}
