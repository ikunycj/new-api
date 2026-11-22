package model

import (
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

const fingerprintModelAliasesOptionKey = "FingerprintTestModelAliases"

type fingerprintModelAliasGroups map[string]int

// Published snapshots are immutable and replaced under the option map lock.
var fingerprintModelAliases fingerprintModelAliasGroups

func parseFingerprintModelAliases(value string) (fingerprintModelAliasGroups, error) {
	var groups [][]string
	if err := common.UnmarshalJsonStr(value, &groups); err != nil || groups == nil {
		return nil, errors.New("指纹测试模型别名必须是模型名数组的数组")
	}
	aliases := make(fingerprintModelAliasGroups)
	for i, group := range groups {
		if len(group) < 2 {
			return nil, fmt.Errorf("第 %d 组至少需要两个不同的模型名", i+1)
		}
		for _, name := range group {
			name = strings.TrimSpace(name)
			if name == "" {
				return nil, fmt.Errorf("第 %d 组包含空模型名", i+1)
			}
			if strings.ContainsAny(name, ",，、\r\n") {
				return nil, fmt.Errorf("第 %d 组的模型名 %q 包含分隔符", i+1, name)
			}
			key := strings.ToLower(name)
			if existing, ok := aliases[key]; ok {
				if existing == i {
					return nil, fmt.Errorf("第 %d 组的模型名 %q 重复", i+1, name)
				}
				return nil, fmt.Errorf("模型名 %q 同时出现在第 %d 组和第 %d 组", name, existing+1, i+1)
			}
			aliases[key] = i
		}
	}
	return aliases, nil
}

func currentFingerprintModelAliases() fingerprintModelAliasGroups {
	common.OptionMapRWMutex.RLock()
	defer common.OptionMapRWMutex.RUnlock()
	return fingerprintModelAliases
}

func (aliases fingerprintModelAliasGroups) sameModel(testModel, prediction string) bool {
	if strings.EqualFold(testModel, prediction) {
		return true
	}
	testGroup, testGrouped := aliases[strings.ToLower(testModel)]
	predictedGroup, predictionGrouped := aliases[strings.ToLower(prediction)]
	return testGrouped && predictionGrouped && testGroup == predictedGroup
}
