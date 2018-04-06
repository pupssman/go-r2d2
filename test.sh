echo Basic test

MSG=`cat test.wav | ./go-r2d2`

echo "Test msg is |${MSG}|, original msg was |hello, world|"

if [ "$MSG" = "hello, world" ]; then
    exit 0
else
    exit 1
fi
