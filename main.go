package main

import (
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
)

type MyURL struct {
	URL string `json="url"`
}

var URLDict = make(map[string]string)

var alphabet = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

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
		_, ok := URLDict[b62val] // If the shortened URL already exists i.e collision, get new URL
		if ok {
			fmt.Printf("%v already exists in the database.\n", b62val)
			b62val = handleCollision(rand.Intn(62), b62val)
		} else {
			break
		}
	}
	URLDict[b62val] = p.URL
	fmt.Println("Shortened", b62val, URLDict[b62val])
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
	longURL, ok := URLDict[id]
	if !ok {
		w.Write([]byte("Shortened URL incorrect."))
		return
	}
	fmt.Println("longURL :", longURL)
	http.Redirect(w, r, longURL, http.StatusSeeOther)
}

func main() {
	sm := mux.NewRouter()
	sm.HandleFunc("/{id:[a-zA-Z0-9]+}", Expander).Methods(http.MethodGet)
	sm.HandleFunc("/", Shortener).Methods(http.MethodPost)

	s := http.Server{
		Addr:         ":8080",          // configure the bind address
		Handler:      sm,               // set the default handler
		ReadTimeout:  5 * time.Second,  // max time to read request from the client
		WriteTimeout: 10 * time.Second, // max time to write response to the client
	}
	s.ListenAndServe()
}
