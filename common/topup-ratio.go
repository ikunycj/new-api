package common

import (
	"fmt"
	"math"
	"sync"
)

var topupGroupRatio = map[string]float64{
	"default": 1,
	"vip":     1,
	"svip":    1,
}
var topupGroupRatioMutex sync.RWMutex

func TopupGroupRatio2JSONString() string {
	topupGroupRatioMutex.RLock()
	defer topupGroupRatioMutex.RUnlock()
	jsonBytes, err := Marshal(topupGroupRatio)
	if err != nil {
		SysError("error marshalling topup group ratio: " + err.Error())
	}
	return string(jsonBytes)
}

func UpdateTopupGroupRatioByJSONString(jsonStr string) error {
	updated := make(map[string]float64)
	if err := UnmarshalJsonStr(jsonStr, &updated); err != nil {
		return err
	}
	for name, ratio := range updated {
		if math.IsNaN(ratio) || math.IsInf(ratio, 0) || ratio < 0 {
			return fmt.Errorf("topup group ratio must be a non-negative finite number: %s", name)
		}
	}
	topupGroupRatioMutex.Lock()
	defer topupGroupRatioMutex.Unlock()
	topupGroupRatio = updated
	return nil
}

func GetTopupGroupRatio(name string) float64 {
	topupGroupRatioMutex.RLock()
	defer topupGroupRatioMutex.RUnlock()
	ratio, ok := topupGroupRatio[name]
	if !ok {
		SysError("topup group ratio not found: " + name)
		return 1
	}
	return ratio
}

func SetTopupGroupRatio(name string, ratio float64) error {
	if name == "" {
		return fmt.Errorf("topup group name cannot be empty")
	}
	if math.IsNaN(ratio) || math.IsInf(ratio, 0) || ratio < 0 {
		return fmt.Errorf("topup group ratio must be a non-negative finite number: %s", name)
	}
	topupGroupRatioMutex.Lock()
	defer topupGroupRatioMutex.Unlock()
	if topupGroupRatio == nil {
		topupGroupRatio = make(map[string]float64)
	}
	topupGroupRatio[name] = ratio
	return nil
}

func DeleteTopupGroupRatio(name string) {
	topupGroupRatioMutex.Lock()
	defer topupGroupRatioMutex.Unlock()
	delete(topupGroupRatio, name)
}
