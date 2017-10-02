package main
// learning from Andrew Gerrand's "Go: code that grows with grace" talk
import (
	"fmt"
	"html/template"
	"io"
	"log"
	"net"
	"net/http"
	"time"

	"golang.org/x/net/websocket"
)

const LISTEN_ADDR = "localhost:4000"

func main() {
	go netListen()
	http.HandleFunc("/", rootHandler)
	http.Handle("/socket", websocket.Handler(socketHandler))

	err := http.ListenAndServe(LISTEN_ADDR, nil)

	if err != nil {
		log.Fatal(err)
	}
}

//tcp version
func netListen() {
	l, err := net.Listen("tcp", "localhost:4001")
	if err != nil {
		log.Fatal(err)
	}

	for {
		c, err := l.Accept()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Fprint(c, "Waiting for a friend ...")
		go match(c)
	}
}

type socket struct {
	io.ReadWriter
	done chan bool
}

func (s socket) Close() error {
	s.done <- true
	return nil
}

func socketHandler(ws *websocket.Conn) {
	s := socket{ws, make(chan bool)}
	fmt.Fprint(s, "Waiting for a friend ...")
	go match(s)
	<-s.done
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	rootTemplate.Execute(w, LISTEN_ADDR)
}

var rootTemplate = template.Must(template.New("root").Parse(`
<!DOCTYPE html>
<html>
<head></head>
<body>
<h1 style="font-family: Arial, Helvetica, sans-serif; margin: 10px; padding: 8px; color: #008CBA;">Goruupu Chat</h1>
<span style="font-family: Arial, Helvetica, sans-serif; margin: 5px; padding: 8px;">What's your name? </span>
<input style="margin: 5px; padding: 8px; width: 20%;" type="text" id="myname" />
<input style="margin: 5px; padding: 8px; width: 80%;" type="text" id="input" />
<input style="background-color: #008CBA; color:white; border: none; padding: 10px 20px; text-align: center;
	text-decoration: none; display: inline-block; font-size: 12px;" type="button" id="send" value="Send" />
<div style="font-family: Arial, Helvetica, sans-serif; margin: 10px; padding: 8px;" id="output"></div>
<script language="javascript" type="text/javascript">
var wsUri = "ws://{{.}}/socket";
var output;
var input;
var send;
function init() {
  name = document.getElementById("myname");
  output = document.getElementById("output");
  input = document.getElementById("input");
  send = document.getElementById("send");
  send.onclick = sendClickHandler;
  input.onkeydown = function(event) { if (event.keyCode == 13) send.click(); };
  testWebSocket();
}
function sendClickHandler() {
  doSend(input.value);
  input.value = '';
}
function testWebSocket() {
  websocket = new WebSocket(wsUri);
  websocket.onopen = function(evt) { onOpen(evt) };
  websocket.onclose = function(evt) { onClose(evt) };
  websocket.onmessage = function(evt) { onMessage(evt) };
  websocket.onerror = function(evt) { onError(evt) };
}
function onOpen(evt) {
  writeToScreen("--Online--");
}
function onClose(evt) {
  writeToScreen("--Offline--");
}
function onMessage(evt) {
  writeToScreen('<div style="color: #008CBA;">' + evt.data+ '</div>');
}
function onError(evt) {
  writeToScreen('<span style="color: red;">ERROR:</span> ' + evt.data);
}
function doSend(message) {
  if (myname.value==="") { var name = "anon"; } else { var name = myname.value; }
  writeToScreen('<div style="color: black;">' + name + ': ' + message + '</div>');
  websocket.send(name + ': ' + message);
}
function writeToScreen(message) {
  var pre = document.createElement("p");
  pre.style.wordWrap = "break-word";
  pre.innerHTML = message;
  output.appendChild(pre);
}
window.addEventListener("load", init, false);
</script>
</body>
</html>
  `))

var partner = make(chan io.ReadWriteCloser)

func match(c io.ReadWriteCloser) {
	select {
	case partner <- c:
		// handled by another goroutine
	case p := <-partner:
		chat(p, c)
	case <-time.After(5 * time.Second):
		fmt.Fprint(c, "Still waiting ...")
		match(c)
	}
}

func chat(a, b io.ReadWriteCloser) {
	fmt.Fprintln(a, "Someone's here!")
	fmt.Fprintln(b, "Someone's here!")

	errc := make(chan error, 1)

	go copy(a, b, errc)
	go copy(b, a, errc)

	if err := <-errc; err != nil {
		log.Println(err)
	}

	a.Close()
	b.Close()
}

func copy(w io.Writer, r io.Reader, errc chan<- error) {
	_, err := io.Copy(w, r)
	errc <- err
}
