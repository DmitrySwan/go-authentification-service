package main

import (
  "context"
  "fmt"
  "log"
  "net/http"
  "time"
  "github.com/gorilla/mux"
  "go.mongodb.org/mongo-driver/bson"
  "go.mongodb.org/mongo-driver/mongo"
  "go.mongodb.org/mongo-driver/mongo/options"
  "os"
  "github.com/dgrijalva/jwt-go"
  "strconv"
  "github.com/twinj/uuid"
  "encoding/json"
  "golang.org/x/crypto/bcrypt"
  "encoding/base64"
)

var collection *mongo.Collection
func init() {
  collection = ConnectDB()
}
func main() {
  fmt.Println("Starting the application...")
  router := mux.NewRouter()
  router.HandleFunc("/{uuid}/{refreshtoken}", deleteRefreshTokenEndpoint).Methods("DELETE")
  router.HandleFunc("/{uuid}", deleteAllRefreshTokenEndpoint).Methods("DELETE")
  router.HandleFunc("/{uuid}", returnTokensEndpoint).Methods("GET")
  router.HandleFunc("/{uuid}/refresh/{refreshtoken}", returnNewActiveTokenEndpoint).Methods("GET")
  http.ListenAndServe(":12345", router)
}

func ConnectDB()  *mongo.Collection {
  clientOptions := options.Client().ApplyURI("mongodb://localhost:27017")
  client, err := mongo.Connect(context.TODO(), clientOptions)
  if err != nil {
    log.Fatal(err)
  }
  fmt.Println("Connected to MongoDB!")
  collection := client.Database("test").Collection("tokens")
  return collection
}

func returnNewActiveTokenEndpoint(response http.ResponseWriter, request *http.Request) {
  params := mux.Vars(request)
  refreshtoken, _ := params["refreshtoken"]
  decoded, _ := base64.StdEncoding.DecodeString(refreshtoken)
  refreshtoken = string(decoded)
  uuid, _ := params["uuid"]
  log.Println("Param 'uuid' is:", string(uuid))
  UUIDunit64, _ := strconv.ParseUint(string(uuid), 10, 64)

  filter := bson.M{"uuid": UUIDunit64}
  cur, err := collection.Find(context.TODO(), filter)
  if err != nil {
    response.WriteHeader(http.StatusInternalServerError)
    response.Write([]byte(`{ "message": "` + err.Error() + `" }`))
    return
  }

  var result TokenDetails
  for cur.Next(context.TODO()) {
    //Create a value into which the single document can be decoded
    var elem TokenDetails
    err := cur.Decode(&elem)
    if err != nil {
      log.Fatal(err)
    }
    check := CheckHash(string(refreshtoken), elem.RefreshToken)
    if check {
      log.Println("check ", elem.RefreshToken)
      result=elem
      break
    }
  }

  if  result.Refreshed  {
    response.Write([]byte(`{ "message": Refresh token cannot be reused more than once }`))
    return
  }
  filterUpdate := bson.M{"refreshtoken": result.RefreshToken}
  tokenDetails, _ := CreateToken(UUIDunit64)
  update := bson.M{
    "$set": bson.M{"accesstoken": tokenDetails.AccessToken, "refreshed": true},
  }
  after := options.After
  opt := options.FindOneAndUpdateOptions{ //!
    ReturnDocument: &after}

    var td TokenDetails

    log.Println("tokenDetails:", opt)
    err1 := collection.FindOneAndUpdate(context.TODO(), filterUpdate, update, &opt).Decode(&td)
    if err1 != nil {
      response.WriteHeader(http.StatusInternalServerError)
      response.Write([]byte(`{ "message": "` + err1.Error() + `" }`))
      return
    }
    log.Println("tokenDetails:", td)
    resp, _ := json.Marshal(td.AccessToken)
    response.Header().Set("content-type", "application/json")
    response.Write(resp)
  }

  func deleteAllRefreshTokenEndpoint(response http.ResponseWriter, request *http.Request) {
    response.Header().Set("content-type", "application/json")
    params := mux.Vars(request)
    uuid, _ := params["uuid"]
    log.Println("Param 'uuid' is:", string(uuid))
    UUIDunit64, _ := strconv.ParseUint(string(uuid), 10, 64)
    filter := bson.M{"uuid": UUIDunit64}
    collection.DeleteMany(context.TODO(), filter)

  }
  func deleteRefreshTokenEndpoint(response http.ResponseWriter, request *http.Request) {
    response.Header().Set("content-type", "application/json")
    params := mux.Vars(request)
    refreshtoken, _ := params["refreshtoken"]
    log.Println("Param 'refreshtoken' is:", string(refreshtoken))
    decoded, _ := base64.StdEncoding.DecodeString(refreshtoken)
    refreshtoken = string(decoded)
    uuid, _ := params["uuid"]
    log.Println("Param 'uuid' is:", string(uuid))
    UUIDunit64, _ := strconv.ParseUint(string(uuid), 10, 64)

    filter := bson.M{"uuid": UUIDunit64}
    cur, err := collection.Find(context.TODO(), filter)
    if err != nil {
      response.WriteHeader(http.StatusInternalServerError)
      response.Write([]byte(`{ "message": "` + err.Error() + `" }`))
      return
    }

    var result TokenDetails
    for cur.Next(context.TODO()) {
      //Create a value into which the single document can be decoded
      var elem TokenDetails
      err := cur.Decode(&elem)
      if err != nil {
        log.Fatal(err)
      }
      check := CheckHash(string(refreshtoken), elem.RefreshToken)
      if check {
        log.Println("check ", elem.RefreshToken)
        result=elem
        break
      }
    }


    filterDelete := bson.M{"refreshtoken": result.RefreshToken}
    collection.DeleteOne(context.TODO(), filterDelete)
  }

  func returnTokensEndpoint(response http.ResponseWriter, request *http.Request) {
    params := mux.Vars(request)
    uuid, _ := params["uuid"]
    log.Println("Param 'uuid' is:", string(uuid))

    /*UUIDs, ok := request.URL.Query()["UUID"] //how to get reuest params after '?'
    if !ok || len(UUIDs[0]) < 1 {
    log.Println("Url Param 'UUID' is missing")
    return
  }
  UUID := UUIDs[0]*/

  UUIDunit64, _ := strconv.ParseUint(string(uuid), 10, 64)
  tokenDetails, _ := CreateToken(UUIDunit64)

  log.Println("tokenDetails is: ", tokenDetails)

  encoded := base64.StdEncoding.EncodeToString([]byte(tokenDetails.RefreshToken))
  tokens := map[string]string{
    "access_token":  tokenDetails.AccessToken,
    "refresh_token": encoded,
  }
  tokenDetails.RefreshToken, _ = Hash(tokenDetails.RefreshToken)
  collection.InsertOne(context.TODO(), tokenDetails)
  resp, _ := json.Marshal(tokens)
  response.Header().Set("content-type", "application/json")
  _, _ = response.Write(resp)
}

func Hash(password string) (string, error) {
  bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
  return string(bytes), err
}

func CheckHash(password, hash string) bool {
  err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
  return err == nil
}

func CreateToken(userid uint64)  (*TokenDetails, error) {
  td := &TokenDetails{}
  atExpires := time.Now().Add(time.Minute * 15).Unix()
  accessUuid := uuid.NewV4().String()

  rtExpires := time.Now().Add(time.Hour * 24 * 7).Unix()
  refreshUuid := uuid.NewV4().String()
  td.Refreshed = false
  td.UUID = userid

  var err error
  //Creating Access Token
  os.Setenv("ACCESS_SECRET", "jdnfksdmfksd") //this should be in an env file
  atClaims := jwt.MapClaims{}
  atClaims["authorized"] = true
  atClaims["access_uuid"] = accessUuid
  atClaims["user_id"] = userid
  atClaims["exp"] = atExpires
  at := jwt.NewWithClaims(jwt.SigningMethodHS512, atClaims)
  td.AccessToken, err = at.SignedString([]byte(os.Getenv("ACCESS_SECRET")))
  if err != nil {
    return nil, err
  }
  //Creating Refresh Token
  os.Setenv("REFRESH_SECRET", "mcmvmkmsdnfsdmfdsjf") //this should be in an env file
  rtClaims := jwt.MapClaims{}
  rtClaims["refresh_uuid"] = refreshUuid
  rtClaims["user_id"] = userid
  rtClaims["exp"] = rtExpires
  rt := jwt.NewWithClaims(jwt.SigningMethodHS512, rtClaims)
  td.RefreshToken, err = rt.SignedString([]byte(os.Getenv("REFRESH_SECRET")))
  if err != nil {
    return nil, err
  }
  return td, err
}

type TokenDetails struct {
  UUID uint64
  AccessToken  string
  RefreshToken string
  Refreshed bool
  //  AccessUuid   string
  //  RefreshUuid  string
  //  AtExpires    int64
  //  RtExpires    int64
}
