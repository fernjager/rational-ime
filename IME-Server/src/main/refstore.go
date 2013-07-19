package main

import (
	"bytes"
	"code.google.com/p/gosqlite/sqlite"
	"container/list"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
)

// Character is an object that stores a Chinese character
// along with its various properties- pinyin, zhuyin, definition
type Character struct {
	Id         int
	Character  string
	Zhuyin     string
	Pinyin     string
	Tone       int
	Definition string
	Freq       int
}

// Phrase is an object that stores a Chinese language phrase string
// along with its definition
type Phrase struct {
	Id         int
	Character  int
	Phrase     string
	Definition string
	Freq       int
}

// CharLookupRequest is an object that contains a partially filled out
// character object. It is sent as a query to the DB thread to fetch
// full character candidates
type CharLookupRequest struct {
	Char      Character
	WriteBack chan *CharLookupResponse
}

// CharLookupResponse is an object that contains a list of complete character
// objects. It is returned by the DB thread as the result of a character query
type CharLookupResponse struct {
	CharList   *list.List
	NumResults int
}

// ReferenceStore is an object that serves as an in-memory cache for the DB,
// holds the handle for the DB connection, and holds the request queue channel
// for character lookup by the DB thread
type ReferenceStore struct {
	conn         *sqlite.Conn
	requestQueue chan *CharLookupRequest
	zhuyinCache  map[string]*CharLookupResponse
	pinyinCache  map[string]*CharLookupResponse
}

// GetByChar retrieves full candidate characters, given a UTF-8 Chinese character
func (ref ReferenceStore) GetByChar(char string) (*list.List, int) {
	writeBack := make(chan *CharLookupResponse)
	char = strings.TrimSpace(char)
	queryInfo := Character{-1, char, "", "", -1, "", -1}
	ref.requestQueue <- &CharLookupRequest{queryInfo, writeBack}
	response := <-writeBack
	return response.CharList, response.NumResults
}

// GetByZhuyin retrieves full candidate characters, given a UTF-8 zhuyin string
func (ref ReferenceStore) GetByZhuyin(zhuyin string) (*list.List, int) {
	writeBack := make(chan *CharLookupResponse)
	zhuyin = strings.TrimSpace(zhuyin)
	queryInfo := Character{-1, "", zhuyin, "", -1, "", -1}
	ref.requestQueue <- &CharLookupRequest{queryInfo, writeBack}
	response := <-writeBack
	return response.CharList, response.NumResults
}

// GetByZhuyin retrieves full candidate characters, given a pinyin string
func (ref ReferenceStore) GetByPinyin(pinyin string) (*list.List, int) {
	writeBack := make(chan *CharLookupResponse)
	pinyin = strings.TrimSpace(pinyin)
	queryInfo := Character{-1, "", "", pinyin, -1, "", -1}
	ref.requestQueue <- &CharLookupRequest{queryInfo, writeBack}
	response := <-writeBack
	return response.CharList, response.NumResults
}

// GetByDefinition retreives full candidate characters, given an English definition
func (ref ReferenceStore) GetByDefinition(definition string) (*list.List, int) {
	writeBack := make(chan *CharLookupResponse)
	definition = strings.TrimSpace(definition)
	queryInfo := Character{-1, "", "", "", -1, definition, -1}
	ref.requestQueue <- &CharLookupRequest{queryInfo, writeBack}
	response := <-writeBack
	return response.CharList, response.NumResults
}

// Get is the base lookup function called only by the DB thread
func (ref ReferenceStore) Get(partialChar Character) *CharLookupResponse {
	var resultCount = 0
	var toneString string

	// first, check the cache
	if val, ok := ref.zhuyinCache[partialChar.Zhuyin]; ok {
		return val
	}

	if val, ok := ref.pinyinCache[partialChar.Pinyin]; ok {
		return val
	}

	searchStmt, _ := ref.conn.Prepare(`SELECT * FROM characters WHERE
						character LIKE ? AND
						zhuyin LIKE ? AND 
						pinyin LIKE ? AND 
						tone LIKE ? AND
						definition LIKE ? 
						LIMIT 50`)
	if partialChar.Id == -1 {
		toneString = ""
	} else {
		toneString = strconv.Itoa(partialChar.Tone)
	}

	err := searchStmt.Exec(
		"%"+partialChar.Character+"%",
		"%"+partialChar.Zhuyin+"%",
		"%"+partialChar.Pinyin+"%",
		"%"+toneString+"%",
		"%"+partialChar.Definition+"%")
	if err != nil {
		fmt.Println("Error while Selecting: %s", err)
	}
	charList := list.New()
	for searchStmt.Next() {
		var zhuyin Character
		err = searchStmt.Scan(&zhuyin.Id, &zhuyin.Character, &zhuyin.Zhuyin, &zhuyin.Pinyin, &zhuyin.Tone, &zhuyin.Definition, &zhuyin.Freq)
		if err != nil {
			fmt.Printf("Error while getting row data: %s\n", err)
			os.Exit(1)
		}

		fmt.Printf("Id => %d\n", zhuyin.Id)
		fmt.Printf("Title => %s\n", zhuyin.Character)
		charList.PushBack(zhuyin)
		resultCount++
	}


	// TODO: if zhuyin is not empty, cache there, etc.
	//ref.zhuyinCache[CharLookupResposeCacheEntry{*charList,resultCount}
	return &CharLookupResponse{charList, resultCount}
}

// requestThread is the "DB thread", an internal running goroutine
// that handles lookup requests coming into the requestQueue channel
func (ref ReferenceStore) requestThread() {
	for request := range ref.requestQueue {
		request.WriteBack <- ref.Get(request.Char)
	}
}

// Cleans up chan and shuts down, saving the cache
func (ref ReferenceStore) shutDown() {
	close(ref.requestQueue)
	// write cache to file
	buffer := new(bytes.Buffer)
	enc := gob.NewEncoder(buffer)
	// This is safe, since go dereferences the pointers to get the actual objects
	enc.Encode(ref.zhuyinCache)
	err := ioutil.WriteFile("zhuyinCache.gob", buffer.Bytes(), 0600)
	if err != nil {
		panic(err)
	}
	buffer = new(bytes.Buffer)
	enc = gob.NewEncoder(buffer)
	enc.Encode(ref.pinyinCache)
	err = ioutil.WriteFile("pinyinCache.gob", buffer.Bytes(), 0600)
	if err != nil {
		panic(err)
	}

}

// NewReference initializes the database and returns a Reference object
func NewReference(dbName string, useCache bool) *ReferenceStore {
	ref := ReferenceStore{nil, make(chan *CharLookupRequest), make(map[string]*CharLookupResponse), make(map[string]*CharLookupResponse)}
	conn, err := sqlite.Open(dbName)
	if err != nil {
		fmt.Println("Unable to open the database: %s", err)
		os.Exit(1)
	}
	ref.conn = conn

	// Create and initialize the database, if it is not yet populated
	ref.conn.Exec(`CREATE TABLE IF NOT EXISTS characters( id INTEGER PRIMARY KEY AUTOINCREMENT,
							      character VARCHAR(4), 
							      zhuyin VARCHAR(12), 
							      pinyin VARCHAR(5), 
							      tone INTEGER, definition VARCHAR(50), 
							      freq INT );`)
	ref.conn.Exec(`CREATE TABLE IF NOT EXISTS phrases( id INTEGER PRIMARY KEY AUTOINCREMENT, 
							   character INT, 
							   phrase VARCHAR(50), 
							   definition TEXT, 
							   freq INT) `)

	//insertSql := `INSERT INTO characters(character, zhuyin, pinyin, tone, definition, freq)
	//		      VALUES("我","WO","wo",3,"I, me", 0);`

	//err = ref.conn.Exec(insertSql)
	//if err != nil {
	//	fmt.Println("Error while Inserting: %s", err)
	//}

	// load from caches, TODO: move to function
	if useCache {
		zhuyinFP, err := ioutil.ReadFile("zhuyin.gob")
		if err == nil {

			zhuyinBytes := bytes.NewBuffer(zhuyinFP)
			zhuyinGobObj := gob.NewDecoder(zhuyinBytes)
			err = zhuyinGobObj.Decode(ref.zhuyinCache)
			if err != nil {
				panic(err)
			}
		} else {
			fmt.Println("Error loading zhuyin.gob, continuing")
		}
		pinyinFP, err := ioutil.ReadFile("pinyin.gob")
		if err == nil {
			pinyinBytes := bytes.NewBuffer(pinyinFP)
			pinyinGobObj := gob.NewDecoder(pinyinBytes)
			err = pinyinGobObj.Decode(ref.pinyinCache)
			if err != nil {
				panic(err)
			}
		} else {
			fmt.Println("Error loading pinyin.gob, continuing")
		}
	}
	// Start the DB thread
	go ref.requestThread()
	return &ref
}
