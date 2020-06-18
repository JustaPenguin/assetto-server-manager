@ECHO OFF

set GO111MODULE=on
set VERSION=unstable

IF [%1]==[] (
    echo "Usage: %0 {all|clean|generate|assets|asset-embed|build|run}"
    GOTO END
)
IF %1==all (
	CALL :clean
	CALL :assets
	CALL :build
    GOTO END
)
IF %1==clean (
	CALL :clean
    GOTO END
)
IF %1==generate (
	CALL :generate
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
IF %1==build (
	CALL :build
    GOTO END
)
IF %1==run (
	CALL :run
    GOTO END
)

echo "Usage: %0 {all|clean|generate|assets|asset-embed|build|run}"

:END
EXIT /B %ERRORLEVEL%

:clean
	rm -rf changelog_embed.go
	cd .\cmd\server-manager
	Call .\make.bat clean
	cd ..\..
EXIT /B 0

:generate
	go get -u github.com/mjibson/esc
	go generate ./...
EXIT /B 0

:assets
	rm -rf changelog_embed.go
	cd .\cmd\server-manager
	Call .\make.bat assets
	cd ..\..
EXIT /B 0

:asset-embed
	rm -rf changelog_embed.go
	cd .\cmd\server-manager
	Call .\make.bat asset-embed
	cd ..\..
EXIT /B 0

:build
	rm -rf changelog_embed.go
	cd .\cmd\server-manager
	Call .\make.bat build
	cd ..\..
EXIT /B 0

:run
	rm -rf changelog_embed.go
	cd .\cmd\server-manager
	Call .\make.bat run
	cd ..\..
EXIT /B 0