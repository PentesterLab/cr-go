package main
import (
  "fmt"
  "strconv"
  "net/http"
  "encoding/json"
  "net/url"
  "strings"
  "time"
  "github.com/gorilla/mux"
  "math/rand"
  jwt "github.com/dgrijalva/jwt-go"
  "github.com/rs/cors"
)

var accountSid = "ACXXXX"
var authToken = "XXXXXX"
var TwilioAPI =  "https://api.twilio.com/2010-04-01/Accounts/"+accountSid+"/Messages.json"
func sendSMS(number string, code string) {
//https://www.twilio.com/blog/2017/09/send-text-messages-golang.html
  msgData := url.Values{}
  msgData.Set("To",number)
  msgData.Set("From","13371337")
  msgData.Set("Body","Your one time code is"+code)
  msgDataReader := *strings.NewReader(msgData.Encode())
  client := &http.Client{}
  req, _ := http.NewRequest("POST", TwilioAPI, &msgDataReader)
  req.SetBasicAuth(accountSid, authToken)
  req.Header.Add("Accept", "application/json")
  req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
  resp, _ := client.Do(req)
  if (resp.StatusCode >= 200 && resp.StatusCode < 300) {
    var data map[string]interface{}
    decoder := json.NewDecoder(resp.Body)
    err := decoder.Decode(&data)
    if (err == nil) {
      fmt.Println(data["sid"])
    }
  } else {
    fmt.Println(resp.Status);
  }
  
}



var GenerateCodeHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request){
  w.Header().Set("Content-Type", "application/json")
  vars := mux.Vars(r)
  fmt.Println(vars["number"])
  code := fmt.Sprintf("%06d",rand.Intn(100000))

  // send code via SMS to Twillio 
  //
  token := jwt.NewWithClaims(jwt.SigningMethodHS256,jwt.MapClaims{
    "code": code,
  })

  tokenString, _ := token.SignedString(Secret)
  cookie := http.Cookie{
                Name: "mfa_auth", 
                Value: tokenString, 
                Expires: time.Now().AddDate(0, 0, 1), 
                HttpOnly: false,
                Secure: false,
            }
  http.SetCookie(w,&cookie)
  w.Write([]byte("{}"))
})

var StatusHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request){
  w.Header().Set("Content-Type", "application/json")
  w.Write([]byte("{\"secret\":\""+string(Secret)+"\",\"startime\":\""+strconv.FormatInt(StartTime,10)+"\"}"))
})


var OkHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request){
  w.Header().Set("Content-Type", "application/json")
})

var VerifyCodeHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request){
  w.Header().Set("Content-Type", "application/json")
  tokenString,err := r.Cookie("mfa_auth")
  if err!=nil {
    http.Error(w, "{\"result\":\"fail\"}", 401)
    return 
  }  
  token, _ := jwt.Parse(tokenString.Value, func(token *jwt.Token) (interface{}, error) {
    return Secret, nil
  }) 
  if claims, ok := token.Claims.(jwt.MapClaims); ok {
    vars := mux.Vars(r)
    if ((claims["code"] !=  "") &&(claims["code"] == vars["code"])) { 
      w.Write([]byte("{\"result\":\"success\"}"))
      return
    } else {
      http.Error(w, "{\"result\":\"fail\"}", 401)
      return
    }
  } else {
    http.Error(w, "{\"result\":\"fail\"}", 401)
    return
  }
})
var StartTime = time.Now().Unix()
var Secret = []byte("secret")
func main() {
  //var StartTime= time.Now().Unix()
  rand.Seed(StartTime)
  r := mux.NewRouter()
  r.HandleFunc("/generate_code", GenerateCodeHandler)
  r.HandleFunc("/verify_code/{code}", VerifyCodeHandler)
  r.HandleFunc("/_status", StatusHandler)
  r.HandleFunc("/", OkHandler)

  c := cors.New(cors.Options{
    AllowCredentials: true,
    AllowedMethods: []string{"GET", "POST", "PUT", "DELETE"},
  })
  r_c:= c.Handler(r)
  http.ListenAndServe("0.0.0.0:8080", r_c)
}

