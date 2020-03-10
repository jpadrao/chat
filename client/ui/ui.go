package ui

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jpadrao/chat/client/mock"
	"github.com/marcusolsson/tui-go"
)

type internalMessage struct {
	command string
	content string
}

type post struct {
	username string
	message  string
	time     string
}

type chatWindow struct {
	history       *tui.Box
	historyScroll *tui.ScrollArea
	historyBox    *tui.Box
}

func loginUI(inboundChannel chan mock.InternalMessage, outboundChannel chan mock.InternalMessage) tui.UI {

	root := tui.NewVBox()
	ui, err := tui.New(root)
	if err != nil {
		log.Fatal(err)
	}

	user := tui.NewEntry()
	user.SetFocused(true)

	form := tui.NewGrid(0, 0)
	form.AppendRow(tui.NewLabel("User"))
	form.AppendRow(user)

	status := tui.NewStatusBar("Ready.")

	login := tui.NewButton("[Login]")
	login.OnActivated(func(b *tui.Button) {

		text := user.Text()

		outboundChannel <- mock.InternalMessage{Command: "login", Content: text}

		msg := <-inboundChannel

		switch msg.Command {

		case "login":

			if msg.Content.(string) == "sucess" {
				log.Println("logged In")
				status.SetText("Logged in.")

				ui.ClearKeybindings()
				ui.SetWidget(chatUI(text, inboundChannel, outboundChannel, ui))

			} else {
				status.SetText("Invalid username")
			}
		}
	})

	// register := tui.NewButton("[Register]")

	buttons := tui.NewHBox(
		tui.NewSpacer(),
		tui.NewPadder(1, 0, login),
		tui.NewSpacer(),
		// tui.NewPadder(1, 0, register),
	)

	window := tui.NewVBox(
		// tui.NewPadder(10, 1, tui.NewLabel(logo)),
		tui.NewPadder(12, 0, tui.NewLabel("Login to Chat")),
		tui.NewPadder(1, 1, form),
		buttons,
	)
	window.SetBorder(true)

	wrapper := tui.NewVBox(
		tui.NewSpacer(),
		window,
		tui.NewSpacer(),
	)
	content := tui.NewHBox(tui.NewSpacer(), wrapper, tui.NewSpacer())

	root.Append(content)
	root.Append(status)

	tui.DefaultFocusChain.Set(user, login)

	ui.SetKeybinding("Up", func() {
		user.SetFocused(true)
		login.SetFocused(false)
	})

	ui.SetKeybinding("Down", func() {
		user.SetFocused(false)
		login.SetFocused(true)
	})

	ui.SetKeybinding("Esc", func() { ui.Quit() })

	return ui

}

func channelReader(inboundChannel chan mock.InternalMessage, outboundChannel chan mock.InternalMessage,
	chat chatWindow, channelsTab *tui.Table, sidebar *tui.Box, ui tui.UI) {

	for {

		msg := <-inboundChannel

		switch msg.Command {
		case "msg":

			ui.Update(func() {
				chat.history.Append(tui.NewHBox(
					tui.NewLabel(time.Now().Format("15:04")),
					tui.NewPadder(1, 0, tui.NewLabel(fmt.Sprintf("<%s>", msg.Content.(mock.InternalTextMessage).Username))),
					tui.NewLabel(msg.Content.(mock.InternalTextMessage).Text),
					tui.NewSpacer(),
				))
			})
			log.Println("done updating UI")

		case "availableRoms":

			avRoms := msg.Content.([]interface{})

			for _, v := range avRoms {
				label := tui.NewLabel(v.(string))
				channelsTab.AppendRow(label)

				channelsTab.OnSelectionChanged(func(tab *tui.Table) {

					rom := avRoms[tab.Selected()]
					log.Println("changed table selection to ", rom)

					len := chat.history.Length()

					for i := 0; i < len; i++ {
						chat.history.Remove(0)
					}
					outboundChannel <- mock.InternalMessage{Command: "changeRom", Content: rom}
					outboundChannel <- mock.InternalMessage{Command: "messages", Content: rom}

				})

				log.Println("added new rom to side bar: ", v)
			}

			ui.Update(func() {

				sidebar.Append(channelsTab)
				sidebar.Append(tui.NewSpacer())
			})

			outboundChannel <- mock.InternalMessage{Command: "messages", Content: avRoms[0].(string)}

		case "messageList":

			msgList := msg.Content.([]interface{})

			for _, v := range msgList {

				time := v.(map[string]interface{})["Time"]
				username := v.(map[string]interface{})["Username"]
				text := v.(map[string]interface{})["Message"]

				if username.(string) != "bot" {
					ui.Update(func() {

						chat.history.Append(tui.NewHBox(
							tui.NewLabel(time.(string)),
							tui.NewPadder(1, 0, tui.NewLabel(fmt.Sprintf("<%s>", username.(string)))),
							tui.NewLabel(text.(string)),
							tui.NewSpacer(),
						))
					})
				}

			}

		default:

			log.Println("UI received unknown message: ", msg.Command)
		}

	}
}

func chatUI(username string, inboundChannel chan mock.InternalMessage, outboundChannel chan mock.InternalMessage, ui tui.UI) *tui.Box {

	sidebar := tui.NewVBox(
		tui.NewLabel("CHANNELS"),
	)

	sidebar.SetBorder(true)

	channelsTab := tui.NewTable(0, 0)

	history := tui.NewVBox()

	historyScroll := tui.NewScrollArea(history)
	historyScroll.SetAutoscrollToBottom(true)

	historyBox := tui.NewVBox(historyScroll)
	historyBox.SetBorder(true)

	chatWindow := chatWindow{history, historyScroll, historyBox}

	input := tui.NewEntry()
	input.SetFocused(true)
	input.SetSizePolicy(tui.Expanding, tui.Maximum)

	inputBox := tui.NewHBox(input)
	inputBox.SetBorder(true)
	inputBox.SetSizePolicy(tui.Expanding, tui.Maximum)

	chat := tui.NewVBox(historyBox, inputBox)
	chat.SetSizePolicy(tui.Expanding, tui.Expanding)

	input.OnSubmit(func(e *tui.Entry) {

		history.Append(tui.NewHBox(
			tui.NewLabel(time.Now().Format("15:04")),
			tui.NewPadder(1, 0, tui.NewLabel(fmt.Sprintf("<%s>", username))),
			tui.NewLabel(e.Text()),
			tui.NewSpacer(),
		))
		outboundChannel <- mock.InternalMessage{Command: "msg", Content: e.Text()}
		input.SetText("")
	})

	root := tui.NewHBox(sidebar, chat)

	ui.SetKeybinding("Left", func() {
		channelsTab.SetFocused(true)
		input.SetFocused(false)
	})

	ui.SetKeybinding("Right", func() {
		channelsTab.SetFocused(false)
		input.SetFocused(true)
	})

	ui.SetKeybinding("Esc", func() { ui.Quit() })

	go channelReader(inboundChannel, outboundChannel, chatWindow, channelsTab, sidebar, ui)

	return root
}

// StartUI initiates the UI
func StartUI(inboundChannel chan mock.InternalMessage, outboundChannel chan mock.InternalMessage) {

	f, err := os.Create("debug.log")
	if err != nil {
		fmt.Println(err)
	}
	defer f.Close()

	logger := log.New(f, "", log.LstdFlags)
	tui.SetLogger(logger)

	ui := loginUI(inboundChannel, outboundChannel)

	if err := ui.Run(); err != nil {
		log.Fatal(err)
	}
}
