package server

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

var (
	allUser    = make(map[net.Conn]string)
	connection = make(chan net.Conn)
	port       = "8989"
	name       = ""
)

func StartServer() {
	portChecker(os.Args)

	listener, err := net.Listen("tcp", "localhost:"+port)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer listener.Close()

	fmt.Println("Listening on the port :" + port)

	chatCache, err := os.Create("internal/files/chat.txt")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer chatCache.Close()

	users, err := os.Create("internal/files/users.txt")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer users.Close()

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil || len(allUser) > 10 {
				conn.Write([]byte("Access to this chat is closed"))
				conn.Close()
				return
			}
			connection <- conn
		}
	}()

	for {
		go userLogin(<-connection, chatCache, users)
	}
}

func userLogin(conn net.Conn, chatCache *os.File, users *os.File) {
	writeWelcomeText(conn)
	authorization(conn, users)
	showAllUsers(conn, users)
	showChatCache(conn, chatCache)

	go Chat(conn, chatCache, users)
}

func userText(conn net.Conn) string {
	return fmt.Sprintf("\r[%s][%s]: ", time.Now().Format("01-02-2006 15:04:05"), allUser[conn])
}

func Chat(conn net.Conn, chatCache *os.File, users *os.File) {
	for {
		conn.Write([]byte(userText(conn)))
		message, err := bufio.NewReader(conn).ReadString('\n')
		var msg string
		if len(message) == 1 {
			continue
		}
		if message[:len(message)-1] == "!changename" {
			textMessage := "User change the name: " + allUser[conn] + " -> "
			authorization(conn, users)
			textMessage += allUser[conn] + "\n"

			for item := range allUser {
				if item == conn {
					continue
				}
				msg = clear(userText(item)) + textMessage

				item.Write([]byte(msg))
				time.Sleep(time.Second / 1000)
				item.Write([]byte(userText(item)))
			}
			if _, err := chatCache.WriteString(msg); err != nil {
				log.Fatal()
			}
			continue
		}

		if err != nil {
			w := allUser[conn]
			delete(allUser, conn)
			textMessage := "\r" + w + " logged out of the chat...\n"
			for item := range allUser {
				msg := clear(userText(item)) + textMessage + userText(item)
				item.Write([]byte(msg))
			}
			if _, err := chatCache.WriteString(textMessage); err != nil {
				log.Fatal()
			}
			return
		}

		textMessage := fmt.Sprintf("%s%s", userText(conn), string(message))

		for item := range allUser {
			if item == conn {
				continue
			}
			msg = clear(userText(item)) + textMessage

			item.Write([]byte(msg))
			time.Sleep(time.Second / 1000)
			item.Write([]byte(userText(item)))
		}

		if _, err := chatCache.WriteString(textMessage); err != nil {
			log.Fatal()
		}
	}
}

func clear(a string) string {
	return "\r" + strings.Repeat(" ", len(a)) + "\r"
}

func portChecker(args []string) {
	if len(args) > 2 {
		fmt.Println("[USAGE]: ./TCPChat $port")
		os.Exit(1)
	}

	if len(args) == 2 {
		_, err := strconv.Atoi(args[1])
		if err != nil {
			fmt.Println("Incorrect port")
			os.Exit(1)
		}
		port = args[1]
	}
}

func writeWelcomeText(conn net.Conn) {
	content, err := ioutil.ReadFile("internal/files/welcome.txt")
	conn.Write([]byte("Welcome to TCP-Chat!\n"))

	if err == nil {
		conn.Write(content)
	}
}

func authorization(conn net.Conn, users *os.File) {
	for {
		conn.Write([]byte("[ENTER YOUR NAME]: "))
		name, _ = bufio.NewReader(conn).ReadString('\n')
		name = name[:len(name)-1]
		flag := false
		for item := range allUser {
			if allUser[item] == name {
				flag = true
				break
			}
		}
		if flag {
			conn.Write([]byte("User with this name already exists\n"))
			continue
		}
		if len(name) == 0 || len(name) > 20 {
			conn.Write([]byte("Incorrect format for the name(0-20)\n\n"))
			continue
		}
		break
	}
	allUser[conn] = name
	users.WriteString(name + "\n")
}

func showAllUsers(conn net.Conn, users *os.File) {
	con, err := ioutil.ReadFile(users.Name())
	if err != nil {
		return
	} else {
		conn.Write([]byte("ALL USERS:\n"))
		conn.Write([]byte("--------------\n"))
		conn.Write(con)
		conn.Write([]byte("--------------\n"))
	}
}

func showChatCache(conn net.Conn, chatCache *os.File) {
	cache, err := ioutil.ReadFile(chatCache.Name())
	if err != nil {
		log.Fatal()
	}

	conn.Write(cache)

	textMessage := "\r" + allUser[conn] + " has joined our chat...\n"

	for item := range allUser {
		msg := clear(userText(item)) + textMessage + userText(item)
		item.Write([]byte(msg))
	}

	if _, err := chatCache.WriteString(textMessage); err != nil {
		os.Exit(1)
	}
}
