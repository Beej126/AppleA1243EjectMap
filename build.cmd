cls

taskkill /im AppleA1243EjectMap.exe /f

:: syso files come from go generate
rm *.syso
rm *.exe

:: go generate reads the go:generate directives in main.go, which for example generates resources corresponding to embedded exe icon
go generate ./...

go build -ldflags="-s -w -H windowsgui" -o AppleA1243EjectMap.exe .
AppleA1243EjectMap.exe lock