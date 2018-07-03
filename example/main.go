package main

import (
	"net/http"

	"github.com/ramonberrutti/steam_go"
)

func loginHandler(w http.ResponseWriter, r *http.Request) {
	opID := steam_go.NewOpenID(r)
	switch opID.Mode() {
	case "":
		http.Redirect(w, r, opID.AuthUrl(), 301)
	case "cancel":
		w.Write([]byte("Authorization cancelled"))
	default:
		steamID, err := opID.ValidateAndGetID()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		// Do whatever you want with steam id
		w.Write([]byte(steamID))
	}
}

func main() {
	http.HandleFunc("/login", loginHandler)
	http.ListenAndServe(":8081", nil)
}
