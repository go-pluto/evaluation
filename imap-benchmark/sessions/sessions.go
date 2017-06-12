package sessions

import (
    "fmt"
	"math/rand"
    "../utils"
	"strconv"
)

// Structs

type IMAPcommand struct {
	Command        string
	Arguments      []string
}

type Folder struct {
    Foldername  string
    Messages    []Message
}

type Message struct {
    Flags      []string
}

// Functions

func removeDeleted(folder *Folder){
	for j := 0; j < len(folder.Messages); j++ {
		for k := 0; k < len(folder.Messages[j].Flags); k++ {
			if folder.Messages[j].Flags[k] == "\\Deleted"{
				folder.Messages = append(folder.Messages[:j], folder.Messages[j+1:]...)
				j = j - 1
				break
			}
		}
	}
}

// TODO: proper documentation
func GenerateSession(length int) []IMAPcommand {

	//fmt.Printf("Session begin\n")

	var commands []IMAPcommand

	// Generate empty mailbox
	var folders []Folder

	selected := -1

	// define the session length
	sessionLength := rand.Intn(length)

	// generate the session content
	for i := 0; i < sessionLength; i++ {
		// Initialize command argument array
		var arguments []string

		if selected != -1 {
			// Equals IMAPs "SELECTED" state. Possible IMAP commands are:
			// STORE, EXPUNGE, CLOSE

			if len(folders[selected].Messages) == 0 {
				// In case no message is present in the folder, only EXPUNGE and
				// CLOSE are possible commands.

				r := rand.Float64()
				switch {
				case 0.0 <= r && r < 0.2:
					commands = append(commands, IMAPcommand{Command: "EXPUNGE", Arguments: arguments})
				case 0.2 <= r && r < 1.0:
					commands = append(commands, IMAPcommand{Command: "CLOSE"})
					selected = -1
				}
			} else {
				// In SELECTED state and messages are present in the selected
				// folder. Possible IMAP commands are:
				// STORE, EXPUNGE, CLOSE

				r := rand.Float64()
				switch {
				case 0.0 <= r && r < 0.6:
					// select the message
					msgIndex := rand.Intn(len(folders[selected].Messages))
					arguments = append(arguments, strconv.Itoa(msgIndex + 1))

					flagstring, flags := utils.GenerateFlags()
					arguments = append(arguments, flagstring)

					folders[selected].Messages[msgIndex].Flags = flags

					commands = append(commands, IMAPcommand{Command: "STORE", Arguments: arguments})

				case 0.6 <= r && r < 0.8:
					removeDeleted(&folders[selected])
					commands = append(commands, IMAPcommand{Command: "EXPUNGE"})

				case 0.8 <= r && r < 1.0:
					removeDeleted(&folders[selected])
					commands = append(commands, IMAPcommand{Command: "CLOSE"})
					selected = -1
				}
			}
		} else {
			// Equals IMAPs "Authenticated" state. No folder is selected.
			// Possible IMAP commands are:
			// CREATE, DELETE, APPEND, SELECT

			if len(folders) == 0 {
				// If no folders are present in the mailbox, the only possible
				// IMAP command is CREATE.

				var messages []Message
				initFoldername := utils.GenerateString(8)
				initFolder := Folder{Foldername: initFoldername, Messages: messages}
				folders = append(folders, initFolder)
				arguments = append(arguments, initFoldername)
				commands = append(commands, IMAPcommand{Command: "CREATE", Arguments: arguments})

			} else {
				// Otherwise all of the above mentioned IMAP commands in
				// Authenticated state are possible.

				r := rand.Float64()
				switch {
				case 0.0 <= r && r < 0.15:

					initFoldername := utils.GenerateString(8)
					// Rerandom in case the generated foldername already exists in this session
					for j := 0; j < len(folders); j++ {
						if initFoldername == folders[j].Foldername {
							initFoldername = utils.GenerateString(8)
							j = -1
						}
					}
					var messages []Message
					initFolder := Folder{Foldername: initFoldername, Messages: messages}
					folders = append(folders, initFolder)
					arguments = append(arguments, initFoldername)
					commands = append(commands, IMAPcommand{Command: "CREATE", Arguments: arguments})

				case 0.15 <= r && r < 0.3:
					folderIndex := rand.Intn(len(folders))
					foldername := folders[folderIndex].Foldername
					folders = append(folders[:folderIndex], folders[folderIndex + 1:]...)
					arguments = append(arguments, foldername)
					commands = append(commands, IMAPcommand{Command: "DELETE", Arguments: arguments})

				case 0.3 <= r && r < 0.9:
					// choose the folder
					folderIndex := rand.Intn(len(folders))

					// lookup the foldername and add it to the arguments list
					foldername := folders[folderIndex].Foldername
					arguments = append(arguments, foldername)

					// generate the flags of the message
					flagstring, flags := utils.GenerateFlags()
					arguments = append(arguments, flagstring)

					//generate the date/time string - OPTIONAL
					arguments = append(arguments, "{310}")

					// generate the message
					// TODO: replace with a proper message generator
					var msg string
					msg = fmt.Sprintf("Date: Mon, 7 Feb 1994 21:52:25 -0800 (PST)\r\nFrom: Fred Foobar <foobar@Blurdybloop.COM>\r\nSubject: afternoon meeting\r\nTo: mooch@owatagu.siam.edu\r\nMessage-Id: <B27397-0100000@Blurdybloop.COM>\r\nMIME-Version: 1.0\r\nContent-Type: TEXT/PLAIN; CHARSET=US-ASCII\r\n\r\nHello Joe, do you think we can meet at 3:30 tomorrow?\r\n")

					arguments = append(arguments, msg)

					folders[folderIndex].Messages = append(folders[folderIndex].Messages, Message{Flags: flags})
					commands = append(commands, IMAPcommand{Command: "APPEND", Arguments: arguments})

				case 0.9 <= r && r < 1.0:
					folderIndex := rand.Intn(len(folders))
					foldername := folders[folderIndex].Foldername
					arguments = append(arguments, foldername)
					commands = append(commands, IMAPcommand{Command: "SELECT", Arguments: arguments})
					selected = folderIndex
				}
			}
		}
	}

	// Exit the Selected state if the last command was SELECT
	if selected != -1 {
		commands = append(commands, IMAPcommand{Command: "CLOSE"})
		selected = -1
	}

	// Finish the session by deleting all created folders.
	for i := 0; i < len(folders); i++ {
		var arguments []string
		arguments = append(arguments, folders[i].Foldername)
		commands = append(commands, IMAPcommand{Command: "DELETE", Arguments: arguments})
	}

	// TODO: delete me
	// for i := 0; i < len(commands); i++ {
	// 	fmt.Printf("Command: %s", commands[i].Command)
	// 	for j := 0; j < len(commands[i].Arguments); j++ {
	// 		fmt.Printf(" %s", commands[i].Arguments[j])
	// 	}
	// 	fmt.Printf("\n")
	// }
	// fmt.Printf("Session end\n")

	// return IMAP Commands
	// jobs <- j
	return commands
}
