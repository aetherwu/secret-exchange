package utils

import (
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/globalsign/mgo/bson"
	"github.com/satori/go.uuid"
)

func Filter(vs []interface{}, f func(interface{}) bool) []interface{} {
	vsf := make([]interface{}, 0)
	for _, v := range vs {
		if f(v) {
			vsf = append(vsf, v)
		}
	}
	return vsf
}

func Map(vs []interface{}, f func(interface{}) interface{}) []interface{} {
	vsm := make([]interface{}, len(vs))
	for i, v := range vs {
		vsm[i] = f(v)
	}
	return vsm
}

func WriteFile(data []byte, path string) error {
	err := ioutil.WriteFile(path, data, 0644)
	return err
}

func Random(min, max int) int {
	return rand.Intn(max-min) + min
}

func RandomWithSeed(min, max int, seed int64) int {
	rand.Seed(seed)
	return rand.Intn(max-min) + min
}

func UUID() string {
	return uuid.NewV4().String()
}

func GetCurrentDir() string {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}
	return strings.Replace(dir, "\\", "/", -1)
}

func ObjectIDRemoveRep(slc []bson.ObjectId) []bson.ObjectId {
	result := []bson.ObjectId{}
	for i := range slc {
		flag := true
		for j := range result {
			if slc[i] == result[j] {
				flag = false
				break
			}
		}
		if flag {
			result = append(result, slc[i])
		}
	}
	return result
}

func StringRemoveRep(slc []string) []string {
	result := []string{}
	for i := range slc {
		flag := true
		for j := range result {
			if slc[i] == result[j] {
				flag = false
				break
			}
		}
		if flag {
			result = append(result, slc[i])
		}
	}
	return result
}

func ZeroTomorrow() time.Time {
	now := time.Now()
	zero := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	return zero.AddDate(0, 0, 1)
}

func LastWeekDate() time.Time {
	now := time.Now()
	return now.AddDate(0, 0, -7)
}
