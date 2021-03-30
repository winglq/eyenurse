win:
	GOOS=windows go build -ldflags="-H windowsgui" -o eyenurse.exe

linux:
	GOOS=linux go build -o eyenurse
