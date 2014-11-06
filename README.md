# moredis

`moredis` is a tool to sync data from MongoDB into redis.

## Motivation

MongoDB (and any database for that matter) becomes unwieldy if you have many applications using it for many different purposes.
Oftentimes building out your infrastructure like this makes sense to start, but as time goes on and the number of applications increases it becomes harder to do things like diagnose database performance problems and make application-specific database optimizations.

`moredis` is a tool that reduces an application's direct dependency on MongoDB by syncing specific data out of MongoDB and into redis.
The data is synced in a way that optimizes for the query patterns needed by the application, so that MongoDB to no longer lies in the request path of the application.

See this talk by foursquare for more detailed motivation behind breaking up MongoDB monoliths into a more service-oriented persistence layer: [Service Oriented Clusters](https://www.mongodb.com/presentations/service-oriented-clusters-foursquare-0).

## How it Works

In a nutshell, `moredis` works by taking a user specified MongoDB query, then for each returned document, mapping some some value in the document to another value in that document using a redis hash.  `moredis` also allows you to parameterize your query with values passed in at runtime.

For more specific examples, see [Examples](#examples)

## Usage
```bash
Usage of ./moredis:
  -c, -cache        Which cache to populate (REQUIRED)
  -m, -mongo_url    MongoDB URL, can also be set via the MONGO_URL environment variable
  -p, -params       JSON object with params used for substitution into queries and collection names in config.yml
  -r, -redis_url    Redis URL, can also be set via the REDIS_URL environment variable
  -f, -conf_file    Config file, defaults to ./config.yml
  -h, -help         Print this usage message.
```

## Configuration

`moredis` cache configuration is done using yaml.  You can specify a config file to use, or `moredis` will default to config.yml in the same folder as the `moredis` executable.  This repo contains a sample config.yml which you can to modify to suit your needs.  The [sample](./config.yml) has comments to describe the various fields and their purposes.

You also need to provide `moredis` with connection parameters for both your MongoDB instance and Redis instance.  These settings can be set with either command line flags or environment variables (with the command line flags taking precedence).  Mongo URL should be a MongoDB connection string (exact form expected can be found [in the mgo docs](http://godoc.org/gopkg.in/mgo.v2#Dial)).  Redis URL should be in the form "host:port".

For each, the settings locations are:

* Mongo URL
    * flag: -m
    * env: MONGO_URL
    * default: localhost

* Redis URL
    * flag: -r
    * env: REDIS_URL
    * default: localhost

## Examples

### Simple case insensitive map

Lets say you have a MongoDB collection called 'users', and in this collection you have documents that look like:

```javascript
{
  _id: ObjectId("507f1f77bcf86cd799431111"),
  username: "CoolDude",
  email: "CoolDude25@hotmail.com",
  group: ObjectId("507f1f77bcf86cd799432222")
}
```

Now imagine you were writing a service which required very fast lookups of ids by email in a case-insensitive way.  You could accomplish this with the following `moredis` configuration:

```yaml
caches:
  -
    name: 'demo-cache'
    collections:
      - collection: 'users'
        query: '{}'
        maps:
          - name: 'users:email'
            key: '{{toLower .email}}'
            val: '{{toString ._id}}'
```

Then run `moredis` with:

```bash
$ ./moredis -c demo-cache
```

After this runs, you will have a key in redis called 'users:email'.  This value for this key will be the key for a hash that has all of your email-to-id mappings.  Doing a lookup of the user id for email 'cooldude25@hotmail.com' in redis would look like the following in redis-cli:

```bash
> GET users:email
"moredis:map:1"

> HGET moredis:map:1 cooldude25@hotmail.com
"507f1f77bcf86cd799431111"
```

### Specifying a query

That's great if you want to create the mapping for every document in a collection, but often you only want to create the mapping for some subset of documents in a collection.  The natural way is to use a query to find the set of documents to operate on (i.e. only for users who are tagged with a specific group).

With `moredis` you can do this by specifying a query in the above config like so:

```yaml
caches:
  -
    name: 'demo-cache'
    collections:
      - collection: 'users'
        query: '{"group": "507f1f77bcf86cd799432222"}'
        maps:
          - name: 'users:email'
            key: '{{toLower .email}}'
            val: '{{toString ._id}}'
```

With this config, we will now only do the mapping for documents in the user collection with the given group id.

### Parameterizing your query

To take this example one step further, not only do you only want to create the cache for a specific group, you want to be able to specify this group at runtime without modifying your config.yml.  With `moredis`, you can do this by taking advantage of parameterization in your config, then you can pass in the parameters you want to use on the command line.

To accomplish passing the group id in at runtime, we could modify our config to now look like:

```yaml
caches:
  -
    name: demo-cache
    collections:
      - collection: users
        query: '{"group": "{{.group}}"}'
        maps:
          - name: 'users:email:{{.group}}'
            key: '{{toLower .email}}'
            val: '{{toString ._id}}'
```

Now you can see that both our query and our map name are parameterized by this "group" parameter.  You can pass that parameter in on the commandline like so:

```bash
$ ./moredis -c demo-cache -p '{"group": "507f1f77bcf86cd799432222"}'
```

The result of this run will be the same as from the previous example, except the map will now contain the group id in the key name (so that caches for different groups don't overwrite each other).

## Installation

You can grab the latest `moredis` release for your platform from the [Releases](https://github.com/Clever/moredis/releases) page.  Then, just extract, configure, and run.

## Local development

You can grab `moredis` for local development in the usual golang way with:

```bash
$ go get github.com/Clever/moredis
```

You can run tests with:

```bash
$ make test
```

## Using as a library

`moredis` can also be used as a library, for example:

```go
package main

import (
  "github.com/Clever/moredis/moredis"
  "log"
)

func main() {
  config, _ := moredis.LoadConfig("./config.yml")
  err := moredis.BuildCache(config, moredis.Params{}, "", "")
  if err != nil {
    log.Fatal(err)
  }
}
```
