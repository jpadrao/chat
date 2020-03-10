package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"sync"
	"time"
)

type client struct {
	id      string
	encoder json.Encoder
}

type post struct {
	Username string
	Message  string
	Time     string
}

type rom struct {
	name          string
	mutex         sync.Mutex
	activeClients map[string]client
	messages      []post
}

type romMap struct {
	mutex sync.Mutex
	roms  map[string]*rom
}

func (rm *rom) addClient(cl client) bool {

	// fmt.Println(rm)

	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	_, ok := rm.activeClients[cl.id]

	if !ok {
		rm.activeClients[cl.id] = cl
	}

	return !ok
}

func (rm *rom) removeClient(id string) {

	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	delete(rm.activeClients, id)
}

func (rm *rom) broadcast(id string, text string) {

	toSend := make(map[string]interface{})

	toSend["Command"] = "msg"
	toSend["user"] = id
	toSend["text"] = text

	fmt.Println("braodcasting msg: ", text)

	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	rm.messages = append(rm.messages, post{id, text, time.Now().Format("15:04")})

	for k, v := range rm.activeClients {
		if k != id {

			if err := v.encoder.Encode(&toSend); err != nil {
				log.Println(err)
			}
		}
	}
}

func (rm *rom) sendToUser(target string, command string, text string) {

	toSend := make(map[string]interface{})

	toSend["Command"] = command
	toSend["user"] = "bot"
	toSend["text"] = text

	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	user := rm.activeClients[target]

	if err := user.encoder.Encode(&toSend); err != nil {
		log.Println(err)
	}
}

func (rm *rom) getMessages(target string) {

	toSend := make(map[string]interface{})
	toSend["Command"] = "messageList"

	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	toSend["list"] = rm.messages

	user := rm.activeClients[target]

	if err := user.encoder.Encode(&toSend); err != nil {
		log.Println(err)
	}
}

// -------------------------------------------------------------------------------------------

func (rmap *romMap) addRom(rm *rom) {

	rmap.mutex.Lock()
	defer rmap.mutex.Unlock()

	_, ok := rmap.roms[rm.name]

	if !ok {

		rmap.roms[rm.name] = rm
	}

}

func (rmap *romMap) changeRom(cl client, prevRom *rom, newRom string) *rom {

	rmap.mutex.Lock()
	defer rmap.mutex.Unlock()

	rm, ok := rmap.roms[newRom]

	if ok {

		prevRom.removeClient(cl.id)
		rm.addClient(cl)
		rm.broadcast("bot", " user "+cl.id+" has joined the rom")
		rm.sendToUser(cl.id, "bot", "welcome to rom "+newRom)

		fmt.Println("changed user to other rom")
	}

	return rm

}

// -------------------------------------------------------------------------------------------

func handleConnection(c net.Conn, rmap *romMap) {

	fmt.Printf("server listening to %s\n", c.RemoteAddr())

	dec := json.NewDecoder(c)
	enc := json.NewEncoder(c)

	loggedIn := false
	id := ""
	cl := client{}

	rm := rmap.roms["main"]

	for {

		var msg map[string]interface{}

		if err := dec.Decode(&msg); err != nil {
			log.Println(err)
			rm.removeClient(id)
			return
		}

		switch msg["Command"] {

		case "login":

			username := msg["username"].(string)
			cl = client{username, *enc}

			log.Println("Login message user: ", username)

			loggedIn = rm.addClient(cl)
			id = cl.id

			if loggedIn {

				fmt.Println("user ", username, " logged in")

				rm.sendToUser(cl.id, "login", "sucess")

				roms := make([]string, len(rmap.roms))
				i := 0
				for k := range rmap.roms {
					roms[i] = k
					i++
				}

				toSend := make(map[string]interface{})

				toSend["Command"] = "availableRoms"
				toSend["list"] = roms

				if err := enc.Encode(&toSend); err != nil {
					log.Println(err)
				}

				rm.broadcast("bot", " user "+cl.id+" has joined the rom")

			} else {

				toSend := make(map[string]interface{})

				toSend["Command"] = "login"
				toSend["text"] = "fail"

				if err := cl.encoder.Encode(&toSend); err != nil {
					log.Println(err)
				}

				fmt.Println("user ", username, " failed to login")
			}

		case "msg":

			text := msg["text"].(string)

			if loggedIn {
				log.Println("msg[", cl.id, ":", text, "]")
				rm.broadcast(id, text)
			}

		case "changeRom":

			if loggedIn {
				romName := msg["rom"].(string)
				log.Println(cl.id, " change to rom ", romName)
				rm = rmap.changeRom(cl, rm, romName)
			}

		case "messages":

			fmt.Println("get Messages from rom")

			if loggedIn {
				log.Println(cl.id, " get messages from rom ", rm.name)
				rm.getMessages(cl.id)
			}

		}
	}

}

func main() {

	f, err := os.OpenFile("serverLog.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()

	log.SetOutput(f)

	port := "6666"

	if len(os.Args) == 2 {
		port = os.Args[1]
	}

	l, err := net.Listen("tcp", "localhost:"+port)

	fmt.Println("listening on localhost:", port)

	if err != nil {
		fmt.Println(err)
		return
	}

	defer l.Close()

	mainRom := rom{name: "main", activeClients: make(map[string]client)}
	lolRom := rom{name: "lol", activeClients: make(map[string]client)}
	kekRom := rom{name: "kek", activeClients: make(map[string]client)}

	rmap := romMap{roms: make(map[string]*rom)}
	rmap.addRom(&mainRom)
	rmap.addRom(&lolRom)
	rmap.addRom(&kekRom)

	for {
		c, err := l.Accept()
		if err != nil {
			fmt.Println(err)
			return
		}
		go handleConnection(c, &rmap)
	}
}
