package main

import (
	"bytes"
	"code.google.com/p/gosqlite/sqlite"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
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
	CharList   []Character
	NumResults int
}

// ReferenceStore is an object that serves as an in-memory cache for the DB,
// holds the handle for the DB connection, and holds the request queue channel
// for character lookup by the DB thread
type ReferenceStore struct {
	conn         *sqlite.Conn
	requestQueue chan *CharLookupRequest
	GlobalCache  map[string]*CharLookupResponse
}

// GetByChar retrieves full candidate characters, given a UTF-8 Chinese character
func (ref ReferenceStore) GetByChar(char string) (*[]Character, int) {
	writeBack := make(chan *CharLookupResponse)
	char = strings.TrimSpace(char)
	queryInfo := Character{-1, char, "", "", -1, "", -1}
	ref.requestQueue <- &CharLookupRequest{queryInfo, writeBack}
	response := <-writeBack
	return &response.CharList, response.NumResults
}

// GetByZhuyin retrieves full candidate characters, given a UTF-8 zhuyin string
func (ref ReferenceStore) GetByZhuyin(zhuyin string) (*[]Character, int) {
	zhuyin, tone := ref.SeparatePhonetic(zhuyin)
	writeBack := make(chan *CharLookupResponse)
	zhuyin = strings.TrimSpace(zhuyin)
	// take last character and see if number. If so, it's the tone
	queryInfo := Character{-1, "", zhuyin, "", tone, "", -1}
	ref.requestQueue <- &CharLookupRequest{queryInfo, writeBack}
	response := <-writeBack
	return &response.CharList, response.NumResults
}

// GetByZhuyin retrieves full candidate characters, given a pinyin string
func (ref ReferenceStore) GetByPinyin(pinyin string) (*[]Character, int) {
	pinyin, tone := ref.SeparatePhonetic(pinyin)
	writeBack := make(chan *CharLookupResponse)
	pinyin = strings.TrimSpace(pinyin)
	queryInfo := Character{-1, "", "", pinyin, tone, "", -1}
	ref.requestQueue <- &CharLookupRequest{queryInfo, writeBack}
	response := <-writeBack
	return &response.CharList, response.NumResults
}

// GetByDefinition retreives full candidate characters, given an English definition
func (ref ReferenceStore) GetByDefinition(definition string) (*[]Character, int) {
	writeBack := make(chan *CharLookupResponse)
	definition = strings.TrimSpace(definition)
	queryInfo := Character{-1, "", "", "", -1, definition, -1}
	ref.requestQueue <- &CharLookupRequest{queryInfo, writeBack}
	response := <-writeBack
	return &response.CharList, response.NumResults
}

// GetToneFromPhonetic extracts the numerical tone from pinyin/zhuyin
// Thus, this doesn't work with accented text or with encodings that have
// characters within the range of ascii numbers
func (ref ReferenceStore) SeparatePhonetic(input string) (string, int) {
	// Match anything that is not 0-9 up to a maximum of 12 characters
	// (3 UTF-8 Zhuyin characters = 3 x 4 bytes = 12 chars)
	// next, match only one number following that string.
	tonePattern, _ := regexp.Compile("^[^0-9]{1,12}[0-6{1,1}]$")
	//noTonePattern, _ := regexp.Compile("^[^0-9]{1,12}$")

	if tonePattern.MatchString(input) {
		num, _ := strconv.Atoi(input[len(input)-1 : len(input)])
		return input[:len(input)-1], num
	}
	return input, -1
}

// Get is the base lookup function called only by the DB thread
func (ref ReferenceStore) Get(partialChar Character) *CharLookupResponse {
	var toneString string

	if partialChar.Tone == -1 {
		toneString = ""
	} else {
		toneString = strconv.Itoa(partialChar.Tone)
	}

	// first, check the cache
	if val, ok := ref.GlobalCache[partialChar.Zhuyin+toneString]; ok {
		return val
	}
	if val, ok := ref.GlobalCache[partialChar.Pinyin+toneString]; ok {
		return val
	}

	searchStmt, _ := ref.conn.Prepare(`SELECT * FROM characters WHERE
						character LIKE ? AND
						zhuyin LIKE ? AND
						pinyin LIKE ? AND
						tone LIKE ? AND
						definition LIKE ?
						ORDER BY freq DESC LIMIT 50`)

	err := searchStmt.Exec(
		"%"+partialChar.Character+"%",
		"%"+partialChar.Zhuyin+"%",
		"%"+partialChar.Pinyin+"%",
		"%"+toneString+"%",
		"%"+partialChar.Definition+"%")
	if err != nil {
		fmt.Println("Error while Selecting: %s", err)
	}

	var charList []Character
	for searchStmt.Next() {
		var resultChar Character
		err = searchStmt.Scan(&resultChar.Id,
			&resultChar.Character,
			&resultChar.Zhuyin,
			&resultChar.Pinyin,
			&resultChar.Tone,
			&resultChar.Definition,
			&resultChar.Freq)
		if err != nil {
			fmt.Printf("Error while getting row data: %s\n", err)
			os.Exit(1)
		}

		fmt.Printf("Id => %d\n", resultChar.Id)
		fmt.Printf("Title => %s\n", resultChar.Character)
		charList = append(charList, resultChar)
	}

	var response = &CharLookupResponse{charList, len(charList)}

	// Cache results, if there are any
	if len(charList) > 0 {
		var firstChar Character
		// wtf Go syntax- convert value of linked list element to struct type Character
		firstChar = charList[0]

		// cache full result in zhuyin and pinyin caches
		if firstChar.Pinyin != "" {
			ref.GlobalCache[firstChar.Pinyin+strconv.Itoa(firstChar.Tone)] = response
		}
		if firstChar.Zhuyin != "" {
			ref.GlobalCache[firstChar.Zhuyin+strconv.Itoa(firstChar.Tone)] = response
		}
	}
	return response
}

// requestThread is the "DB thread", an internal running goroutine
// that handles lookup requests coming into the requestQueue channel
func (ref ReferenceStore) requestThread() {
	for request := range ref.requestQueue {
		request.WriteBack <- ref.Get(request.Char)
	}
}

// Cleans up chan and shuts down, saving the cache
func (ref ReferenceStore) Close() {
	close(ref.requestQueue)
	// write cache to file
	buffer := new(bytes.Buffer)
	enc := gob.NewEncoder(buffer)
	// This is wrong.
	// map -> *CharLookupResponse -> *Character
	err := enc.Encode(&ref)
	if err != nil {
		panic(err)
	}
	err = ioutil.WriteFile("globalCache.gob", buffer.Bytes(), 0600)
	if err != nil {
		panic(err)
	}
}

// NewReference initializes the database and returns a Reference object
func NewReference(dbName string, useCache bool) *ReferenceStore {
	ref := ReferenceStore{nil, make(chan *CharLookupRequest), make(map[string]*CharLookupResponse)}
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
	//fmt.Println("Error while Inserting: %s", err)
	//}

	// load from caches, TODO: move to function, also doesn't work
	if useCache {

		// clean up cache and build new objects
		cacheFP, err := ioutil.ReadFile("globalCache.gob")
		if err == nil {

			cacheBytes := bytes.NewBuffer(cacheFP)
			cacheGobObj := gob.NewDecoder(cacheBytes)
			err = cacheGobObj.Decode(&ref)
			if err != nil {
				panic(err)
			}
		} else {
			fmt.Println("Error loading globalCache.gob, continuing")
		}
	}
	// Start the DB thread
	go ref.requestThread()
	return &ref
}
