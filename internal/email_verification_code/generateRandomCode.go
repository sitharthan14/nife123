package emailverificationcode

import (
	"crypto/rand"
	"log"
	"math/big"
)



func sixDigits() int64 {
	max := big.NewInt(999999)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		log.Println(err)
	}
	return n.Int64()
}

func RandomNumber6Digit() int64 {
	var randNo int64
	for i := 0; i < 5; i++ {
		s := sixDigits()
		randNo = s
	}

	return randNo
}