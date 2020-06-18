@ECHO OFF


IF [%1]==[] (
    echo "Usage: %0 {deps|build|watch}"
    GOTO END
)

IF %1==deps (
	CALL :deps
    GOTO END
)
IF %1==build (
	CALL :deps
	CALL :build
    GOTO END
)

IF %1==watch (
	CALL :deps
	CALL :watch
    GOTO END
)

echo "Usage: %0 {deps|build|watch}"
	


:END
EXIT /B %ERRORLEVEL%


:deps
	Call npm install
EXIT /B 0

:watch
  Call ./node_modules/.bin/gulp
EXIT /B 0

:build
  Call ./node_modules/.bin/gulp build
EXIT /B 0