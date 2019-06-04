# Discovery Streams

This project is aimed at creating a Streams API to deliver Discovery video content through HTTP requests.
In order to do this this project uses JWT tokens for authentication and caching through Redis 
to speed up request, as well as, blacklist tokens after logout.

The Endpoints are:

```
POST /signup - Sign Up to get access to service
POST /login - Provides a JWT token to make api requests to Stream Endpoint
DELETE /logout - Invalidates JWT token 
GET /v1/streams/{streamID} - Get Stream Data
GET /v1/streams - Lists all StreamID's
```

* On /login the jwt token will be returned not in the response body but in the Authorization Header
* All endpoints besides login and signup require a token to access data
* JWT tokens are good for 1 hour
## Getting Started

These instructions will get you a copy of the project up and running on your local machine for development and testing purposes.

### Prerequisites

What things you need to install:


[Dep](https://github.com/golang/dep) - Golang's Dependency Management tool

[Docker](https://www.docker.com/products/docker-desktop) - Tool for application containerization


## Installing

```
git clone https://github.com/ImperiousEnterprise/DiscoveryStream.git (clone to $GOPATH/src)
```

## Running the application

#### With Dockers

1.) Run With Dockers
```
 - docker-compose up --build (start dockers)
 - Ctrl + C (close dockers)
```
* Once docker is up access the api with http://localhost:7000


#### Without Dockers

```
1.) dep ensure
2.) Export Environment Variables:
    MONGO_URI=mongodb://<mongo_db_host>:<mongodb_ip>
    MONGO_DB_NAME=<name_of_db_in_mongo>
    ADS_URL=https://coding-challenge.dsc.tv/v1/ads/
    REDIS_ADDRESS=<redis_host>:6379
    TOKEN_SECRET=<your_secret_text>
    PORT=<any_port_you_wanna_use>
3.) go run main.go
```

* As an environment variable, ```REDIS_PASSWORD``` is possible to use if your redis
instance will be password protected.
#### Hybrid with Dockers

```
1.) docker-compose up -d mongodb redis
2.) dep ensure
2.) Export Environment Variables:
    MONGO_URI=mongodb://localhost:5001
    MONGO_DB_NAME=discovery
    ADS_URL=https://coding-challenge.dsc.tv/v1/ads/
    REDIS_ADDRESS=localhost:6001
    TOKEN_SECRET=<your_secret_text>
    PORT=<any_port_you_wanna_use>
3.) go run main.go

```

## How to query API

Here are few curl quests to help

1.) /signup
```
curl -X POST \
  http://localhost:7000/signup \
  -H 'Content-Type: application/json' \
  -d '{
	"email":"ab@example.com",
	"firstname":"atel",
	"lastname": "smith",
	"password":"rock12"
}'
```

2.) /login
```
curl -v -X POST \
  http://localhost:7000/login \
  -H 'Content-Type: application/json' \
  -d '{
	"email":"ab@example.com",
	"password":"rock12"
}'
```

3.) /v1/streams
```
curl -X GET \
  http://localhost:7000/v1/streams \
  -H 'Authorization: Bearer <replace_with_token_from_login_api>'
```

4.) /v1/streams/{streamID}
```
curl -X GET \
  http://localhost:7000/v1/streams/5938b99cb6906eb1fbaf1f1d \
  -H 'Authorization: Bearer <replace_with_token_from_login_api>'
```

4.) /logout
```
curl -X DELETE \
  http://localhost:7000/logout \
  -H 'Authorization: Bearer <replace_with_token_from_login_api>'
  ```

## Running tests

Integration tests are ran in docker containers
```
docker-compose -f docker-compose.test.yml up --abort-on-container-exit
```

After the test run docker takes down the containers.



## Built With

* [Golang's go-chi](https://github.com/go-chi/chi) - The web framework used
* [jwtauth](https://github.com/go-chi/jwtauth) - JWT authentication middleware for go-chi
* [mongo-go-driver](https://github.com/mongo-go-driver) - Mongo db driver for golang
* [go-redis](https://github.com/go-redis/redis) - Redis driver for golang

## Author

* **Adefemi Adeyemi**  - [ImperiousEnterprise](https://github.com/ImperiousEnterprise)



# Etc..

1.) If mongodb name will be changed be sure to update ```MONGO_DB_NAME``` as well as the import.sh under ```/build/mongo```
