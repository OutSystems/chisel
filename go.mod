module github.com/jpillora/chisel

go 1.19

require (
	github.com/armon/go-socks5 v0.0.0-20160902184237-e75332964ef5
	github.com/fsnotify/fsnotify v1.6.0
	github.com/gorilla/websocket v1.4.2
	github.com/jpillora/backoff v1.0.0
	github.com/jpillora/requestlog v1.0.0
	github.com/jpillora/sizestr v1.0.0
	golang.org/x/crypto v0.0.0-20221005025214-4161e89ecf1b
	golang.org/x/net v0.0.0-20221004154528-8021a29435af
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
)

require (
	github.com/andrew-d/go-termutil v0.0.0-20150726205930-009166a695a2 // indirect
	github.com/jpillora/ansi v1.0.2 // indirect
	github.com/tomasen/realip v0.0.0-20180522021738-f0c99a92ddce // indirect
	golang.org/x/sys v0.0.0-20220908164124-27713097b956 // indirect
	golang.org/x/text v0.3.7 // indirect
)

replace github.com/jpillora/chisel => ../chisel
