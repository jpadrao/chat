package mock

// import "github.com/jpadrao/chat/"

type InternalMessage struct {
	Command string
	Content interface{}
}

type InternalTextMessage struct {
	Username string
	Text     string
}
