package dataset

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"
)

//
// Helper functions
//

func keyFound(s string, l []string) bool {
	for _, ky := range l {
		if ky == s {
			return true
		}
	}
	return false
}

func findBuckets(p string) ([]string, error) {
	var buckets []string

	dirInfo, err := ioutil.ReadDir(p)
	if err != nil {
		return buckets, err
	}
	for _, item := range dirInfo {
		if item.IsDir() == true {
			buckets = append(buckets, item.Name())
		}
	}
	return buckets, nil
}

func findJSONDocs(p string) ([]string, error) {
	var jsonDocs []string

	dirInfo, err := ioutil.ReadDir(p)
	if err != nil {
		return jsonDocs, err
	}
	for _, item := range dirInfo {
		if item.IsDir() == false {
			jname := item.Name()
			if ext := path.Ext(jname); ext == ".json" {
				jsonDocs = append(jsonDocs, jname)
			}
		}
	}
	return jsonDocs, nil
}

func checkFileExists(p string) (string, bool) {
	_, err := os.Stat(p)
	if os.IsNotExist(err) {
		return p, false
	}
	return p, true
}

//
// Exported functionds for dataset cli usage
//

// Analyzer checks a collection for problems
//
// + checks if collection.json exists and is valid
// + checks if keys.json exits and is valid
// + checks version of collection and version of dataset tool running
// + compares keys.json with k/v pairs in collectio.keymap
// + checks if all collection.buckets exist
// + checks for unaccounted for buckets
// + checks if all keys in collection.keymap exist
// + checks for unaccounted for keys in buckets
// + checks for keys in multiple buckets and reports duplicate record modified times
//
func Analyzer(collectionName string) error {
	var (
		eCnt    int
		wCnt    int
		data    interface{}
		keys    []string
		buckets []string
		c       *Collection
		err     error
	)

	// Check of collections.json and keys.json exist
	for _, fname := range []string{"collection.json", "keys.json"} {
		if docPath, exists := checkFileExists(path.Join(collectionName, fname)); exists == false {
			log.Printf("Missing %s", docPath)
			eCnt++
		} else {
			// Make sure we can JSON parse the file
			if src, err := ioutil.ReadFile(docPath); err == nil {
				if err := json.Unmarshal(src, &data); err == nil {
					// release the memory
					data = nil
				} else {
					log.Printf("Error parsing %s, %s", docPath, err)
					eCnt++
				}
			} else {
				log.Printf("Error opening %s, %s", docPath, err)
				eCnt++
			}
		}
	}

	// See if we can open a collection, if not then create an empty struct
	if c, err = Open(collectionName); err == nil {
		if c.Version != Version {
			log.Printf("Version mismatch collection %s, dataset %s", c.Version, Version)
			wCnt++
		}
		defer c.Close()
	} else {
		log.Printf("Open collection error, %s", err)
		c = new(Collection)
		eCnt++
	}

	// Open and parse the keys.json file for comparison
	if src, err := ioutil.ReadFile(path.Join(collectionName, "keys.json")); err == nil {
		json.Unmarshal(src, &keys)
	}
	// Check to see if keys.json matches length of keymap
	if len(c.KeyMap) == len(keys) {
		// Check to see if the keys in keymap and keys in keys.json are the same
		for ky, _ := range c.KeyMap {
			if keyFound(ky, keys) == false {
				log.Printf("%s not found in keys.json", ky)
				wCnt++
			}
		}
	} else {
		log.Printf("%d keys in keymap, %d keys in keys.json", len(c.KeyMap), len(keys))
		eCnt++
	}

	// Find buckets
	buckets, err = findBuckets(collectionName)
	if err != nil {
		log.Printf("No buckets found for %s, %s", collectionName, err)
		eCnt++
	}
	// Check if buckets match
	for _, bck := range buckets {
		if keyFound(bck, c.Buckets) == false {
			log.Printf("%s is missing from collection bucket list", bck)
			eCnt++
		}
	}

	// Check to see if records can be found in their buckets
	for ky, bucket := range c.KeyMap {
		if docPath, exists := checkFileExists(path.Join(collectionName, bucket, ky+".json")); exists == false {
			log.Printf("%s is missing", docPath)
		}
	}

	// Check for duplicate records, and missing records
	for _, bck := range buckets {
		if jsonDocs, err := findJSONDocs(path.Join(collectionName, bck)); err == nil {
			for _, jsonDoc := range jsonDocs {
				ky := strings.TrimSuffix(path.Base(jsonDoc), ".json")
				if val, ok := c.KeyMap[ky]; ok == true {
					if val != bck {
						log.Printf("%s is a duplicate")
					}
				} else {
					log.Printf("%s is an orphaned JSON Doc", path.Join(collectionName, bck, jsonDoc))
					eCnt++
				}
			}
		} else {
			log.Printf("Can't open bucket %s, %s", bck, err)
			eCnt++
		}
	}
	if eCnt > 0 || wCnt > 0 {
		return fmt.Errorf("%d errors, %d warnings detected", eCnt, wCnt)
	}
	return nil
}

// Repair will take a collection name and attempt to recreate
// valid collection.json and keys.json files from content
// in discovered buckets and json documents
func Repair(collectionName string) error {
	var (
		eCnt int
		wCnt int
		c    *Collection
		err  error
	)
	// See if we can open a collection, if not then create an empty struct
	if c, err = Open(collectionName); err == nil {
		if c.Version != Version {
			log.Printf("WARNING: Version mismatch collection moving from %s to %s", c.Version, Version)
			wCnt++
		}
		defer c.Close()
	} else {
		log.Printf("ERROR: %s, creating empty collection object", err)
		c = new(Collection)
		eCnt++
	}
	if buckets, err := findBuckets(path.Join(collectionName)); err == nil {
		c.Buckets = buckets
	} else {
		return err
	}
	for _, bck := range c.Buckets {
		if jsonDocs, err := findJSONDocs(path.Join(collectionName, bck)); err == nil {
			for i, jsonDoc := range jsonDocs {
				ky := strings.TrimSuffix(jsonDoc, ".json")
				if val, ok := c.KeyMap[ky]; ok == true {
					if stat1, err := os.Stat(path.Join(collectionName, bck, ky+".json")); err == nil {
						if stat2, err := os.Stat(path.Join(collectionName, val, ky+".json")); err == nil {
							m1 := stat1.ModTime()
							m2 := stat2.ModTime()
							if m1.Unix() > m2.Unix() {
								log.Printf("Updating key %s in %s (%s) over %s (%s)", ky, bck, m1, val, m2)
								c.KeyMap[ky] = bck
							}
						}
					}
				} else {
					c.KeyMap[ky] = bck
				}
				if i > 0 && (i%5000) == 0 {
					if err := c.saveMetadata(); err != nil {
						return err
					}
					log.Printf("Saving %d items in bucket %s", i, bck)
				}
			}
		} else {
			return err
		}
		if err := c.saveMetadata(); err != nil {
			return err
		}
		log.Printf("Saving bucket %s", bck)
	}
	return c.saveMetadata()
}