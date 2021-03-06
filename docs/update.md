
# update

## Syntax

```
    dataset update COLLECTION_NAME KEY
```

## Description

_update_ will replace a JSON document in a dataset collection for 
a given KEY.  By default the JSON document is read from standard 
input but you can specific a spefic file with the "-input" 
option. The JSON document should aready exist in the collection
when you use update.


## Usage

In this example we assume there is a JSON document on local disc 
named _jane-doe.json_. It contains `{"name":"Jane Doe"}` and the 
KEY is "jane.doe". In the first one we specify the full JSON document 
via the command line after the KEY.  In the second example we read the 
data from _jane-doe.json_. Finally in the last we read the JSON 
document from standard input and save the update to "jane.doe".
The collection name is "people.ds".

```shell
    dataset update people.ds jane.doe '{"name":"Jane Doiel"}'
    dataset update people.ds jane.doe jane-doe.json
    dataset update -i jane-doe.json people.ds jane.doe
    cat jane-doe.json | dataset update people.ds jane.doe
```

Related topics: [keys](keys.html), [create](create.html), [read](read.html), [delete](delete.html)

