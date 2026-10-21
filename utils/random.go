package utils

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

const alphabet = "abcdefghijklmnopqrstuvwxyz"
const numbers = "0123456789"

func init() {
	rand.Seed(time.Now().UnixNano())
}

// RandomInt generates a random integer between min and max.
func RandomInt(min, max int64) int64 {
	return min + rand.Int63n(max-min+1)
}

// RandomString generates a random string of length n.
func RandomString(n int) string {
	var sb strings.Builder

	k := len(alphabet)

	for i := 0; i < n; i++ {
		c := alphabet[rand.Intn(k)]
		sb.WriteByte(c)
	}

	return sb.String()
}
func RandomNumbers(n int) string {
	var sb strings.Builder

	k := len(numbers)

	for i := 0; i < n; i++ {
		c := numbers[rand.Intn(k)]
		sb.WriteByte(c)
	}

	return sb.String()
}

// RandomFirstName generates a random first name.
func RandomFirstName() string {
	return RandomString(5)
}

// RandomLastName generates a random last name.
func RandomLastName() string {
	return RandomString(5)
}

// RandomEmail generates a random email address.
func RandomEmail() string {
	return fmt.Sprintf("%s@gmail.com", RandomString(10))
}

// RandomPhone generates a random phone number.
func RandomPhone() string {
	return fmt.Sprintf("06%d", RandomInt(10000000, 99999999))
}

// RandomPasswordHash generates a random password hash for testing.
func RandomPasswordHash() string {
	return RandomString(20)
}

// RandomStatus generates a random user status.
func RandomStatus() string {
	statuses := []string{
		"user",
		"admin",
	}

	return statuses[rand.Intn(len(statuses))]
}
