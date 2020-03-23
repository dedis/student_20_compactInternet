package main

import (
	"fmt"
)

type SomeStruct struct {
	content []int
	number  int
}

func memAlloc() *SomeStruct {
	return &SomeStruct{[]int{1, 2}, 19}
}

func main() {
	fmt.Println("It works!")
	var retVal *SomeStruct = memAlloc()

	fmt.Println(retVal.number)
	fmt.Println(retVal.content)

	mp := make(map[int]string)

	mp[19] = "This is a string"
	mp[2] = "This is another one"

	fmt.Println(mp[19])
	fmt.Println(mp[2])
	fmt.Println(mp[1])
	fmt.Println(string('↑'))
}
