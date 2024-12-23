chcp 65001
@echo off
@REM go run main.go -p 0 -tll 40 -tl 500 -httping -cfcolo LAX,HKG,NRT,ICN,DXB,SIN,NYC,SFO,KIX,BOM,DEL,JKT,KUL,TPE -dd
go run main.go -p 0 -tll 40 -tl 300 -httping -cfcolo * -dd
git add .
git commit -m "自动提交生成的代码"
git push
exit