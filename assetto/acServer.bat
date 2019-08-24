@echo off
TITLE AC Server
echo Starting AC Server...
SETLOCAL EnableDelayedExpansion
 for /f "skip=1 tokens=1-6 delims= " %%a in ('wmic path Win32_LocalTime Get Day^,Hour^,Minute^,Month^,Second^,Year /format:table') do (
        IF NOT "%%~f"=="" (
            set /a _d=10000 * %%f + 100 * %%d + %%a
			set /a _t=10000 * %%b + 100 * %%c + %%e
        )
    )

echo Output is logs/session/output%_d%_%_t%.log

acserver.exe > logs/session/output%_d%_%_t%.log 2> logs/error/error%_d%_%_t%.log