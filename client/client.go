package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"strings"

	"github.com/jpadrao/chat/client/mock"
	"github.com/jpadrao/chat/client/ui"
)

type internalMessage struct {
	command string
	content string
}

func fromServer(conn net.Conn, inboundChannel chan mock.InternalMessage) {

	dec := json.NewDecoder(conn)

	for {

		var msg map[string]interface{}

		if err := dec.Decode(&msg); err != nil {
			log.Println(err)
			return
		}

		command := msg["Command"].(string)

		switch command {

		case "login":

			inboundChannel <- mock.InternalMessage{Command: "login", Content: msg["text"]}

		case "msg":

			log.Println("received broadcast msg")

			inboundChannel <- mock.InternalMessage{Command: "msg",
				Content: mock.InternalTextMessage{Username: msg["user"].(string), Text: strings.TrimSuffix(msg["text"].(string), "\n")}}

		case "availableRoms":

			log.Println("received available roms: ", msg["list"].([]interface{}))

			inboundChannel <- mock.InternalMessage{Command: "availableRoms", Content: msg["list"].([]interface{})}

			log.Println("DONE roms")

		case "messageList":

			log.Println("received message list: ", msg["list"])

			inboundChannel <- mock.InternalMessage{Command: "messageList", Content: msg["list"]}
		}
	}
}

func toServer(conn net.Conn, outboundChannel chan mock.InternalMessage) {

	enc := json.NewEncoder(conn)

	for {

		// log.Println("wating for ui input")

		userInput := <-outboundChannel

		switch userInput.Command {

		case "login":

			log.Printf("trying to login: %s", userInput.Content)

			logInMessage := make(map[string]interface{})

			logInMessage["Command"] = "login"
			logInMessage["username"] = userInput.Content

			if err := enc.Encode(&logInMessage); err != nil {
				log.Println(err)
			}

		case "changeRom":

			log.Println("changing rom")

			changeRomMsg := make(map[string]interface{})

			changeRomMsg["Command"] = "changeRom"
			changeRomMsg["rom"] = userInput.Content

			if err := enc.Encode(&changeRomMsg); err != nil {
				log.Println(err)
			}

		case "messages":

			log.Println("getting rom messages")

			getMessagesMsg := make(map[string]interface{})

			getMessagesMsg["Command"] = "messages"
			getMessagesMsg["rom"] = userInput.Content

			if err := enc.Encode(&getMessagesMsg); err != nil {
				log.Println(err)
			}

		default:

			toSend := make(map[string]interface{})

			if userInput.Content != "" {

				log.Println("sending to server ", userInput.Content)

				toSend["Command"] = "msg"
				toSend["text"] = userInput.Content

				if err := enc.Encode(&toSend); err != nil {
					log.Println(err)
				}
			}
		}
	}

}

func connect(port string) (net.Conn, error) {

	log.Println("Connecting to localhost:", port)

	conn, err := net.Dial("tcp", "localhost:"+port)

	if err != nil {
		log.Println(err)
		return nil, err
	}

	return conn, nil

}

func main() {

	f, err := os.OpenFile("clLog.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()

	log.SetOutput(f)

	port := "6666"

	if len(os.Args) == 2 {
		port = os.Args[1]
	}

	conn, err := connect(port)

	if err != nil {
		fmt.Println(err)
		return
	}

	inboundChannel := make(chan mock.InternalMessage)
	outboundChannel := make(chan mock.InternalMessage)

	go toServer(conn, outboundChannel)
	go fromServer(conn, inboundChannel)

	ui.StartUI(inboundChannel, outboundChannel)

}
