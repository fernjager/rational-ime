package main

import (
	"flag"
	"fmt"
	"strconv"
)

func main() {
	fmt.Println("\n********* Initializing Server **********")

	dbName := flag.String("db", "main.db", "Path to Chinese character DB")
	cacheFlag := flag.Bool("cache", true, "Use the cache?")
	flag.Parse()

	ref := NewReference(*dbName, *cacheFlag)
	InitServer(ref)
	_, num := ref.GetByChar("我")
	_, num = ref.GetByPinyin("wo3")
	_, num = ref.GetByPinyin("wo3")
	ref.Close()
	fmt.Println("num" + strconv.Itoa(num))
}
