package messages

import (
	"fmt"
)

// Variables

var Msg01 string
var Msg02 string
var Msg03 string

// Functions

func init() {

	// Prepare messages with correct newline symbol.

	Msg01 = fmt.Sprintf("Date: Mon, 7 Feb 1994 21:52:25 -0800 (PST)\r\nFrom: Fred Foobar <foobar@Blurdybloop.COM>\r\nSubject: afternoon meeting\r\nTo: mooch@owatagu.siam.edu\r\nMessage-Id: <B27397-0100000@Blurdybloop.COM>\r\nMIME-Version: 1.0\r\nContent-Type: TEXT/PLAIN; CHARSET=US-ASCII\r\n\r\nHello Joe, do you think we can meet at 3:30 tomorrow?\r\n")

	Msg02 = fmt.Sprintf("From: John Doe <jdoe@machine.example>\r\nTo: Mary Smith <mary@example.net>\r\nSubject: Saying Hello\r\nDate: Fri, 21 Nov 1997 09:55:06 -0600\r\nMessage-ID: <1234@local.machine.example>\r\n\r\nThis is a message just to say hello.\r\nSo, \"Hello\".\r\n")

	Msg03 = fmt.Sprintf("From: Mary Smith <mary@example.net>\r\nTo: John Doe <jdoe@machine.example>\r\nReply-To: \"Mary Smith: Personal Account\" <smith@home.example>\r\nSubject: Re: Saying Hello\r\nDate: Fri, 21 Nov 1997 10:01:10 -0600\r\nMessage-ID: <3456@example.net>\r\nIn-Reply-To: <1234@local.machine.example>\r\nReferences: <1234@local.machine.example>\r\n\r\nThis is a reply to your hello.\r\n")
}
