package u

import "strconv"

// Int converts strings to ints
func Int(a string) int {
	num, err := strconv.Atoi(a)
	if err != nil {
		panic(err)
	}
	return num
}

// Str converts ints to strings
func Str(a int) string {
	return strconv.Itoa(a)
}
