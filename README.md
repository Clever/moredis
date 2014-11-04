# moredis

moredis is a tool to sync data from MongoDB into redis.

## Motivation

MongoDB (and any database for that matter) becomes unwieldy if you have many applications using it for many different purposes.
Oftentimes building out your infrastructure like this makes sense to start, but as time goes on and the number of applications increases it becomes harder to do things like diagnose database performance problems and make application-specific database optimizations.

moredis is a tool that reduces an application's direct dependency on MongoDB by syncing specific data out of MongoDB and into redis.
The data is synced in a way that optimizes for the query patterns needed by the application, so that the application no longer requires mongoDB to .

See this talk by foursquare for more detailed motivation behind breaking up mongodb monoliths into a more service-oriented persistence layer: [Service Oriented Clusters](https://www.mongodb.com/presentations/service-oriented-clusters-foursquare-0).

## How it Works

In a nutshell, moredis works by taking a user specified MongoDB query, then for each returned document, mapping some some value in the document to another value in that document using a redis hash.  moredis also allows you to paramaterize your query with values passed in at runtime.

I think this can best be explained with examples:

### Simple case insensitive map

Lets say you have a MongoDB collection called 'users', and in this collection you have documents that look like:

```json
{
  _id: ObjectId("507f1f77bcf86cd799431111"),
  username: "CoolDude",
  email: "CoolDude25@hotmail.com",
  group: ObjectId("507f1f77bcf86cd799432222")
}
```

Now imagine you were writing a service which required very fast lookups of ids by email in a case-insensitive way.  You could accomplish this with the following moredis configuration:

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

Then run moredis with:

```bash
$ ./moredis '{"cache": "demo-cache", "mongo_url": <mongo url>, "mongo_db": <mongo db>, "redis_url": <redis url>}'
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

With moredis you can do this by specifying a query in the above config like so:

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

### Paramaterizing your query

To take this example one step further, not only do you only want to create the cache for a specific group, you want to be able to specify this group at runtime without modifying your config.yml.  With moredis, you can do this by taking advantage of paramaterization in your config, then you can pass in the parameters you want to use on the command line.

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

Now you can see that both our query and our map name are paramaterized by this "group" parameter.  You can pass that parameter in on the commandline like so:

```bash
$ ./moredis '{"cache": "demo-cache", "group": "507f1f77bcf86cd799432222", "mongo_url": <mongo url>, "mongo_db": <mongo db>, "redis_url": <redis url>}'
```

The result of this run will be the same as from the previous example, except the map will now contain the group id in the key name (so that caches for different groups don't overwrite each other).

## Configuration

## Example usage

TODO

## Installation

TODO

## Local development

TODO
