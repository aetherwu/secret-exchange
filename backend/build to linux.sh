targetDir="bin"

buildPkg="question"

buildResult=""

build() {
    echo "开始编译"
    buildResult=`CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o "${targetDir}/$buildPkg" 2>&1`

    if [ -z "$buildResult" ]; then
      buildResult="success"
    fi
}

build

if [ "$buildResult" = "success" ]; then
    echo "编译成功 ${targetDir}/$buildPkg"
else
    echo "编译失败：$buildResult"
    exit
fi