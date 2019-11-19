module github.com/jbrady42/h2chat

go 1.13

require (
	github.com/cznic/mathutil v0.0.0-20181122101859-297441e03548
	github.com/marcusolsson/tui-go v0.4.0
	github.com/r3labs/sse v0.0.0-20191018093120-b8132ebb4c21
	github.com/remyoudompheng/bigfft v0.0.0-20190728182440-6a916e37a237 // indirect
	golang.org/x/net v0.0.0-20191116160921-f9c825593386
)

replace github.com/r3labs/sse => github.com/jbrady42/sse v1.7.0
