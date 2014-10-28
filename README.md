# moredis

moredis is a tool to sync data from MongoDB into redis.

## Motivation

MongoDB (and any database for that matter) becomes unwieldy if you have many applications using it for many different purposes.
Oftentimes building out your infrastructure like this makes sense to start, but as time goes on and the number of applications increases it becomes harder to do things like diagnose database performance problems and make application-specific database optimizations.

moredis is a tool that reduces an application's direct dependency on MongoDB by syncing specific data out of MongoDB and into redis.
The data is synced in a way that optimizes for the query patterns needed by the application, so that the application no longer requires mongoDB to .

See this talk by foursquare for more detailed motivation behind breaking up mongodb monoliths into a more service-oriented persistence layer: [Service Oriented Clusters](https://www.mongodb.com/presentations/service-oriented-clusters-foursquare-0).

## Example usage

TODO

## Installation

TODO

## Local development

TODO
