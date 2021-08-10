Test task
Description
Create an authentication service. There should be 4 REST API routes with the following functionality:
  returning Access and Refresh tokens for the client whose identifier (UUID) is specified in the request parameter; +
  refreshing the Access token;                                                                                      +
  removing of the specified Refresh token from DB (only 1 token can be specified);                                  +
  removing all Refresh tokens that relate to a certain client from DB.                                              +
There are several requirements to the tokens:
  Refresh token must be protected from changes on the client-side                                                   ?
  Refresh token cannot be reused more than once                                                                     +
  Refresh operation for an Access token can be performed only with the Refresh token that was issued along with it. +
Technologies
  Programming language: Go                                                                                          +
  DB:
    DBMS: MongoDB                                                                                                   +
    Topology: Replica set                                                                                           ?
  Access token
    Type: JWT                                                                                                       +
    Encryption algorithm: SHA 512                                                                                   +
  Refresh token
    Type: any                                                                                                       +
    Transfer format: base 64                                                                                        +
    Storing in DB: bcrypt hash                                                                                      +
  Dependency management should be done with Go Modules                                                              ?



//пометки
mongod Program Files\MongoDB\Server\5.0\bin
mongosh Users\{user}\AppData\Local\Programs\mongosh
start database - в cmd->mongod (старт сервера mongodb) -> в другом cmd-> mongosh (консоль бд) (use test -> db.tokens.find())

start authentication service - go run main.go

запросы из Postman:

1. returning Access and Refresh tokens for the client whose identifier (UUID) is specified in the request parameter;
    GET http://localhost:12345/{uuid}
2. refreshing the Access token;
    GET http://localhost:12345/{uuid}/refresh/{refreshtoken}
3. removing of the specified Refresh token from DB (only 1 token can be specified);
    DELETE http://localhost:12345/{uuid}/{refreshtoken}
4. removing all Refresh tokens that relate to a certain client from DB.
    DELETE http://localhost:12345/{uuid}

    вынести приложение и базу в контейнер, параметры env в docker-compose
