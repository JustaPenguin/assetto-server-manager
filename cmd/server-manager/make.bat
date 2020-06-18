@ECHO OFF

set GO111MODULE=on
set VERSION=dev

IF [%1]==[] (
    echo "Usage: %0 {clean|run|assets|asset-embed}"
    GOTO END
)
IF %1==clean (
	CALL :clean
    GOTO END
)

IF %1==run (
	CALL :run
    GOTO END
)
IF %1==assets (
	CALL :assets
    GOTO END
)
IF %1==asset-embed (
	CALL :asset-embed
    GOTO END
)

echo "Usage: %0 {clean|run|assets|asset-embed}"


:END
EXIT /B %ERRORLEVEL%

:clean
	DEL /F/Q/S .\server-manager server-manager.exe
	DEL /F/Q/S .\static\manager.js
	DEL /F/Q/S build/*.*
	DEL /F/Q/S .\rsrc.syso
	DEL /F/Q/S .\views\static_embed.go
	DEL /F/Q/S .\static\static_embed.go
	DEL /F/Q/S .\typescript\.gulp-cache
EXIT /B 0

:run
	go build -ldflags "-s -w -X github.com/JustaPenguin/assetto-server-manager.BuildVersion=%VERSION% %LDFLAGS%"
	set FILESYSTEM_HTML=true
	set DEBUG=true
	.\server-manager
EXIT /B 0

:assets
	cd typescript
	Call .\make.bat build
	cd ..
EXIT /B 0

:asset-embed
	go get -u github.com/mjibson/esc
	go generate ./...
EXIT /B 0
