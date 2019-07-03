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
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	// Caltech Library Packages
	"github.com/caltechlibrary/dataset"
	"github.com/caltechlibrary/dataset/gsheets"
	"github.com/caltechlibrary/dataset/tbl"
)

var (
	verbose          = false
	useStrictDotpath = true
	// NOTE: error state is shared because C doesn't easily pass multiple
	// return values without resorting to complex structures.
	errorValue error
)

// error_clear will set the global error state to nil.
//
//export error_clear
func error_clear() {
	errorValue = nil
}

// error_dispatch logs error messages to console based on string template
// Not exported.
//
func error_dispatch(err error, s string, values ...interface{}) {
	errorValue = err
	if verbose == true {
		log.Printf(s, values...)
	}
}

// error_message returns an error message previously recorded or
// an empty string if no errors recorded
//
//export error_message
func error_message() *C.char {
	if errorValue != nil {
		s := fmt.Sprintf("%s", errorValue)
		errorValue = nil
		return C.CString(s)
	}
	return C.CString("")
}

// use_strict_dotpath sets the library option value for
// enforcing strict dotpaths. 1 is true, any other value is false.
//
//export use_strict_dotpath
func use_strict_dotpath(v C.int) C.int {
	if int(v) == 1 {
		useStrictDotpath = true
		return C.int(1)
	}
	useStrictDotpath = false
	return C.int(0)
}

// is_verbose returns the library options' verbose value.
//
//export is_verbose
func is_verbose() C.int {
	if verbose == true {
		return C.int(1)
	}
	return C.int(0)
}

// verbose_on set library verbose to true
//
//export verbose_on
func verbose_on() {
	verbose = true
}

// verbose_off set library verbose to false
//
//export verbose_off
func verbose_off() {
	verbose = false
}

// messagef is an intertal library function for logging messages to
//
// the console. Not exported.
func messagef(s string, values ...interface{}) {
	if verbose == true {
		log.Printf(s, values...)
	}
}

// dataset_version returns the dataset version the libdataset presents.
//
//export dataset_version
func dataset_version() *C.char {
	return C.CString(dataset.Version)
}

// init_collection intializes a collection and records as much metadata
// as it can from the execution environment (e.g. username,
// datetime created)
//
//export init_collection
func init_collection(name *C.char) C.int {
	collectionName := C.GoString(name)
	if verbose == true {
		messagef("creating %s\n", collectionName)
	}
	error_clear()
	_, err := dataset.InitCollection(collectionName)
	if err != nil {
		error_dispatch(err, "Cannot create collection %s, %s", collectionName, err)
		return C.int(0)
	}
	messagef("%s initialized", collectionName)
	return C.int(1)
}

// has_key returns 1 if the key exists in a collection or 0 if not.
//
//export has_key
func has_key(name, key *C.char) C.int {
	collectionName := C.GoString(name)
	k := C.GoString(key)

	error_clear()
	c, err := dataset.Open(collectionName)
	if err != nil {
		error_dispatch(err, "Cannot open collection %s, %s", collectionName, err)
		return C.int(0)
	}
	defer c.Close()

	if c.HasKey(k) {
		return C.int(1)
	}
	return C.int(0)
}

// create_record takes JSON source and adds it to the collection with
// the provided key.
//
//export create_record
func create_record(name, key, src *C.char) C.int {
	collectionName := C.GoString(name)
	k := C.GoString(key)
	v := []byte(C.GoString(src))

	error_clear()
	c, err := dataset.Open(collectionName)
	if err != nil {
		error_dispatch(err, "Cannot open collection %s, %s", collectionName, err)
		return C.int(0)
	}
	defer c.Close()

	err = c.CreateJSON(k, v)
	if err != nil {
		error_dispatch(err, "Create %s failed, %s", k, err)
		return C.int(0)
	}
	return C.int(1)
}

// read_record takes a key and returns JSON source of the record
//
//export read_record
func read_record(name, key *C.char, clean_object C.int) *C.char {
	collectionName := C.GoString(name)
	k := C.GoString(key)
	cleanObject := (C.int(1) == clean_object)

	error_clear()
	c, err := dataset.Open(collectionName)
	if err != nil {
		error_dispatch(err, "Cannot open collection %s, %s", collectionName, err)
		return C.CString("")
	}
	defer c.Close()

	var (
		src []byte
	)

	m := map[string]interface{}{}
	if err = c.Read(k, m, cleanObject); err != nil {
		error_dispatch(err, "Can't read %s, %s", k, err)
		return C.CString("")
	}
	if src, err = json.Marshal(m); err != nil {
		error_dispatch(err, "Can't read (marshal) %s, %s", k, err)
		return C.CString("")
	}
	txt := fmt.Sprintf("%s", src)
	return C.CString(txt)
}

// THIS IS AN UGLY HACK, Python ctypes doesn't **easily** support
// undemensioned arrays of strings. So we will assume the array of
// keys has already been transformed into JSON before calling
// read_list.
//
//export read_record_list
func read_record_list(name *C.char, keys_as_json *C.char, clean_object C.int) *C.char {
	collectionName := C.GoString(name)
	cleanObject := C.int(1) == clean_object
	l := []string{}
	errList := []string{}

	error_clear()
	c, err := dataset.Open(collectionName)
	if err != nil {
		error_dispatch(err, "Cannot open collection %s, %s", collectionName, err)
		return C.CString("")
	}
	defer c.Close()

	// Now unpack our keys into an array of strings.
	src := []byte(C.GoString(keys_as_json))
	key_list := []string{}
	err = json.Unmarshal(src, &key_list)
	if err != nil {
		error_dispatch(err, "Can't unmarshal key list, %s", err)
		return C.CString("")
	}

	for _, key := range key_list {
		//NOTE: we want to use c.Read() if cleanObject is true
		if cleanObject == true {
			m := map[string]interface{}{}
			if err = c.Read(key, m, cleanObject); err == nil {
				errList = append(errList, fmt.Sprintf("(%s) %s", key, err))
			} else {
				if src, err := json.Marshal(m); err == nil {
					l = append(l, fmt.Sprintf("%s", src))
				} else {
					errList = append(errList, fmt.Sprintf("(%s) %s", key, err))
				}
			}
		} else {
			src, err := c.ReadJSON(key)
			if err != nil {
				errList = append(errList, fmt.Sprintf("(%s) %s", key, err))
			} else {
				l = append(l, fmt.Sprintf("%s", src))
			}
		}
	}
	if len(errList) > 0 {
		err = fmt.Errorf("%s", strings.Join(errList, "; "))
		error_dispatch(err, "Key read errors %s", err)
	}

	txt := fmt.Sprintf("[%s]", strings.Join(l, ","))
	return C.CString(txt)
}

// update_record takes a key and JSON source and replaces the record
// in the collection.
//
//export update_record
func update_record(name, key, src *C.char) C.int {
	collectionName := C.GoString(name)
	k := C.GoString(key)
	v := []byte(C.GoString(src))

	error_clear()
	c, err := dataset.Open(collectionName)
	if err != nil {
		error_dispatch(err, "Cannot open collection %s, %s", collectionName, err)
		return C.int(0)
	}
	defer c.Close()

	err = c.UpdateJSON(k, v)
	if err != nil {
		error_dispatch(err, "Update %s failed, %s", k, err)
		return C.int(0)
	}
	return C.int(1)
}

// delete_record takes a key and removes a record from the collection
//
//export delete_record
func delete_record(name, key *C.char) C.int {
	collectionName := C.GoString(name)
	k := C.GoString(key)

	error_clear()
	c, err := dataset.Open(collectionName)
	if err != nil {
		error_dispatch(err, "Cannot open collection %s, %s", collectionName, err)
		return C.int(0)
	}
	defer c.Close()

	err = c.Delete(k)
	if err != nil {
		error_dispatch(err, "Update %s failed, %s", k, err)
		return C.int(0)
	}
	return C.int(1)
}

// join takes a collection name, a key, and merges JSON source with an
// existing JSON record. If overwrite is 1 it overwrites and replaces
// common values, if not 1 it only adds missing attributes.
//
//export join
func join(cName *C.char, cKey *C.char, cObjSrc *C.char, cOverwrite C.int) C.int {
	collectionName := C.GoString(cName)
	key := C.GoString(cKey)
	overwrite := (cOverwrite == 1)
	objectSrc := C.GoString(cObjSrc)

	error_clear()
	c, err := dataset.Open(collectionName)
	if err != nil {
		error_dispatch(err, "%s", err)
		return C.int(0)
	}
	defer c.Close()

	outObject := map[string]interface{}{}
	newObject := map[string]interface{}{}

	if err := c.Read(key, outObject, false); err != nil {
		error_dispatch(err, "%s", err)
		return C.int(0)
	}

	if err := json.Unmarshal([]byte(objectSrc), &newObject); err != nil {
		error_dispatch(err, "%s", err)
		return C.int(0)
	}
	if overwrite {
		for k, v := range newObject {
			outObject[k] = v
		}
	} else {
		for k, v := range newObject {
			if _, ok := outObject[k]; ok != true {
				outObject[k] = v
			}
		}
	}
	if err := c.Update(key, outObject); err != nil {
		error_dispatch(err, "%s", err)
		return C.int(0)
	}
	return C.int(1)
}

// keys returns JSON source of an array of keys from the collection
//
//export keys
func keys(cname, cFilterExpr, cSortExpr *C.char) *C.char {
	collectionName := C.GoString(cname)
	filterExpr := C.GoString(cFilterExpr)
	sortExpr := C.GoString(cSortExpr)

	error_clear()
	c, err := dataset.Open(collectionName)
	if err != nil {
		error_dispatch(err, "Cannot open collection %s, %s", collectionName, err)
		return C.CString("")
	}
	defer c.Close()

	keyList := c.Keys()
	if filterExpr != "" {
		keyList, err = c.KeyFilter(keyList, filterExpr)
		if err != nil {
			error_dispatch(err, "Filter error, %s", err)
			return C.CString("")
		}
	}
	if sortExpr != "" {
		keyList, err = c.KeySortByExpression(keyList, sortExpr)
		if err != nil {
			error_dispatch(err, "Sort error, %s", err)
			return C.CString("")
		}
	}
	src, err := json.Marshal(keyList)
	if err != nil {
		error_dispatch(err, "Can't marshal key list, %s", err)
		return C.CString("")
	}
	txt := fmt.Sprintf("%s", src)
	return C.CString(txt)
}

// key_filter returns JSON source of an array of keys passing
// through the filter of objects in the collection.
//
//export key_filter
func key_filter(cname, cKeyListExpr, cFilterExpr *C.char) *C.char {
	collectionName := C.GoString(cname)
	keyListExpr := C.GoString(cKeyListExpr)
	filterExpr := C.GoString(cFilterExpr)

	error_clear()
	c, err := dataset.Open(collectionName)
	if err != nil {
		error_dispatch(err, "Cannot open collection %s, %s", collectionName, err)
		return C.CString("")
	}
	defer c.Close()

	keyList := []string{}
	if err := json.Unmarshal([]byte(keyListExpr), &keyList); err != nil {
		error_dispatch(err, "Unable to unmarshal keys", err)
		return C.CString("")
	}
	keys, err := c.KeyFilter(keyList, filterExpr)
	if err != nil {
		error_dispatch(err, "filter error, %s", err)
		return C.CString("")
	}
	src, err := json.Marshal(keys)
	if err != nil {
		error_dispatch(err, "Can't marshal filtered keys, %s", err)
		return C.CString("")
	}
	txt := fmt.Sprintf("%s", src)
	return C.CString(txt)
}

// key_sort returns JSON source of an array of keys sorted by
// the sort expression applied to the objects in the collection.
//
//export key_sort
func key_sort(cname, cKeyList, cSortExpr *C.char) *C.char {
	collectionName := C.GoString(cname)
	keyList := C.GoString(cKeyList)
	sortExpr := C.GoString(cSortExpr)

	error_clear()
	c, err := dataset.Open(collectionName)
	if err != nil {
		error_dispatch(err, "Cannot open collection %s, %s", collectionName, err)
		return C.CString("")
	}
	defer c.Close()

	keys := []string{}
	if err := json.Unmarshal([]byte(keyList), &keys); err != nil {
		error_dispatch(err, "Unable to unmarshal keys", err)
		return C.CString("")
	}
	keys, err = c.KeySortByExpression(keys, sortExpr)
	if err != nil {
		error_dispatch(err, "filter error, %s", err)
		return C.CString("")
	}
	src, err := json.Marshal(keys)
	if err != nil {
		error_dispatch(err, "Can't marshal sorted keys, %s", err)
		return C.CString("")
	}
	txt := fmt.Sprintf("%s", src)
	return C.CString(txt)
}

// count returns the number of objects (records) in a collection.
//
//export count
func count(cName *C.char) C.int {
	collectionName := C.GoString(cName)
	error_clear()
	c, err := dataset.Open(collectionName)
	if err != nil {
		error_dispatch(err, "Cannot open collection %s, %s", collectionName, err)
		return C.int(0)
	}
	defer c.Close()
	i := c.Length()
	return C.int(i)
}

// import_csv - import a CSV file into a collection
// syntax: COLLECTION CSV_FILENAME ID_COL
//
// options that should support sensible defaults:
//
//     cUseHeaderRow
//     cOverwrite
//
//export import_csv
func import_csv(cName *C.char, cCSVFName *C.char, cIDCol C.int, cUseHeaderRow C.int, cOverwrite C.int) C.int {
	// Covert options
	collectionName := C.GoString(cName)
	csvFName := C.GoString(cCSVFName)
	idCol := int(cIDCol)
	useHeaderRow := (int(cUseHeaderRow) == 1)
	overwrite := (int(cOverwrite) == 1)

	error_clear()

	c, err := dataset.Open(collectionName)
	if err != nil {
		error_dispatch(err, "Can't open %s, %s", collectionName, err)
		return C.int(0)
	}
	defer c.Close()

	if idCol < 1 {
		error_dispatch(fmt.Errorf("invalid column number"), "Column number must be greater than zero, got %s", idCol)
		return C.int(0)
	}

	// NOTE: we need to adjust to zero based index
	idCol--

	// Now import our CSV file
	fp, err := os.Open(csvFName)
	if err != nil {
		error_dispatch(err, "Can't open %s, %s", csvFName, err)
		return C.int(0)
	}
	cnt, err := c.ImportCSV(fp, idCol, useHeaderRow, overwrite, verbose)
	if err != nil {
		error_dispatch(err, "%s\n", err)
		return C.int(0)
	}
	messagef("%d total rows processed", cnt)

	return C.int(1)
}

// export_csv - export collection objects to a CSV file
// syntax: COLLECTION FRAME CSV_FILENAME
//
//export export_csv
func export_csv(cName *C.char, cFrameName *C.char, cCSVFName *C.char) C.int {
	// Convert out parameters
	collectionName := C.GoString(cName)
	frameName := C.GoString(cFrameName)
	csvFName := C.GoString(cCSVFName)

	error_clear()
	c, err := dataset.Open(collectionName)
	if err != nil {
		error_dispatch(err, "%s", err)
		return C.int(0)
	}
	defer c.Close()

	fp, err := os.Create(csvFName)
	if err != nil {
		error_dispatch(err, "Can't create %s, %s", csvFName, err)
		return C.int(0)
	}
	defer fp.Close()

	// Get Frame
	if c.HasFrame(frameName) == false {
		error_dispatch(err, "Missing frame %q in %s\n", frameName, collectionName)
		return C.int(0)
	}

	// Get dotpaths and column labels from frame
	f, err := c.Frame(frameName, nil, nil, nil, verbose)
	if err != nil {
		error_dispatch(err, "%s\n", err)
		return C.int(0)
	}

	if f.FilterExpr == "" {
		f.FilterExpr = "true"
	}

	// Now export to CSV
	cnt, err := c.ExportCSV(fp, os.Stderr, f, verbose)
	if err != nil {
		error_dispatch(err, "Can't export CSV %s, %s", csvFName, err)
		return C.int(0)
	}
	messagef("%d total rows processed", cnt)
	return C.int(1)
}

// import_gsheet - import a GSheet into a collection
// syntax: COLLECTION GSHEET_ID SHEET_NAME ID_COL CELL_RANGE
//
// options that should support sensible defaults:
//
//    cUseHeaderRow
//    cOverwrite
//
//export import_gsheet
func import_gsheet(cName *C.char, cSheetID *C.char, cSheetName *C.char, cIDCol C.int, cCellRange *C.char, cUseHeaderRow C.int, cOverwrite C.int) C.int {
	collectionName := C.GoString(cName)
	sheetID := C.GoString(cSheetID)
	sheetName := C.GoString(cSheetName)
	cellRange := C.GoString(cCellRange)
	idCol := int(cIDCol)
	useHeaderRow := (C.int(cUseHeaderRow) == 1)
	overwrite := (C.int(cOverwrite) == 1)

	error_clear()
	c, err := dataset.Open(collectionName)
	if err != nil {
		error_dispatch(err, "%s", err)
		return C.int(0)
	}
	defer c.Close()

	// NOTE: we need to adjust to zero based index
	idCol--

	//FIXME: Need better search process for finding the google access key
	clientSecretJSON := os.Getenv("GOOGLE_CLIENT_SECRET_JSON")
	if clientSecretJSON == "" {
		//clientSecretJSON = "client_secret.json"
		clientSecretJSON = "credentials.json"
	}

	table, err := gsheets.ReadSheet(clientSecretJSON, sheetID, sheetName, cellRange)
	if err != nil {
		error_dispatch(err, "%s", err)
		return C.int(0)
	}

	cnt, err := c.ImportTable(table, idCol, useHeaderRow, overwrite, verbose)
	if err != nil {
		error_dispatch(err, "Errors importing %s %s, %s", sheetID, sheetName, err)
		return C.int(0)
	}
	messagef("%d total rows processed", cnt)
	return C.int(1)
}

// export_gsheet - export collection objects to a GSheet
// syntax: COLLECTION FRAME GSHEET_ID GSHEET_NAME CELL_RANGE
//
//export export_gsheet
func export_gsheet(cName *C.char, cFrameName *C.char, cSheetID *C.char, cSheetName *C.char, cCellRange *C.char) C.int {
	collectionName := C.GoString(cName)
	frameName := C.GoString(cFrameName)
	sheetID := C.GoString(cSheetID)
	sheetName := C.GoString(cSheetName)
	cellRange := C.GoString(cCellRange)

	error_clear()
	c, err := dataset.Open(collectionName)
	if err != nil {
		error_dispatch(err, "failed, %s %s, %s", sheetID, sheetName, err)
		return C.int(0)
	}
	defer c.Close()

	// Get Frame
	if c.HasFrame(frameName) == false {
		error_dispatch(err, "Missing frame %q in %s\n", frameName, collectionName)
		return C.int(0)
	}

	// Get dotpaths and column labels from frame
	f, err := c.Frame(frameName, nil, nil, nil, verbose)
	if err != nil {
		error_dispatch(err, "%s\n", err)
		return C.int(0)
	}

	if f.FilterExpr == "" {
		f.FilterExpr = "true"
	}

	//FIXME: Need a better way to indentify the clientSecretName...
	clientSecretJSON := os.Getenv("GOOGLE_CLIENT_SECRET_JSON")
	if clientSecretJSON == "" {
		//clientSecretJSON = "client_secret.json"
		clientSecretJSON = "credentials.json"
	}
	// gSheet expects a cell range, so we will generate one if needed.
	if cellRange == "" {
		lastCol := gsheets.ColNoToColLetters(len(f.Labels))
		lastRow := len(f.Keys) + 2
		cellRange = fmt.Sprintf("A1:%s%d", lastCol, lastRow)
	}

	//NOTE: we export to GSheet via creating a table [][]interface{}{}
	cnt, table, err := c.ExportTable(os.Stderr, f, verbose)
	if err != nil {
		error_dispatch(err, "%s\n", err)
		return C.int(0)
	}
	err = gsheets.WriteSheet(clientSecretJSON, sheetID, sheetName, cellRange, table)

	if err != nil {
		error_dispatch(err, "Failed to write %s %s, %s", sheetID, sheetName, err)
		return C.int(0)
	}
	messagef("%d total rows processed", cnt)
	return C.int(1)
}

// status checks to see if a collection exists or not.
//
//export status
func status(cName *C.char) C.int {
	collectionName := C.GoString(cName)
	error_clear()
	c, err := dataset.Open(collectionName)
	if err != nil {
		error_dispatch(err, "failed, %s, %s", collectionName, err)
		return C.int(0)
	}
	c.Close()
	return C.int(1)
}

// list returns JSON array of objects in a collections based on a
// JSON array of keys.
//
//export list
func list(cName *C.char, cKeys *C.char) *C.char {
	collectionName := C.GoString(cName)
	sKeys := C.GoString(cKeys)

	error_clear()
	c, err := dataset.Open(collectionName)
	if err != nil {
		error_dispatch(err, "%s", err)
		return C.CString("")
	}
	defer c.Close()

	keys := []string{}
	err = json.Unmarshal([]byte(sKeys), &keys)
	if err != nil {
		error_dispatch(err, "Failed to unmarshal key list, %s", err)
		return C.CString("")
	}

	recs := []map[string]interface{}{}
	for _, name := range keys {
		m := map[string]interface{}{}
		err = c.Read(name, m, false)
		if err != nil {
			error_dispatch(err, "%s", err)
			return C.CString("")
		}
		recs = append(recs, m)
	}
	src, err := json.Marshal(recs)
	if err != nil {
		error_dispatch(err, "failed to marshal result, %s", err)
		return C.CString("")
	}
	return C.CString(string(src))
}

// path returns the path on disc to an JSON object document
// in the collection.
//
//export path
func path(cName *C.char, cKey *C.char) *C.char {
	collectionName := C.GoString(cName)
	key := C.GoString(cKey)

	error_clear()
	c, err := dataset.Open(collectionName)
	if err != nil {
		error_dispatch(err, "%s", err)
		return C.CString("")
	}
	defer c.Close()
	s, err := c.DocPath(key)
	if err != nil {
		error_dispatch(err, "%s", err)
		return C.CString("")
	}
	return C.CString(s)
}

// check runs the analyzer over a collection and looks for
// problem records.
//
//export check
func check(cName *C.char) C.int {
	collectionName := C.GoString(cName)
	err := dataset.Analyzer(collectionName)
	if err != nil {
		error_dispatch(err, "%s", err)
		return C.int(0)
	}
	return C.int(1)
}

// repair runs the analyzer over a collection and repairs JSON
// objects and attachment discovered having a problem. Also is
// useful for upgrading a collection between dataset releases.
//
//export repair
func repair(cName *C.char) C.int {
	collectionName := C.GoString(cName)
	err := dataset.Repair(collectionName)
	if err != nil {
		error_dispatch(err, "%s", err)
		return C.int(0)
	}
	return C.int(1)
}

// attach will attach a file to a JSON object in a collection. It takes
// a semver string (e.g. v0.0.1) and associates that with where it stores
// the file.  If semver is v0.0.0 it is considered unversioned, if v0.0.1
// or larger it is considered versioned.
//
//export attach
func attach(cName *C.char, cKey *C.char, cSemver *C.char, cFNames *C.char) C.int {
	collectionName := C.GoString(cName)
	key := C.GoString(cKey)
	semver := C.GoString(cSemver)
	if semver == "" {
		semver = "v0.0.0"
	}
	srcFNames := C.GoString(cFNames)
	fNames := []string{}
	if len(srcFNames) > 0 {
		err := json.Unmarshal([]byte(srcFNames), &fNames)
		if err != nil {
			error_dispatch(err, "Can't unmarshal %q, %s", srcFNames, err)
			return C.int(0)
		}
	}

	error_clear()
	c, err := dataset.Open(collectionName)
	if err != nil {
		error_dispatch(err, "%s", err)
		return C.int(0)
	}
	defer c.Close()

	if c.HasKey(key) == false {
		error_dispatch(fmt.Errorf("missing key"), "%q is not in collection", key)
		return C.int(0)
	}
	for _, fname := range fNames {
		if _, err := os.Stat(fname); os.IsNotExist(err) {
			error_dispatch(err, "%s does not exist", fname)
			return C.int(0)
		}
	}
	err = c.AttachFiles(key, semver, fNames...)
	if err != nil {
		error_dispatch(err, "%s", err)
		return C.int(0)
	}
	return C.int(1)
}

// attachments returns a list of attachments and their size in
// associated with a JSON obejct in the collection.
//
//export attachments
func attachments(cName *C.char, cKey *C.char) *C.char {
	collectionName := C.GoString(cName)
	key := C.GoString(cKey)
	error_clear()
	c, err := dataset.Open(collectionName)
	if err != nil {
		error_dispatch(err, "%s", err)
		return C.CString("")
	}
	defer c.Close()
	if c.HasKey(key) == false {
		error_dispatch(fmt.Errorf("missing key"), "%q is not in collection", key)
		return C.CString("")
	}
	results, err := c.Attachments(key)
	if err != nil {
		error_dispatch(err, "%s", err)
		return C.CString("")
	}
	if len(results) > 0 {
		return C.CString(strings.Join(results, "\n"))
	}
	return C.CString("")
}

// detach exports the file associated with the semver from the JSON
// object in the collection. The file remains "attached".
//
//export detach
func detach(cName *C.char, cKey *C.char, cSemver *C.char, cFNames *C.char) C.int {
	collectionName := C.GoString(cName)
	key := C.GoString(cKey)
	semver := C.GoString(cSemver)
	if semver == "" {
		semver = "v0.0.0"
	}
	srcFNames := C.GoString(cFNames)
	fNames := []string{}
	if len(srcFNames) > 0 {
		err := json.Unmarshal([]byte(srcFNames), &fNames)
		if err != nil {
			error_dispatch(err, "Can't unmarshal filename list, %s", err)
			return C.int(0)
		}
	}
	error_clear()
	c, err := dataset.Open(collectionName)
	if err != nil {
		error_dispatch(err, "%s", err)
		return C.int(0)
	}
	defer c.Close()
	if c.HasKey(key) == false {
		error_dispatch(err, "%q is not in collection", key)
		return C.int(0)
	}
	err = c.GetAttachedFiles(key, semver, fNames...)
	if err != nil {
		error_dispatch(err, "%s", err)
		return C.int(0)
	}
	return C.int(1)
}

// prune removes an attachment by semver from a JSON object in the
// collection. This is destructive, the file is removed from disc.
//
//export prune
func prune(cName *C.char, cKey *C.char, cSemver *C.char, cFNames *C.char) C.int {
	collectionName := C.GoString(cName)
	key := C.GoString(cKey)
	semver := C.GoString(cSemver)
	srcFNames := C.GoString(cFNames)
	fNames := []string{}
	if len(srcFNames) > 0 {
		err := json.Unmarshal([]byte(srcFNames), &fNames)
		if err != nil {
			error_dispatch(err, "Can't unmarshal filename list, %s", err)
			return C.int(0)
		}
	}
	error_clear()
	c, err := dataset.Open(collectionName)
	if err != nil {
		error_dispatch(err, "%s", err)
		return C.int(0)
	}
	defer c.Close()

	err = c.Prune(key, semver, fNames...)
	if err != nil {
		error_dispatch(err, "%s", err)
		return C.int(0)
	}
	return C.int(1)
}

// clone takes a collection name, a JSON array of keys and creates
// a new collection with a new name based on the origin's collections'
// objects.
//
//export clone
func clone(cName *C.char, cKeys *C.char, dName *C.char) C.int {
	collectionName := C.GoString(cName)
	srcKeys := C.GoString(cKeys)
	destName := C.GoString(dName)
	error_clear()
	c, err := dataset.Open(collectionName)
	if err != nil {
		error_dispatch(err, "%s", err)
		return C.int(0)
	}
	defer c.Close()
	keys := []string{}
	err = json.Unmarshal([]byte(srcKeys), &keys)
	if err != nil {
		error_dispatch(err, "Can't unmarshal keys, %s", err)
		return C.int(0)
	}
	err = c.Clone(destName, keys, verbose)
	if err != nil {
		error_dispatch(err, "%s", err)
		return C.int(0)
	}
	return C.int(1)
}

// clone_sample is like clone both generates a sample or test and
// training set of sampled of the cloned collection.
//
//export clone_sample
func clone_sample(cName *C.char, cTrainingName *C.char, cTestName *C.char, cSampleSize C.int) C.int {
	collectionName := C.GoString(cName)
	sampleSize := int(cSampleSize)
	trainingName := C.GoString(cTrainingName)
	testName := C.GoString(cTestName)

	error_clear()
	c, err := dataset.Open(collectionName)
	if err != nil {
		error_dispatch(err, "%s", err)
		return C.int(0)
	}
	defer c.Close()
	keys := c.Keys()
	err = c.CloneSample(trainingName, testName, keys, sampleSize, verbose)
	if err != nil {
		error_dispatch(err, "%s", err)
		return C.int(0)
	}
	return C.int(1)
}

// grid generates a "Grid" structure from a collection.
//
//export grid
func grid(cName *C.char, cKeys *C.char, cDotPaths *C.char) *C.char {
	collectionName := C.GoString(cName)
	srcKeys := C.GoString(cKeys)
	srcDotpaths := C.GoString(cDotPaths)
	error_clear()
	c, err := dataset.Open(collectionName)
	if err != nil {
		error_dispatch(err, "%s", err)
		return C.CString("")
	}
	defer c.Close()
	keys := []string{}
	err = json.Unmarshal([]byte(srcKeys), &keys)
	if err != nil {
		error_dispatch(err, "Can't unmarshal keys, %s", err)
		return C.CString("")
	}
	dotPaths := []string{}
	err = json.Unmarshal([]byte(srcDotpaths), &dotPaths)
	if err != nil {
		error_dispatch(err, "Can't unmarshal dot paths, %s", err)
		return C.CString("")
	}
	//NOTE: We're picking up the verbose flag from the modules global state
	g, err := c.Grid(keys, dotPaths, verbose)
	if err != nil {
		error_dispatch(err, "failed to create grid, %s", err)
		return C.CString("")
	}
	src, err := json.Marshal(g)
	if err != nil {
		error_dispatch(err, "failed to marshal grid, %s", err)
		return C.CString("")
	}
	txt := fmt.Sprintf("%s", src)
	return C.CString(txt)
}

// frame creates a data frame in a collection. It needs a JSON array of
// keys, dotpaths and labels and returns JSON source of the new frame.
//
//export frame
func frame(cName *C.char, cFName *C.char, cKeys *C.char, cDotPaths *C.char, cLabels *C.char) *C.char {
	collectionName := C.GoString(cName)
	frameName := C.GoString(cFName)
	srcKeys := C.GoString(cKeys)
	srcDotpaths := C.GoString(cDotPaths)
	srcLabels := C.GoString(cLabels)
	error_clear()
	c, err := dataset.Open(collectionName)
	if err != nil {
		error_dispatch(err, "%s", err)
		return C.CString("")
	}
	defer c.Close()
	keys := []string{}
	err = json.Unmarshal([]byte(srcKeys), &keys)
	if err != nil {
		error_dispatch(err, "Can't unmarshal keys, %s", err)
		return C.CString("")
	}
	dotPaths := []string{}
	err = json.Unmarshal([]byte(srcDotpaths), &dotPaths)
	if err != nil {
		error_dispatch(err, "Can't unmarshal dot paths, %s", err)
		return C.CString("")
	}
	labels := []string{}
	err = json.Unmarshal([]byte(srcLabels), &labels)
	if err != nil {
		error_dispatch(err, "Can't unmarshal labels, %s", err)
		return C.CString("")
	}
	//NOTE: We're picking up the verbose flag from the modules global state
	f, err := c.Frame(frameName, keys, dotPaths, labels, verbose)
	if err != nil {
		error_dispatch(err, "failed to create frame, %s", err)
		return C.CString("")
	}
	src, err := json.Marshal(f)
	if err != nil {
		error_dispatch(err, "failed to marshal frame, %s", err)
		return C.CString("")
	}
	txt := fmt.Sprintf("%s", src)
	return C.CString(txt)
}

// has_frame returns 1 if the frame name exists in the collection,
// otherwise 0.
//
//export has_frame
func has_frame(cName *C.char, cFName *C.char) C.int {
	collectionName := C.GoString(cName)
	frameName := C.GoString(cFName)
	error_clear()
	c, err := dataset.Open(collectionName)
	if err != nil {
		error_dispatch(err, "%s", err)
		return C.int(0)
	}
	defer c.Close()
	if c.HasFrame(frameName) {
		return C.int(1)
	}
	return C.int(0)
}

// frames returns a JSON array of frames names in the collection.
//
//export frames
func frames(cName *C.char) *C.char {
	collectionName := C.GoString(cName)
	error_clear()
	c, err := dataset.Open(collectionName)
	if err != nil {
		error_dispatch(err, "%s", err)
		return C.CString("")
	}
	defer c.Close()

	frameNames := c.Frames()
	if len(frameNames) == 0 {
		return C.CString("[]")
	}
	src, err := json.Marshal(frameNames)
	if err != nil {
		error_dispatch(err, "failed to marshal frame names, %s", err)
		return C.CString("")
	}
	txt := fmt.Sprintf("%s", src)
	return C.CString(txt)
}

// reframe takes a JSON array of keys and updates the frame's
// object list.
//
//export reframe
func reframe(cName *C.char, cFName *C.char, cKeys *C.char) C.int {
	collectionName := C.GoString(cName)
	frameName := C.GoString(cFName)
	srcKeys := C.GoString(cKeys)
	error_clear()
	c, err := dataset.Open(collectionName)
	if err != nil {
		error_dispatch(err, "%s", err)
		return C.int(1)
	}
	defer c.Close()
	keys := []string{}
	err = json.Unmarshal([]byte(srcKeys), &keys)
	if err != nil {
		error_dispatch(err, "Can't unmarshal keys, %s", err)
		return C.int(1)
	}
	//NOTE: We're picking up the verbose flag from the modules global state
	err = c.Reframe(frameName, keys, verbose)
	if err != nil {
		error_dispatch(err, "failed to reframe, %s", err)
		return C.int(1)
	}
	return C.int(0)
}

// delete_frame removes a frame from a collection.
//
//export delete_frame
func delete_frame(cName *C.char, cFName *C.char) C.int {
	collectionName := C.GoString(cName)
	frameName := C.GoString(cFName)
	error_clear()
	c, err := dataset.Open(collectionName)
	if err != nil {
		error_dispatch(err, "%s", err)
		return C.int(1)
	}
	defer c.Close()
	//NOTE: We're picking up the verbose flag from the modules global state
	err = c.DeleteFrame(frameName)
	if err != nil {
		error_dispatch(err, "failed to delete frame %s", err)
		return C.int(1)
	}
	return C.int(0)
}

// sync_send_csv - synchronize a frame sending data to a CSV file
//
//export sync_send_csv
func sync_send_csv(cName *C.char, cFName *C.char, cCSVFilename *C.char, cSyncOverwrite C.int) C.int {
	var (
		c   *dataset.Collection
		src []byte
		err error
	)
	collectionName := C.GoString(cName)
	frameName := C.GoString(cFName)
	csvFilename := C.GoString(cCSVFilename)
	syncOverwrite := (cSyncOverwrite == 1)

	src, err = ioutil.ReadFile(csvFilename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		return C.int(1)
	}
	if len(src) == 0 {
		fmt.Fprintf(os.Stderr, "No data in csv file %s\n", csvFilename)
		return C.int(1)
	}

	table := [][]interface{}{}
	// Populate table to sync
	if len(src) > 0 {
		// for CSV
		r := csv.NewReader(bytes.NewReader(src))
		csvTable, err := r.ReadAll()
		if err == nil {
			table = tbl.TableStringToInterface(csvTable)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			return C.int(1)
		}
	}

	c, err = dataset.Open(collectionName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		return C.int(1)
	}
	defer c.Close()

	// Merge collection content into table
	table, err = c.MergeIntoTable(frameName, table, syncOverwrite, verbose)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		return C.int(1)
	}

	// Save the resulting table
	if len(src) > 0 {
		if err = os.Rename(csvFilename, csvFilename+".bak"); err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			return C.int(1)
		}
		if out, err := os.Create(csvFilename); err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			return C.int(1)
		} else {
			w := csv.NewWriter(out)
			w.WriteAll(tbl.TableInterfaceToString(table))
			err = w.Error()
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s\n", err)
				return C.int(1)
			}
		}
	}
	return C.int(0)
}

// sync_recieve_csv - synchronize a frame recieving data from a CSV file
//
//export sync_recieve_csv
func sync_recieve_csv(cName *C.char, cFName *C.char, cCSVFilename *C.char, cSyncOverwrite C.int) C.int {
	var (
		c   *dataset.Collection
		src []byte
		err error
	)
	collectionName := C.GoString(cName)
	frameName := C.GoString(cFName)
	csvFilename := C.GoString(cCSVFilename)
	syncOverwrite := (cSyncOverwrite == 1)

	src, err = ioutil.ReadFile(csvFilename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		return C.int(1)
	}

	table := [][]interface{}{}
	// Populate table to sync
	if len(src) > 0 {
		// for CSV
		r := csv.NewReader(bytes.NewReader(src))
		csvTable, err := r.ReadAll()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			return C.int(1)
		}
		table = tbl.TableStringToInterface(csvTable)
	}

	c, err = dataset.Open(collectionName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		return C.int(1)
	}
	defer c.Close()

	// Merge table contents into Collection and Frame
	err = c.MergeFromTable(frameName, table, syncOverwrite, verbose)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		return C.int(1)
	}
	return C.int(0)
}

// sync_send - synchronize a frame sending data to a GSheet
//
//export sync_send_gsheet
func sync_send_gsheet(cName, cFName, cGSheetID, cGSheetName, cCellRange *C.char, cSyncOverwrite C.int) C.int {

	var (
		c   *dataset.Collection
		err error
	)
	collectionName := C.GoString(cName)
	frameName := C.GoString(cFName)
	gSheetID := C.GoString(cGSheetID)
	gSheetName := C.GoString(cGSheetName)
	cellRange := C.GoString(cCellRange)
	syncOverwrite := (cSyncOverwrite == 1)

	table := [][]interface{}{}
	// Populate table to sync
	// for GSheet
	clientSecretJSON := os.Getenv("GOOGLE_CLIENT_SECRET_JSON")
	if clientSecretJSON == "" {
		//clientSecretJSON = "client_secret.json"
		clientSecretJSON = "credentials.json"
	}
	table, err = gsheets.ReadSheet(clientSecretJSON, gSheetID, gSheetName, cellRange)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		return C.int(1)
	}

	c, err = dataset.Open(collectionName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		return C.int(1)
	}
	defer c.Close()

	// Merge collection content into table
	table, err = c.MergeIntoTable(frameName, table, syncOverwrite, verbose)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		return C.int(1)
	}

	// Save the resulting table
	clientSecretJSON = os.Getenv("GOOGLE_CLIENT_SECRET_JSON")
	if clientSecretJSON == "" {
		//clientSecretJSON = "client_secret.json"
		clientSecretJSON = "credentials.json"
	}
	// NOTE: WriteSheet expects a [][]interface{} not [][]string,
	// need to convert. This is a hack...
	t := [][]interface{}{}
	for _, row := range table {
		cells := []interface{}{}
		for _, cell := range row {
			cells = append(cells, cell)
		}
		t = append(t, cells)
	}
	err = gsheets.WriteSheet(clientSecretJSON, gSheetID, gSheetName, cellRange, t)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		return C.int(1)
	}
	return C.int(0)
}

// sync_recieve_gsheet - synchronize a frame recieving data from a GSheet
//
//export sync_recieve_gsheet
func sync_recieve_gsheet(cName, cFName, cGSheetID, cGSheetName, cCellRange *C.char, cSyncOverwrite C.int) C.int {
	var (
		c   *dataset.Collection
		err error
	)
	collectionName := C.GoString(cName)
	frameName := C.GoString(cFName)
	gSheetID := C.GoString(cGSheetID)
	gSheetName := C.GoString(cGSheetName)
	cellRange := C.GoString(cCellRange)
	syncOverwrite := (cSyncOverwrite == 1)

	if cellRange == "" {
		cellRange = "A1:Z"
	}

	table := [][]interface{}{}
	// Populate table to sync
	// for GSheet
	clientSecretJSON := os.Getenv("GOOGLE_CLIENT_SECRET_JSON")
	if clientSecretJSON == "" {
		//clientSecretJSON = "client_secret.json"
		clientSecretJSON = "credentials.json"
	}
	table, err = gsheets.ReadSheet(clientSecretJSON, gSheetID, gSheetName, cellRange)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		return C.int(1)
	}

	c, err = dataset.Open(collectionName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		return C.int(1)
	}
	defer c.Close()

	// Merge table contents into Collection and Frame
	err = c.MergeFromTable(frameName, table, syncOverwrite, verbose)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		return C.int(1)
	}
	return C.int(0)
}

// frame_grid takes a frames object list and returns a grid
// (2D JSON array) representation of the object list.
// If the "header row" value is 1 a header row of labels is
// included, otherwise it is only the values of returned in the grid.
//
//export frame_grid
func frame_grid(cName *C.char, cFName *C.char, cIncludeHeaderRow C.int) *C.char {
	collectionName := C.GoString(cName)
	frameName := C.GoString(cFName)
	includeHeaderRow := false
	if cIncludeHeaderRow == C.int(1) {
		includeHeaderRow = true
	}
	error_clear()
	c, err := dataset.Open(collectionName)
	if err != nil {
		error_dispatch(err, "%s", err)
		return C.CString("")
	}
	defer c.Close()
	f, err := c.Frame(frameName, nil, nil, nil, false)
	if err != nil {
		error_dispatch(err, "%s", err)
		return C.CString("")
	}
	g := f.Grid(includeHeaderRow)
	src, err := json.Marshal(g)
	if err != nil {
		error_dispatch(err, "%s", err)
		return C.CString("")
	}
	txt := fmt.Sprintf("%s", src)
	return C.CString(txt)
}

// frame_objects returns a copy of a frame's object list as an
// array of objects in JSON source. The array is ordered, the attributes
// in the objects are not ordered.
//
//export frame_objects
func frame_objects(cName *C.char, cFName *C.char) *C.char {
	collectionName := C.GoString(cName)
	frameName := C.GoString(cFName)
	error_clear()
	c, err := dataset.Open(collectionName)
	if err != nil {
		error_dispatch(err, "%s", err)
		return C.CString("")
	}
	defer c.Close()
	f, err := c.Frame(frameName, nil, nil, nil, false)
	if err != nil {
		error_dispatch(err, "%s", err)
		return C.CString("")
	}
	src, err := json.Marshal(f.ObjectList)
	if err != nil {
		error_dispatch(err, "%s", err)
		return C.CString("")
	}
	txt := fmt.Sprintf("%s", src)
	return C.CString(txt)
}

func main() {}
