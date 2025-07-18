package dao

import (
	"github.com/daodao97/xgo/xdb"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/mattn/go-sqlite3"
)

var MessageModel xdb.Model
var ConversationModel xdb.Model

func Init() {
	ConversationModel = xdb.New(
		"companion_conversation",
	)

	MessageModel = xdb.New(
		"companion_message",
	)
}
