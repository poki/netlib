package util

import "os"

func Getenv(key, def string) string {
	if val, found := os.LookupEnv(key); found {
		return val
	}
	return def
}
