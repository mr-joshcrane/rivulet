package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type RivuletEvent struct {
	Detail struct {
		Message string `json:"message"`
	} `json:"detail"`
}

func main() {
	handler := func(w http.ResponseWriter, r *http.Request) {
		data, err := io.ReadAll(r.Body)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer r.Body.Close()
		var event RivuletEvent
		err = json.Unmarshal(data, &event)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println(event.Detail.Message)

	}
	http.HandleFunc("/", handler)
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println(err)
	}
}
