//
// py/dataset.go is a C shared library for implementing a dataset module in Python3
//
// Authors R. S. Doiel, <rsdoiel@library.caltech.edu> and Tom Morrel, <tmorrell@library.caltech.edu>

// Copyright (c) 2018, Caltech
// All rights not granted herein are expressly reserved by Caltech.
//
// Redistribution and use in source and binary forms, with or without modification, are permitted provided that the following conditions are met:
//
// 1. Redistributions of source code must retain the above copyright notice, this list of conditions and the following disclaimer.
//
// 2. Redistributions in binary form must reproduce the above copyright notice, this list of conditions and the following disclaimer in the documentation and/or other materials provided with the distribution.
//
// 3. Neither the name of the copyright holder nor the names of its contributors may be used to endorse or promote products derived from this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
//
package main

import (
	"C"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	// Caltech Library Packages
	"github.com/caltechlibrary/dataset"
)

var verbose = false

//export verbose_on
func verbose_on() {
	verbose = true
}

//export verbose_off
func verbose_off() {
	verbose = false
}

func messagef(s string, values ...interface{}) {
	if verbose == true {
		log.Printf(s, values...)
	}
}

//export init_collection
func init_collection(name *C.char) C.int {
	collectionName := C.GoString(name)
	if verbose == true {
		messagef("creating %s\n", collectionName)
	}
	_, err := dataset.InitCollection(collectionName)
	if err != nil {
		messagef("Cannot create collection %s, %s", collectionName, err)
		return C.int(0)
	}
	messagef("%s initialized", collectionName)
	return C.int(1)
}

//export has_key
func has_key(name, key *C.char) C.int {
	collectionName := C.GoString(name)
	k := C.GoString(key)

	c, err := dataset.Open(collectionName)
	if err != nil {
		messagef("Cannot open collection %s, %s", collectionName, err)
		return C.int(0)
	}
	defer c.Close()

	if c.HasKey(k) {
		return C.int(1)
	}
	return C.int(0)
}

//export create_record
func create_record(name, key, src *C.char) C.int {
	collectionName := C.GoString(name)
	k := C.GoString(key)
	v := []byte(C.GoString(src))

	c, err := dataset.Open(collectionName)
	if err != nil {
		messagef("Cannot open collection %s, %s", collectionName, err)
		return C.int(0)
	}
	defer c.Close()

	err = c.CreateJSON(k, v)
	if err != nil {
		messagef("Create %s failed, %s", k, err)
		return C.int(0)
	}
	return C.int(1)
}

//export read_record
func read_record(name, key *C.char) *C.char {
	collectionName := C.GoString(name)
	k := C.GoString(key)

	c, err := dataset.Open(collectionName)
	if err != nil {
		messagef("Cannot open collection %s, %s", collectionName, err)
		return C.CString("")
	}
	defer c.Close()

	src, err := c.ReadJSON(k)
	if err != nil {
		messagef("Can't read %s, %s", k, err)
		return C.CString("")
	}
	txt := fmt.Sprintf("%s", src)
	return C.CString(txt)
}

//export update_record
func update_record(name, key, src *C.char) C.int {
	collectionName := C.GoString(name)
	k := C.GoString(key)
	v := []byte(C.GoString(src))

	c, err := dataset.Open(collectionName)
	if err != nil {
		messagef("Cannot open collection %s, %s", collectionName, err)
		return C.int(0)
	}
	defer c.Close()

	err = c.UpdateJSON(k, v)
	if err != nil {
		messagef("Update %s failed, %s", k, err)
		return C.int(0)
	}
	return C.int(1)
}

//export delete_record
func delete_record(name, key *C.char) C.int {
	collectionName := C.GoString(name)
	k := C.GoString(key)

	c, err := dataset.Open(collectionName)
	if err != nil {
		messagef("Cannot open collection %s, %s", collectionName, err)
		return C.int(0)
	}
	defer c.Close()

	err = c.Delete(k)
	if err != nil {
		messagef("Update %s failed, %s", k, err)
		return C.int(0)
	}
	return C.int(1)
}

//export keys
func keys(cname, cFilterExpr, cSortExpr *C.char) *C.char {
	collectionName := C.GoString(cname)
	filterExpr := C.GoString(cFilterExpr)
	sortExpr := C.GoString(cSortExpr)

	c, err := dataset.Open(collectionName)
	if err != nil {
		messagef("Cannot open collection %s, %s", collectionName, err)
		return C.CString("")
	}
	defer c.Close()

	keyList := c.Keys()
	if filterExpr != "" {
		keyList, err = c.KeyFilter(keyList, filterExpr)
		if err != nil {
			messagef("Filter error, %s", err)
			return C.CString("")
		}
	}
	if sortExpr != "" {
		keyList, err = c.KeySortByExpression(keyList, sortExpr)
		if err != nil {
			messagef("Sort error, %s", err)
			return C.CString("")
		}
	}
	src, err := json.Marshal(keyList)
	if err != nil {
		messagef("Can't marshal key list, %s", err)
		return C.CString("")
	}
	txt := fmt.Sprintf("%s", src)
	return C.CString(txt)
}

//export key_filter
func key_filter(cname, cKeyListExpr, cFilterExpr *C.char) *C.char {
	collectionName := C.GoString(cname)
	keyListExpr := C.GoString(cKeyListExpr)
	filterExpr := C.GoString(cFilterExpr)

	c, err := dataset.Open(collectionName)
	if err != nil {
		messagef("Cannot open collection %s, %s", collectionName, err)
		return C.CString("")
	}
	defer c.Close()

	keyList := []string{}
	if err := json.Unmarshal([]byte(keyListExpr), &keyList); err != nil {
		messagef("Unable to unmarshal keys", err)
		return C.CString("")
	}
	keys, err := c.KeyFilter(keyList, filterExpr)
	if err != nil {
		messagef("filter error, %s", err)
		return C.CString("")
	}
	src, err := json.Marshal(keys)
	if err != nil {
		messagef("Can't marshal filtered keys, %s", err)
		return C.CString("")
	}
	txt := fmt.Sprintf("%s", src)
	return C.CString(txt)
}

//export key_sort
func key_sort(cname, cKeyList, cSortExpr *C.char) *C.char {
	collectionName := C.GoString(cname)
	keyList := C.GoString(cKeyList)
	sortExpr := C.GoString(cSortExpr)

	c, err := dataset.Open(collectionName)
	if err != nil {
		messagef("Cannot open collection %s, %s", collectionName, err)
		return C.CString("")
	}
	defer c.Close()

	keys := []string{}
	if err := json.Unmarshal([]byte(keyList), &keys); err != nil {
		messagef("Unable to unmarshal keys", err)
		return C.CString("")
	}
	keys, err = c.KeySortByExpression(keys, sortExpr)
	if err != nil {
		messagef("filter error, %s", err)
		return C.CString("")
	}
	src, err := json.Marshal(keys)
	if err != nil {
		messagef("Can't marshal sorted keys, %s", err)
		return C.CString("")
	}
	txt := fmt.Sprintf("%s", src)
	return C.CString(txt)
}

//export count
func count(cName *C.char) C.int {
	collectionName := C.GoString(cName)
	c, err := dataset.Open(collectionName)
	if err != nil {
		messagef("Cannot open collection %s, %s", collectionName, err)
		return C.int(0)
	}
	defer c.Close()
	i := c.Length()
	return C.int(i)
}

//export extract
func extract(name, filterExpr, dotExpr *C.char) *C.char {
	collectionName := C.GoString(name)
	c, err := dataset.Open(collectionName)
	if err != nil {
		messagef("Cannot open collection %s, %s", collectionName, err)
		return C.CString("")
	}
	defer c.Close()
	values, err := c.Extract(C.GoString(filterExpr), C.GoString(dotExpr))
	if err != nil {
		messagef("Extract failed for %s, %q, %q:  %s", collectionName, filterExpr, dotExpr, err)
		return C.CString("")
	}
	src, err := json.Marshal(values)
	if err != nil {
		messagef("Can't marshal extracted values for %s, %s", collectionName, err)
		return C.CString("")
	}
	txt := fmt.Sprintf("%s", src)
	return C.CString(txt)
}

//export indexer
func indexer(cName, cIndexName, cIndexMapName, cKeyList *C.char, cBatchSize C.int) C.int {
	collectionName := C.GoString(cName)
	indexName := C.GoString(cIndexName)
	indexMapName := C.GoString(cIndexMapName)
	keyList := C.GoString(cKeyList)
	batchSize := int(cBatchSize)

	c, err := dataset.Open(collectionName)
	if err != nil {
		messagef("Cannot open collection %s, %s", collectionName, err)
		// return 0 (false)
		return C.int(0)
	}
	defer c.Close()

	keys := []string{}
	if keyList != "" {
		err = json.Unmarshal([]byte(keyList), &keys)
		if err != nil {
			messagef("Can't unmarshal key list, %s", err)
			// return 0 (false)
			return C.int(0)
		}
	}

	err = c.Indexer(indexName, indexMapName, keys, batchSize)
	if err != nil {
		messagef("Indexing error %s %s, %s", collectionName, indexName, err)
		// return 0 (false)
		return C.int(0)
	}
	// return 1 (true) for success
	return C.int(1)
}

//export deindexer
func deindexer(cName, cIndexName, cKeyList *C.char, cBatchSize C.int) C.int {
	collectionName := C.GoString(cName)
	indexName := C.GoString(cIndexName)
	keyList := C.GoString(cKeyList)
	batchSize := int(cBatchSize)

	c, err := dataset.Open(collectionName)
	if err != nil {
		messagef("Cannot open collection %s, %s", collectionName, err)
		// return 0 (false), failed
		return C.int(0)
	}
	defer c.Close()

	keys := []string{}
	if keyList != "" {
		err = json.Unmarshal([]byte(keyList), &keys)
		if err != nil {
			messagef("Can't unmarshal key list, %s", err)
			// return 0 (false), failed
			return C.int(0)
		}
	}

	err = c.Deindexer(indexName, keys, batchSize)
	if err != nil {
		messagef("Deindexing error %s %s, %s", collectionName, indexName, err)
		// return 0 (false), failed
		return C.int(0)
	}
	// return 1 (true) for success
	return C.int(1)
}

//export find
func find(cIndexNames, cQueryString, cOptionsMap *C.char) *C.char {
	indexNamesSrc := C.GoString(cIndexNames)
	queryString := C.GoString(cQueryString)
	optionsSrc := C.GoString(cOptionsMap)

	indexNames := []string{}
	if strings.HasPrefix(indexNamesSrc, "[") {
		err := json.Unmarshal([]byte(indexNamesSrc), &indexNames)
		if err != nil {
			messagef("Can't unmarshal index names, %s", err)
			return C.CString("")
		}
	} else if strings.Contains(indexNamesSrc, ":") {
		indexNames = strings.Split(indexNamesSrc, ":")
	} else {
		indexNames = []string{indexNamesSrc}
	}
	options := map[string]string{}
	if optionsSrc != "" {
		err := json.Unmarshal([]byte(optionsSrc), &options)
		if err != nil {
			messagef("Options error, %s", err)
			// return "", failed
			return C.CString("")
		}
	}

	idxList, _, err := dataset.OpenIndexes(indexNames)
	if err != nil {
		messagef("Can't open index %s, %s", strings.Join(indexNames, ", "), err)
		return C.CString("")
	}

	result, err := dataset.Find(idxList.Alias, strings.Split(queryString, "\n"), options)
	if err != nil {
		messagef("Find error %s, %s", strings.Join(indexNames, ", "), err)
		// return "", failed
		return C.CString("")
	}
	err = idxList.Close()
	if err != nil {
		messagef("Can't close indexes %s, %s", strings.Join(indexNames, ", "), err)
	}

	src, err := json.Marshal(result)
	if err != nil {
		messagef("Can't marshal results, %s", err)
		// return "", failed
		return C.CString("")
	}

	txt := fmt.Sprintf("%s", src)
	// return our encoded results, success
	return C.CString(txt)
}

func main() {}