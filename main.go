package main

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MyURL struct {
	URL string `json="url"`
}

var alphabet = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func dbClient(ctx context.Context) (*mongo.Client, error) {
	uri := ""
	client, err := mongo.NewClient(options.Client().ApplyURI(uri))
	if err != nil {
		fmt.Printf("Couldn't connect to the cluster : %v\n", err)
		return nil, err
	}

	err = client.Connect(ctx)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func handleCollision(n int, s string) string {
	return s[:len(s)-1] + string(alphabet[n])
}

func reverse(str string) (result string) {
	for _, v := range str {
		result = string(v) + result
	}
	return
}

func encode(num uint64) string {
	var arr []string
	for {
		if num == 0 {
			break
		}
		rem := num % 62
		num = num / 62
		arr = append(arr, string(alphabet[rem]))
	}
	return reverse(strings.Join(arr, ""))
}

func Shortener(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var p MyURL
	err := json.NewDecoder(r.Body).Decode(&p)
	if err != nil {
		fmt.Printf("Couldn't decoce URL : %v\n", err)
		return
	}
	hash := md5.Sum([]byte(p.URL))
	strHash := hex.EncodeToString(hash[:])[:10] // First 40 bits out of the 128 bits should be enough
	val, err := strconv.ParseUint(strHash, 16, 64)
	if err != nil {
		fmt.Println(err)
		return
	}
	b62val := encode(val) // encoding in base62
	for {
		longURL := bson.M{}
		if err = collection.FindOne(ctx, bson.M{"surl": b62val}).Decode(longURL); err == nil {
			fmt.Printf("%v already exists in the database.\n", b62val)
			lURL := longURL["lurl"]
			if lURL == p.URL {
				fmt.Println("Shortened URL already exists for this URL")
				break
			}
			b62val = handleCollision(rand.Intn(62), b62val) // If the shortened URL already exists i.e collision, get new URL
		} else {
			_, err = collection.InsertOne(ctx, bson.M{"surl": b62val, "lurl": p.URL})
			if err != nil {
				resp := fmt.Sprintf("Failed to insert document : %v", err)
				fmt.Println(resp)
				err = json.NewEncoder(w).Encode(resp)
				return
			}
			break
		}
	}

	fmt.Println("Base62 & LongURL", b62val, p.URL)
	resp := fmt.Sprintf("Your shortened URL is : %v/%v", r.Host, b62val)
	fmt.Println(resp)
	err = json.NewEncoder(w).Encode(resp)
	if err != nil {
		fmt.Println("There seems to be an error : ", err)
		return
	}
}

func Expander(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	longURL := bson.M{}
	if err = collection.FindOne(ctx, bson.M{"surl": id}).Decode(longURL); err != nil {
		resp := fmt.Sprintf("Your URL doesn't exist yet : %v", err)
		fmt.Println(resp)
		err = json.NewEncoder(w).Encode(resp)
		return
	}
	lURL := longURL["lurl"]
	fmt.Println("longURL :", lURL)
	strURL, _ := lURL.(string)
	http.Redirect(w, r, strURL, http.StatusSeeOther)
}

var ctx context.Context
var client *mongo.Client
var err error
var collection *mongo.Collection

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err = dbClient(ctx)
	if err != nil {
		fmt.Printf("Some error in database connection %v\n", err)
	}
	defer func() {
		if err := client.Disconnect(ctx); err != nil {
			panic(err)
		}
	}()
	collection = client.Database("URLService").Collection("Shortened")

	sm := mux.NewRouter()
	sm.HandleFunc("/{id:[a-zA-Z0-9]+}", Expander).Methods(http.MethodGet)
	sm.HandleFunc("/", Shortener).Methods(http.MethodPost)

	s := http.Server{
		Addr:         "127.0.0.1:8080", // configure the bind address
		Handler:      sm,               // set the default handler
		ReadTimeout:  5 * time.Second,  // max time to read request from the client
		WriteTimeout: 10 * time.Second, // max time to write response to the client
	}
	s.ListenAndServe()
}
