package main

import (
	"labix.org/v2/mgo"
	"net/http"
	"fmt"
	"github.com/opesun/chill/frame/top"
	"github.com/opesun/chill/frame/config"
)

func err() {
	if r := recover(); r != nil {
		fmt.Println(r)
		panic(fmt.Sprint(r))
	}
}

func main() {
	defer err()
	fmt.Println("Starting server.")
	config := config.New()
	config.LoadFromFile()
	dial := config.DBAddr
	if len(config.DBUser) != 0 || len(config.DBPass) != 0 {
		if len(config.DBUser) == 0 {
			panic("Database password provided but username is missing.")
		}
		if len(config.DBPass) == 0 {
			panic("Database username is provided but password is missing.")
		}
		dial = config.DBUser + ":" + config.DBPass + "@" + config.DBAddr
		if !config.DBAdmMode {
			dial = dial + "/" + config.DBName
		}
	}
	session, err := mgo.Dial(dial)
	if err != nil {
		panic(err)
	}
	db := session.DB(config.DBName)
	defer session.Close()
	http.HandleFunc("/",
	func(w http.ResponseWriter, req *http.Request) {
		top.New(session, db, w, req, config).Route()
	})
	err = http.ListenAndServe(config.Addr+":"+config.PortNum, nil)
	if err != nil {
		fmt.Println(err)
		return
	}
}
