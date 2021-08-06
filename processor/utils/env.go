package utils

import "fmt"

func ConvertEnvMp2StrSlice(mp map[string]string) (ret []string) {
	for k, v := range mp {
		ret = append(ret, fmt.Sprintf("%s=%s", k, v))
	}
	return ret
}
