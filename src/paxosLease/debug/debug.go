package debug

import (
	"fmt"
	"regexp"
	"time"
)

var conds = map[string]bool{}

func SetCond(cond string) {
	conds[cond] = true
}

func UnsetCond(cond string) {
	conds[cond] = false
}

func HasCond(expects ...string) bool {
	for _, cond := range expects {
		if conds[cond] {
			return true
		}
	}
	return false
}

func ListConds(pattern string) []string {
	ret := make([]string, 0)
	for cond, _ := range conds {
		if a, _ := regexp.MatchString(pattern, cond); a {
			ret = append(ret, cond)
		}
	}
	return ret
}

func If(cond string, fns ...func(string)) {
	if conds[cond] {
		fns[0](cond)
	} else {
		if len(fns) > 1 {
			fns[1](cond)
		}
	}
}

func Unless(cond string, fn func(string)) {
	if !conds[cond] {
		fn(cond)
	}
}

func StuckIf(cond string) {
	If(cond, func(cond string) {
		for {
			if !conds[cond] {
				return
			}
			time.Sleep(100 * time.Millisecond)
		}
	})
}

func TmpLogf(a string, args ...interface{}) {
	fmt.Printf(a, args...)
}

func ResetConds() {
	conds = map[string]bool{}
}
