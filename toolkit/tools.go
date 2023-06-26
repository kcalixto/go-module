package toolkit

import (
	"crypto/rand"
)

const randomStringSource = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_+"

type Tools struct{}

func (t *Tools) RandomString(length int) string {
	// rune is alias for int32
	s := make([]rune, length)
	// this code is converting these strings to their ASCII codes
	// since we're not trespassing 127, we're safe to use byte or int
	// but if we'd had emojis or other special characters, rune would've made
	// all the difference
	r := []rune(randomStringSource)
	// output: [97 98 99 100 101 102 103 104 105 106 107 108 109 110 111... etc]

	for i := range s {
		// now crazy shit starts, get ready...
		// This function generates a prime number containing 2ˆlen(r-1)+1 to 2ˆlen(r)-1 bits
		// considering our 64 characters long it's like:
		// 9.2 quintillions to 18 quintillions bits
		// remember that it generates a prime number?
		// to validate it, the code performs a primality test
		// one very known and used one is Miller-Rabin test
		// that iterates the generated number like 20 times
		// before returning the number for us
		// All of that to generated a 99.9% random number
		p, _ := rand.Prime(rand.Reader, len(r))

		// this part is just converting it to int64
		// so we can work with it properly :D
		x := p.Uint64()

		// this one is doing the same
		// converting so we can work better
		y := uint64(len(r))

		// And here we're taking the rest of
		// the division of 9.who_fucking_cares quintillions by 64
		// And this number can never be greater than 63
		// due to Modular Arithmetic:
		// 100 = 1 x 64 + 36
		// 200 = 3 x 64 + 8
		// 300 = 4 x 64 + 52
		// And it's not limited by 64
		// any positive number(x) / any positive number(y)
		// can never have a rest greater than dividend(y) - 1
		s[i] = r[x%y]
	}

	// finally we're converting all that rune array(bits of characters) into string
	return string(s)
}
