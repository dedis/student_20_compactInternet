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

// Int64 converts strings to int64s
func Int64(a string) int64 {
	num, err := strconv.ParseInt(a, 10, 64)
	if err != nil {
		panic(err)
	}
	return num
}

// Str converts ints to strings
func Str(a int) string {
	return strconv.Itoa(a)
}

// Str64 converts int64s to strings
func Str64(a int64) string {
	return strconv.FormatInt(a, 10)
}

// Union performs the union of two sets
func Union(acculator map[int]bool, toAdd map[int]bool) map[int]bool {
	if acculator == nil {
		panic("Cannot perform union on nil accumulator")
	}
	if toAdd != nil {
		for e := range toAdd {
			acculator[e] = true
		}
	}
	return acculator
}
