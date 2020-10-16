echo "构建应用程序中"
CGO_ENABLED=0 go build -a -ldflags '-s' ..
echo "构建镜像中"
docker build -t packagewjx/workload-classifier .