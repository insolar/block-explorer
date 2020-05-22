// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/insolar/blob/master/LICENSE.md
//

package generator

import (
	"go/build"
	"math"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/satori/go.uuid"
)

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
var intRunes = []rune("1234567890")
var extraCharsBase64 = []rune("-_")
var SpecialChars = []rune("!@#$%^&*()+\\?,.<>±§{}[]|'\"`~;: ")
var lettersAndIntRunes = append(letterRunes, intRunes...)
var base64Runes = append(lettersAndIntRunes, extraCharsBase64...)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// GetGoPath returns system GOPATH if it is presented, otherwise return default value
func GetGoPath() string {
	goPath := os.Getenv("GOPATH")
	if goPath == "" {
		goPath = build.Default.GOPATH
	}
	return goPath
}

func RandomString() string {
	newUUID, err := uuid.NewV4()
	if err != nil {
		panic(err)
	}
	return newUUID.String()
}

func RandBase64(length int) string {
	b := make([]rune, length)
	for i := range b {
		b[i] = base64Runes[rand.Intn(len(base64Runes))]
	}
	return string(b)
}

// RandCharOverString returns one random character taken from the specified string
func RandCharOverString(fromString string) string {
	str := []rune(fromString)
	return string(str[rand.Intn(len(str))])
}

// RandStringRunes generates random string with specified length
func RandStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

// RandStringRunesNumber generates random string of digits with specified length.
func RandStringRunesNumber(length int) string {
	if length == 1 {
		return string(intRunes[rand.Intn(9)])
	}
	b := make([]rune, length)
	for i := range b {
		b[i] = intRunes[rand.Intn(len(intRunes))]
	}
	return string(b)
}

// RandNumber generates random number with specified length
func RandNumber(length int) int64 {
	if length <= 0 || length > 19 {
		panic("incorrect length (int64 limit)")
	}
	n1 := math.Pow10(length - 1)
	n2 := math.Pow10(length)
	number := RandNumberOverRange(int64(n1), int64(n2))
	if len(strconv.FormatInt(number, 10)) > length {
		return number - 1
	} else {
		return number
	}
}

// RandNumberString generates random number with specified length and returns it as string
func RandNumberString(length int) string {
	return strconv.FormatInt(RandNumber(length), 10)
}

// RandNumberOverRange generates random number over a range
func RandNumberOverRange(min int64, max int64) int64 {
	return rand.Int63n(max-min+1) + min
}
