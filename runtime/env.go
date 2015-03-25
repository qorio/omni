package runtime

import (
	"os"
	"strconv"
)

func EnvString(key, def string) string {
	v := os.Getenv(key)
	if v != "" {
		return v
	} else {
		return def
	}
}

func EnvInt(key string, def int) int {
	v := os.Getenv(key)
	if i, e := strconv.Atoi(v); e == nil {
		return i
	} else {
		return def
	}
}
