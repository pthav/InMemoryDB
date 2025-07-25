# InMemoryDB
InMemoryDB is a simple database that stores key-value pairs in memory as strings.

## Table of Contents
- [Installation](#installation)
- [Features](#features)
- [Usage](#usage)
- [License](#license)
## Installation
- Make sure your installed go version is at least 1.24.
  - You can check your version by running `go version` in a terminal.
  - If needed, you can find download instructions [here](https://go.dev/doc/install) on the official Go website.
- Clone the repository (`git clone https://github.com/pthav/InMemoryDB`).
- Run `go mod tidy`.
- Docker can be optionally installed if you need to dockerize the application. 
## Features
### Database
- Key-value pairs are stored in memory. Currently string is the only supported type.
- TTL (time to live) can be optionally provided when creating or updating key-value pairs.
- Configuration is enabled through optional functions that may be passed in with instantiation.
  - A start up JSON file may be provided.
  - Persistence may be enabled with a specified cycle and a specified file to persist to. The database will also attempt to persist on shutdown. The format will be JSON.
  - Logging can be customized with an injectable logger
- Concurrency is supported through a read-write mutex.
### API
- Response bodies are of type JSON
- In the event of an error, all endpoints will respond with an appropriate status code and a JSON struct of the form `{"error":"error message"}`.
- Likewise to the database, the handler implementation supports logging with a customized, injectable logger.
- `GET /v1/keys/{key}` provides access to key-value pairs.
- `GET /v1/ttl/{key}` provides access to key-TTL pairs. 
- `DELETE /v1/keys/{key}` will delete a key-value pair if it exists.
- `PUT /v1/keys/{key}` will put a key-value pair into the database with the option to also assign a TTL.
- `POST /v1/keys` will post a value into the database and return the generated UUID associated with the posted value. You can also optionally assign a TTL.
- `GET /v1/subscribe/{channel}` will subscribe to a channel and receive messages in the SSE (server-sent events) format.
- `POST /v1/publish/{channel}` will publish a message to the corresponding channel and all subscribers to this channel will receive the message.
- `GET /metrics` provides prometheus friendly metrics. Currently the number of active subscriptions, the cumulative number of published messages, a latency histogram, and a request counter histogram are provided.
### CLI (command line interface)
- The CLI provides commands for serving a database and communicating with the API of a database instance.
- server is a parent command
  - serve is used to serve database instances
- endpoint is a parent command
  - get is used to get key-value pairs
  - getTTL is used to get key-TTL pairs
  - delete is used to delete key-value pairs
  - put is used to put key-value pairs with an optional TTL
  - post is used to post values with an optional TTL
  - publish is used to publish messages to channels
  - subscribe is used to subscribe to channels
### Docker
A docker file and docker compose file have been provided. If built unchanged, the compose should serve a database with an '8080:8080' port binding.
  
## Usage
### API
- `GET /v1/keys/{key}`: Sending a GET request to the uri `/v1/keys/hello` will return the value associated with key 'hello' if such a key-value pair exists. The resulting JSON response is of the form `{"key":"the key", "value":"the value"}`
- `GET /v1/ttl/{key}`: Sending a GET request to the uri `/v1/keys/hello` will return the TTL associated with the key `hello` if such a key-value pair exists. The resulting JSON response is of the form `{"key":"the key", "ttl":10}`
- `DELETE /v1/keys/{key}`: Sending a DELETE request to the uri `/v1/keys/hello` will delete the key-value pair associated with `hello` if it exists. The response body will be empty JSON.
- `PUT /v1/keys/{key}`: Sending a PUT request to the uri `/v1/keys/hello` with a request body of `{"value":"world", "ttl":10}` will update the key-value pair if it already exists or create it if it doesn't. It will additionally have a TTL of 10 seconds. Only the 'value' is required for the request body.
- `POST /v1/keys`: Sending a POST request to the uri `/v1/keys/hello` with a request body of `{"value":"world", "ttl":10}` will create a UUID key for the value and add it to the database. Like the PUT request, it will have a TTL of 10 seconds. The TTL is also optional for POST.
- `POST /v1/publish/{channel}`: Sending a POST request to the uri `/v1/publish/workspace` with a request body of `{"message":"hello"}` will send 'hello' to all subscribers listening on the 'workspace' channel.
- `GET /v1/subscribe/{channel}`: Sending a GET request to the uri `/v1/subscribe/workspace` will open an SSE subscription to the 'workspace' channel.
### CLI
The CLI is split into `server` and `endpoint` parent commands.
- Server
  - serve allows you to serve an instance of the database.
    - `--host` sets the host for the API to listen on.
    - `--startup-file` allows specification of JSON formatted starting data to boot with.
    - `--persist` is a boolean flag that enables persistence. This flag is required when using the `--persist-file` flag.
    - `--persist-file` will set the persistence output to the specified file and is required when using the `--persist` flag.
    - `--persist-cycle, -c` allows for a set cycle in seconds to routinely persist on.
    - `--no-log` is a boolean flag that will disable logging for both the database and API when set.
- Endpoint commands will forward a request to the API of a database and output the response to STDOUT in indented JSON.
  - `--rootURL, -u` establishes the root URL to forward requests to.
  - get
    - `--key, -k` sets the key to retrieve an associated value for.
  - getTTL
    - `--key, -k` sets the key to retrieve an associated TTL for.
  - delete
    - `--key, -k` sets the key to delete.
  - put
    - `--key, -k` sets the key to put.
    - `--value, -v` sets the value to put.
    - `--ttl` sets the TTL to put.
  - post
    - `--value, -v` sets the value to put.
    - `--ttl` sets the TTL to put.
  - publish
    - `--channel, -c` sets the channel to send to.
    - `--message, -m` sets the message to send.
  - subscribe
    - `--channel, -c` sets the channel to subscribe to.
    - `--timeout, -t` sets the timeout for a subscription.
#### CLI Examples
- `server serve --host localhost:8080 --startup-file startup.json --persist --persist-file persist.json --persist-cycle 120 --no-log` will serve a database on localhost:8080 initialized with the data stored in startup.json. It will also persist every 120 seconds to persist.json and will not log.
- `endpoint get -k hello` will get the value associated with the key 'hello'.
- `endpoint getTTL -k hello` will get the TTL associated with the key 'hello'.
- `endpoint delete -k hello` will delete the 'hello' key.
- `endpoint put -k hello -v world` will put the key-value pair (hello,world) onto the database.
- `endpoint put -k hello -v world --ttl 30` will put the key-value pair (hello,world) onto the database with a TTL of 30 seconds.
- `endpoint post -v world` will post the value 'world' onto the database.
- `endpoint post -v world --ttl 30` will post the value 'world' onto the database with a TTL of 30 seconds.
- `endpoint publish -c workspace -m cats` will send the message 'cats' to the 'workspace' channel.
- `endpoint subscribe -c workspace -t 60` will subscribe to the 'workspace' channel for 60 seconds.

## License
This project is licensed under the [MIT License](LICENSE).
