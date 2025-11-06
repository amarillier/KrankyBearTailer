# compile, then create an Inno setup installer package

# go build .
# GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC="x86_64-w64-mingw32-gcc" go build -ldflags="-w -s -H windowsgui -r KrankyBearTailer.rc" -trimpath -o bin/WinAMD64/
go-winres make
go build -ldflags="-w -s -H windowsgui -r KrankyBearTailer.rc" -trimpath-o bin/WinAMD64/

Copy-Item bin/WinAMD64/KrankyBearTailer.exe .\KrankyBearTailer.exe

& 'C:\Program Files (x86)\Inno Setup 6\ISCC.exe' .\Inno\KrankyBearTailer.iss

# "Now this is not the end. It is not even the beginning of the end. But it is, perhaps, the end of the beginning." Winston Churchill, November 10, 1942
